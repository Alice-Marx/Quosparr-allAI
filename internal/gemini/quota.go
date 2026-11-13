package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/licoy/quosparr/internal/netx"
)

// Code Assist endpoints. Both are "colon methods" of the same v1internal
// service: the method name follows the version with a colon.
const (
	// CodeAssistEndpoint is the production host.
	CodeAssistEndpoint = "https://cloudcode-pa.googleapis.com"
	// CodeAssistVersion is the API version of the colon methods.
	CodeAssistVersion = "v1internal"
	// EndpointEnvVar and VersionEnvVar are the CLI's own overrides.
	EndpointEnvVar = "CODE_ASSIST_ENDPOINT"
	VersionEnvVar  = "CODE_ASSIST_API_VERSION"
	// methodLoadCodeAssist reports the tier and the Cloud project.
	methodLoadCodeAssist = "loadCodeAssist"
	// methodUserQuota reports the per-model quota buckets.
	methodUserQuota = "retrieveUserQuota"
	// ReasonNoProject is this package's own reason code for a reading that
	// found no Cloud project to ask about.
	ReasonNoProject = "NO_PROJECT"
	// fallbackLimit is the CLI's own limit when a bucket has no
	// remainingAmount: it hardcodes 100 requests.
	fallbackLimit int64 = 100
	// maxQuotaBuckets bounds a response this package will draw.
	maxQuotaBuckets = 64
	// maxReasonLen bounds a service reason code before it reaches the panel.
	maxReasonLen = 48
)

// Bucket is one model's (or model family's) quota.
type Bucket struct {
	Model string
	// Remaining is the request count the service reported, or, when it
	// reported only a fraction, the round(fraction * fallbackLimit) estimate.
	Remaining int64
	// Limit is the derived total. The service never sends it: the CLI derives
	// it as round(remaining / fraction).
	Limit    int64
	HasLimit bool
	// Fraction is remainingFraction, 0..1.
	Fraction    float64
	HasFraction bool
	ResetAt     time.Time
	HasReset    bool
}

// UsedPercent is (1 - remainingFraction) * 100 clamped to 0..100. A bucket
// without a fraction has no percentage and reports false.
func (b Bucket) UsedPercent() (float64, bool) {
	if !b.HasFraction {
		return 0, false
	}
	used := (1 - b.Fraction) * 100
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	return used, true
}

// Quota is one retrieveUserQuota answer, worst bucket first.
type Quota struct {
	Buckets []Bucket
}

// Tier is what loadCodeAssist reported about the account.
type Tier struct {
	ID   string
	Name string
	// Project is the Cloud project the service named, if any.
	Project string
	// IneligibleReason is the reason code of a tier this client may not use,
	// for example UNSUPPORTED_CLIENT.
	IneligibleReason string
	IneligibleName   string
}

// Client calls the Code Assist colon methods.
type Client struct {
	HTTP     *http.Client
	Endpoint string
	Version  string
	// Token is the access token; it is sent as an Authorization header and
	// appears in no error.
	Token   string
	Timeout time.Duration
}

func (c Client) endpoint() string {
	if c.Endpoint != "" {
		return strings.TrimSuffix(c.Endpoint, "/")
	}
	if fromEnv := strings.TrimSpace(os.Getenv(EndpointEnvVar)); fromEnv != "" {
		return strings.TrimSuffix(fromEnv, "/")
	}
	return CodeAssistEndpoint
}

func (c Client) version() string {
	if c.Version != "" {
		return c.Version
	}
	if fromEnv := strings.TrimSpace(os.Getenv(VersionEnvVar)); fromEnv != "" {
		return fromEnv
	}
	return CodeAssistVersion
}

// methodURL is <endpoint>/<version>:<method>.
func (c Client) methodURL(method string) string {
	return c.endpoint() + "/" + c.version() + ":" + method
}

type loadAssistRequest struct {
	CloudaicompanionProject string     `json:"cloudaicompanionProject,omitempty"`
	Metadata                assistMeta `json:"metadata"`
	Mode                    string     `json:"mode,omitempty"`
}

type assistMeta struct {
	IDEType     string `json:"ideType"`
	Platform    string `json:"platform"`
	PluginType  string `json:"pluginType"`
	DuetProject string `json:"duetProject,omitempty"`
}

type tierAnswer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ineligibleAnswer struct {
	ReasonCode    string `json:"reasonCode"`
	ReasonMessage string `json:"reasonMessage"`
	TierID        string `json:"tierId"`
	TierName      string `json:"tierName"`
}

type loadAssistAnswer struct {
	CurrentTier             *tierAnswer        `json:"currentTier"`
	AllowedTiers            []tierAnswer       `json:"allowedTiers"`
	IneligibleTiers         []ineligibleAnswer `json:"ineligibleTiers"`
	CloudaicompanionProject json.RawMessage    `json:"cloudaicompanionProject"`
}

