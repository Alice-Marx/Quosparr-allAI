package glm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/licoy/quosparr/internal/netx"
)

const (
	// BigModelQuotaURL is the domestic monitor endpoint.
	BigModelQuotaURL = "https://bigmodel.cn/api/monitor/usage/quota/limit"
	// ZaiQuotaURL is the international monitor endpoint: same path, other
	// host, other accounts.
	ZaiQuotaURL = "https://api.z.ai/api/monitor/usage/quota/limit"
	// requestTimeout matches the ZCode CLI.
	requestTimeout = 15 * time.Second
	// maxResponseBytes bounds the body of a small JSON envelope.
	maxResponseBytes = 1 << 20
)

// quotaURLEnv are the endpoint overrides of the ZCode CLI, in its order.
var quotaURLEnv = []string{"ZCODE_BIGMODEL_USAGE_QUOTA_URL", "BIGMODEL_USAGE_QUOTA_URL"}

// Client reads the quota endpoint. A nil HTTP resolves the current netx
// context for every request, so it follows later proxy changes; tests set
// HTTP and QuotaURL to an httptest server.
type Client struct {
	HTTP *http.Client
	// QuotaURL overrides the endpoint of every region.
	QuotaURL string
	// Now is the clock of the reading; nil is time.Now.
	Now func() time.Time
}

func (c Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c Client) http(ctx context.Context) (*http.Client, error) {
	if c.HTTP != nil {
		return c.HTTP, nil
	}
	return netx.ClientFor(ctx, netx.WithoutCookies)
}

// URLFor resolves the endpoint of a region. The environment override only
// applies to the domestic host, as in the ZCode CLI.
func (c Client) URLFor(region Region) string {
	if c.QuotaURL != "" {
		return c.QuotaURL
	}
	if region == BigModel {
		for _, name := range quotaURLEnv {
			if value := strings.TrimSpace(os.Getenv(name)); value != "" {
				return value
			}
		}
		return BigModelQuotaURL
	}
	return ZaiQuotaURL
}

// Fetch reads the quota of one candidate. No error carries the credential.
func (c Client) Fetch(ctx context.Context, cand Candidate) (Quota, error) {
	if cand.Key == "" {
		return Quota{}, ErrNoCredentials
	}
	endpoint, err := c.endpointFor(cand)
	if err != nil {
		return Quota{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Quota{}, ErrEndpoint
	}
	// The monitor API takes the credential verbatim: a "Bearer " prefix is
	// rejected exactly like an expired token.
	req.Header.Set("authorization", cand.Key)
	req.Header.Set("accept", "application/json")
	if cand.Team {
		if cand.Organization != "" {
			req.Header.Set("bigmodel-organization", cand.Organization)
		}
		if cand.Project != "" {
			req.Header.Set("bigmodel-project", cand.Project)
		}
	}
	client, err := c.http(ctx)
	if err != nil {
		return Quota{}, fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return Quota{}, fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		return Quota{}, fmt.Errorf("%w: %d", ErrBadResponse, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return Quota{}, fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	if len(body) > maxResponseBytes {
		return Quota{}, ErrDecode
	}
	quota, err := DecodeQuota(body, c.now())
	if err != nil {
		return Quota{}, err
	}
	quota.Endpoint = endpoint
	return quota, nil
}

// FetchFirst tries every candidate in order and returns the reading the
// endpoint accepted. The last error wins when every credential is rejected,
// as in the ZCode CLI.
func (c Client) FetchFirst(ctx context.Context, candidates []Candidate) (Quota, Candidate, error) {
	var last error = ErrNoCredentials
	for _, cand := range candidates {
		quota, err := c.Fetch(ctx, cand)
		if err == nil {
			return quota, cand, nil
		}
		last = err
		if ctx.Err() != nil {
			break
		}
	}
	return Quota{}, Candidate{}, last
}

// endpointFor adds the team query of the monitor API.
func (c Client) endpointFor(cand Candidate) (string, error) {
	endpoint := c.URLFor(cand.Region)
	if !cand.Team {
		return endpoint, nil
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", ErrEndpoint
	}
	query := parsed.Query()
	query.Set("type", "2")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
