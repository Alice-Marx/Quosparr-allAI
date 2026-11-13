package opencode

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/licoy/quosparr/internal/netx"
)

// Vendor endpoints. The usage path is a hard-coded string in the CLI, so it
// is the steadier of the two; the budget path is an internal Console route.
const (
	ZenUsageURL = "https://opencode.ai/zen/go/v1/usage"
	BudgetsURL  = "https://opencode.ai/console/api/policies/current-budgets"
)

// The two org headers differ: the usage endpoint takes x-opencode-org-id and
// the budget endpoint takes x-org-id. Neither is required for the usage
// endpoint, but sending it keeps the answer in the console organization.
const (
	usageOrgHeader   = "x-opencode-org-id"
	budgetOrgHeader  = "x-org-id"
	requestTimeout   = 15 * time.Second
	maxResponseBytes = 1 << 20
)

var (
	// ErrUnauthorized is HTTP 401: the stored login is no longer accepted.
	ErrUnauthorized = errors.New("opencode: login not accepted")
	// ErrUnavailable is any other unusable answer.
	ErrUnavailable = errors.New("opencode: endpoint unavailable")
	// ErrDecode is an answer this package cannot read.
	ErrDecode = errors.New("opencode: unexpected response")
	// ErrForbidden is HTTP 403: the organization has that feature off. It is
	// mapped to "no data" by Usage and Budgets and never reaches a caller.
	errForbidden = errors.New("opencode: feature not enabled")
)

// Client fetches the two vendor endpoints. A nil HTTP resolves the current
// netx client for every request, so a proxy change is followed; tests set it
// (and the URLs) to a local server.
type Client struct {
	HTTP       *http.Client
	ZenURL     string
	BudgetsURL string
	Timeout    time.Duration
}

func (c Client) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return requestTimeout
}

func (c Client) http(ctx context.Context) (*http.Client, error) {
	if c.HTTP != nil {
		return c.HTTP, nil
	}
	return netx.ClientFor(ctx, netx.WithoutCookies)
}

func (c Client) usageURL() string {
	if c.ZenURL != "" {
		return c.ZenURL
	}
	return ZenUsageURL
}

func (c Client) budgetsURL() string {
	if c.BudgetsURL != "" {
		return c.BudgetsURL
	}
	return BudgetsURL
}

// Usage fetches the Zen/Go windows. A 403 is reported as ok=false without an
// error, because an organization without the Go usage page is not a failure.
func (c Client) Usage(ctx context.Context, cred Credential) (Usage, bool, error) {
	body, err := c.get(ctx, c.usageURL(), usageOrgHeader, cred)
	if err != nil {
		if errors.Is(err, errForbidden) {
			return Usage{}, false, nil
		}
		return Usage{}, false, err
	}
	usage, err := decodeUsage(body)
	if err != nil {
		return Usage{}, false, err
	}
	return usage, true, nil
}

// Budgets fetches the Console budget rules. A 403 is reported as ok=false
// without an error; an empty list is a normal answer with ok=true.
func (c Client) Budgets(ctx context.Context, cred Credential) ([]Budget, bool, error) {
	body, err := c.get(ctx, c.budgetsURL(), budgetOrgHeader, cred)
	if err != nil {
		if errors.Is(err, errForbidden) {
			return nil, false, nil
		}
		return nil, false, err
	}
	budgets, err := decodeBudgets(body)
	if err != nil {
		return nil, false, err
	}
	return budgets, true, nil
}

// get performs one authenticated GET and returns its body. The error never
// carries a header or the token: the netx error has the host only, and every
// status failure maps to a sentinel.
func (c Client) get(ctx context.Context, url, orgHeader string, cred Credential) ([]byte, error) {
	if !cred.Usable(time.Now()) {
		return nil, ErrUnauthorized
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()
	client, err := c.http(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, ErrUnavailable
	}
	req.Header.Set("Accept", "application/json")
	if token := cred.token(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if cred.HasOrg() {
		req.Header.Set(orgHeader, cred.OrgID)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusForbidden:
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		return nil, errForbidden
	case resp.StatusCode == http.StatusUnauthorized:
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		return nil, ErrUnauthorized
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		return nil, ErrUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, ErrUnavailable
	}
	if len(body) > maxResponseBytes {
		return nil, ErrDecode
	}
	return body, nil
}