// LoadCodeAssist reports the tier and the Cloud project. It is the only call
// that also tells the account apart from a client the service still accepts.
func (c Client) LoadCodeAssist(ctx context.Context, project string) (Tier, error) {
	body := loadAssistRequest{
		CloudaicompanionProject: project,
		Metadata: assistMeta{
			IDEType:     "IDE_UNSPECIFIED",
			Platform:    "PLATFORM_UNSPECIFIED",
			PluginType:  "GEMINI",
			DuetProject: project,
		},
		Mode: "HEALTH_CHECK",
	}
	var answer loadAssistAnswer
	if err := c.post(ctx, methodLoadCodeAssist, body, &answer); err != nil {
		return Tier{}, err
	}
	tier := Tier{Project: validProject(projectID(answer.CloudaicompanionProject))}
	if answer.CurrentTier != nil {
		tier.ID = strings.TrimSpace(answer.CurrentTier.ID)
		tier.Name = strings.TrimSpace(answer.CurrentTier.Name)
	}
	if len(answer.IneligibleTiers) > 0 {
		first := answer.IneligibleTiers[0]
		tier.IneligibleReason = sanitizeReason(first.ReasonCode)
		tier.IneligibleName = strings.TrimSpace(first.TierName)
		if tier.Name == "" {
			tier.Name = strings.TrimSpace(first.TierName)
		}
	}
	if tier.Name == "" && len(answer.AllowedTiers) > 0 {
		tier.Name = strings.TrimSpace(answer.AllowedTiers[0].Name)
	}
	return tier, nil
}

// projectID reads cloudaicompanionProject, which is a string in the CLI's own
// code but is accepted as {"id": "..."} as well.
func projectID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	if s := rawString(raw); s != "" {
		return s
	}
	var object struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(raw, &object) == nil {
		return strings.TrimSpace(object.ID)
	}
	return ""
}

type quotaAnswer struct {
	Buckets []rawBucket `json:"buckets"`
}

type rawBucket struct {
	ModelID           string          `json:"modelId"`
	RemainingAmount   json.RawMessage `json:"remainingAmount"`
	RemainingFraction *json.Number    `json:"remainingFraction"`
	ResetTime         string          `json:"resetTime"`
}

// UserQuota asks for the per-model quota of project. A 403 is returned as an
// HTTPError with its reason code, which the caller turns into the
// no-subscription state.
func (c Client) UserQuota(ctx context.Context, project string) (Quota, error) {
	var answer quotaAnswer
	if err := c.post(ctx, methodUserQuota, map[string]string{"project": project}, &answer); err != nil {
		return Quota{}, err
	}
	return answer.quota()
}

func (a quotaAnswer) quota() (Quota, error) {
	if len(a.Buckets) == 0 {
		return Quota{}, nil
	}
	out := Quota{Buckets: make([]Bucket, 0, min(len(a.Buckets), maxQuotaBuckets))}
	for _, raw := range a.Buckets {
		if len(out.Buckets) >= maxQuotaBuckets {
			break
		}
		// The CLI skips a bucket without a model id; so does this package.
		model := strings.TrimSpace(raw.ModelID)
		if model == "" || !validLabel(model, 120) {
			continue
		}
		out.Buckets = append(out.Buckets, raw.bucket(model))
	}
	sort.SliceStable(out.Buckets, func(i, j int) bool {
		left, right := out.Buckets[i], out.Buckets[j]
		switch {
		case left.HasFraction != right.HasFraction:
			return left.HasFraction
		case left.Fraction != right.Fraction:
			return left.Fraction < right.Fraction
		default:
			return left.Model < right.Model
		}
	})
	return out, nil
}

func (r rawBucket) bucket(model string) Bucket {
	b := Bucket{Model: model}
	remaining, hasAmount := parseInt64(r.RemainingAmount)
	if hasAmount {
		b.Remaining = remaining
	}
	if r.RemainingFraction != nil {
		fraction, err := r.RemainingFraction.Float64()
		if err == nil && !math.IsNaN(fraction) && !math.IsInf(fraction, 0) {
			b.Fraction = clampUnit(fraction)
			b.HasFraction = true
		}
	}
	switch {
	case b.HasFraction && hasAmount && b.Fraction > 0:
		// The service never sends the limit; the CLI derives it this way.
		b.Limit = clampLimit(int64(math.Round(float64(b.Remaining)/b.Fraction)), b.Remaining)
		b.HasLimit = true
	case b.HasFraction && hasAmount:
		// A zero fraction says the window is exhausted, not how big it was.
		b.Limit, b.HasLimit = b.Remaining, true
	case b.HasFraction:
		// The CLI's own fallback: no request count, so assume 100.
		b.Limit, b.HasLimit = fallbackLimit, true
		b.Remaining = int64(math.Round(b.Fraction * float64(fallbackLimit)))
	}
	if reset, ok := parseTime(r.ResetTime); ok {
		b.ResetAt, b.HasReset = reset, true
	}
	return b
}

// ParseQuota reads a retrieveUserQuota body. It exists for tests and for a
// caller that already holds the answer.
func ParseQuota(data []byte) (Quota, error) {
	var answer quotaAnswer
	if err := json.Unmarshal(data, &answer); err != nil {
		return Quota{}, ErrBadResponse
	}
	return answer.quota()
}

func clampUnit(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// clampLimit keeps a derived limit sane: never below the remaining count and
// never above a ceiling that keeps the arithmetic finite.
func clampLimit(limit, remaining int64) int64 {
	if limit < remaining {
		limit = remaining
	}
	if limit > 1_000_000_000 {
		limit = 1_000_000_000
	}
	if limit < 0 {
		limit = 0
	}
	return limit
}

// parseInt64 reads an int64 the proto JSON encoder writes as a string, and
// also accepts a plain number.
func parseInt64(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		value, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		return value, err == nil
	}
	var n json.Number
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if dec.Decode(&n) != nil {
		return 0, false
	}
	if i, err := n.Int64(); err == nil {
		return i, true
	}
	if f, err := n.Float64(); err == nil {
		return int64(math.Round(f)), true
	}
	return 0, false
}

func parseTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (c Client) post(ctx context.Context, method string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return ErrBadResponse
	}
	client, err := c.client(ctx)
	if err != nil {
		return err
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.methodURL(method), bytes.NewReader(payload))
	if err != nil {
		return ErrBadResponse
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := client.Do(req)
	if err != nil {
		return ErrNetwork
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return ErrNetwork
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &HTTPError{Status: resp.StatusCode, Reason: reasonOf(data)}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return ErrBadResponse
	}
	return nil
}

func (c Client) client(ctx context.Context) (*http.Client, error) {
	if c.HTTP != nil {
		return c.HTTP, nil
	}
	client, err := netx.ClientFor(ctx, netx.WithoutCookies)
	if err != nil {
		return nil, ErrNetwork
	}
	return client, nil
}

// reasonOf reads the google.rpc ErrorInfo reason of an error body. The message
// itself is dropped: this package never echoes remote text.
func reasonOf(data []byte) string {
	var envelope struct {
		Error struct {
			Details []struct {
				Reason string `json:"reason"`
				Status string `json:"status"`
			} `json:"details"`
			Status string `json:"status"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return ""
	}
	for _, detail := range envelope.Error.Details {
		if reason := sanitizeReason(detail.Reason); reason != "" {
			return reason
		}
	}
	return sanitizeReason(envelope.Error.Status)
}

// sanitizeReason keeps a reason code drawable: letters, digits, underscore and
// dash, at most maxReasonLen.
func sanitizeReason(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			return b.String()
		}
		if b.Len() >= maxReasonLen {
			break
		}
	}
	return b.String()
}

// ProjectFromEnv reads the Cloud project from the environment, in the order
// the CLI itself resolves it. A purely numeric project id is rejected: the CLI
// treats it as an error rather than sending it.
func ProjectFromEnv() string {
	for _, key := range []string{"GOOGLE_CLOUD_PROJECT", "GOOGLE_CLOUD_PROJECT_ID"} {
		if project := validProject(os.Getenv(key)); project != "" {
			return project
		}
	}
	return ""
}

// ProjectFromSettings reads the Cloud project from the CLI settings file. The
// key names are the ones loadCodeAssist uses; a file without them, or an
// unreadable file, yields the empty string.
func ProjectFromSettings(path string) string {
	data, err := readSmallFile(path, maxCredsBytes)
	if err != nil {
		return ""
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return ""
	}
	for _, key := range []string{"cloudaicompanionProject", "project"} {
		if project := validProject(rawString(raw[key])); project != "" {
			return project
		}
	}
	return ""
}

// validProject keeps an id a Cloud project could be: not empty, not purely
// numeric (the CLI rejects those), bounded and made of id characters.
func validProject(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 100 {
		return ""
	}
	digits := true
	for _, r := range raw {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '-', r == '.', r == ':', r == '_':
			digits = false
		default:
			return ""
		}
	}
	if digits {
		return ""
	}
	return raw
}

// readSmallFile reads at most max bytes of a regular file.
func readSmallFile(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return nil, ErrUnreadable
	}
	if info.Size() > max {
		return nil, ErrTooLarge
	}
	return io.ReadAll(io.LimitReader(f, max))
}

// validLabel reports whether raw is a short, control-free label.
func validLabel(raw string, maxRunes int) bool {
	if raw == "" || utf8.RuneCountInString(raw) > maxRunes {
		return false
	}
	for _, r := range raw {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
