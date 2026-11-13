package glm

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"
)

var (
	// ErrUnauthorized means the endpoint rejected the credential, which on
	// this vendor means it expired and the user has to sign in again.
	ErrUnauthorized = errors.New("glm: credential expired or rejected")
	// ErrBadResponse is an HTTP status outside 2xx.
	ErrBadResponse = errors.New("glm: quota endpoint returned an HTTP error")
	// ErrBusinessCode is a vendor error envelope under an HTTP 200.
	ErrBusinessCode = errors.New("glm: quota endpoint returned a business error")
	// ErrNetwork is a request that never produced a response.
	ErrNetwork = errors.New("glm: quota request failed")
	// ErrEndpoint is an unparseable endpoint URL.
	ErrEndpoint = errors.New("glm: quota endpoint URL is invalid")
	// ErrDecode is a body that is not the documented envelope.
	ErrDecode = errors.New("glm: quota response could not be read")
)

// PrimaryLimitType is the bucket the ZCode CLI shows as the headline share.
const PrimaryLimitType = "TIME_LIMIT"

// Limit is one quota bucket. A bucket the endpoint typed with an empty
// string is dropped, as the CLI does.
type Limit struct {
	Type string
	// Percent is the used share from 0 to 100.
	Percent    float64
	HasPercent bool
	// Derived marks a share computed from remaining and total because the
	// endpoint reported no percentage.
	Derived  bool
	ResetAt  time.Time
	HasReset bool
}

// Quota is one reading of the quota endpoint.
type Quota struct {
	Level      string
	Limits     []Limit
	Primary    Limit
	HasPrimary bool
	// Endpoint is the URL that answered; it never holds the credential.
	Endpoint  string
	FetchedAt time.Time
}

// envelope is the vendor wrapper. HTTP 200 does not imply success: the code
// may be a 401 business error.
type envelope struct {
	Code    *int          `json:"code"`
	Success *bool         `json:"success"`
	Data    *dataEnvelope `json:"data"`
}

type dataEnvelope struct {
	Level  *string     `json:"level"`
	Limits []limitWire `json:"limits"`
}

// limitWire is the part of a bucket the panel shows. The other documented
// fields, including usageDetails, are ignored: their unit is not verified.
type limitWire struct {
	Type          *string  `json:"type"`
	Number        *float64 `json:"number"`
	Remaining     *float64 `json:"remaining"`
	Percentage    *float64 `json:"percentage"`
	NextResetTime *float64 `json:"nextResetTime"`
}

// DecodeQuota parses one quota response body. The business code is checked
// before the data, so an expired credential reported under HTTP 200 is an
// error rather than an empty reading.
func DecodeQuota(data []byte, now time.Time) (Quota, error) {
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return Quota{}, ErrDecode
	}
	if env.Success != nil && !*env.Success {
		return Quota{}, businessError(env.Code)
	}
	if env.Code != nil && *env.Code != 0 && *env.Code != 200 {
		return Quota{}, businessError(env.Code)
	}
	if env.Data == nil {
		return Quota{}, ErrDecode
	}
	quota := Quota{FetchedAt: now}
	if env.Data.Level != nil {
		quota.Level = strings.TrimSpace(*env.Data.Level)
	}
	for _, wire := range env.Data.Limits {
		limit, ok := parseLimit(wire)
		if !ok {
			continue
		}
		quota.Limits = append(quota.Limits, limit)
	}
	quota.Primary, quota.HasPrimary = PickPrimary(quota.Limits)
	return quota, nil
}

// businessError classifies a vendor envelope error.
func businessError(code *int) error {
	if code != nil && *code == 401 {
		return ErrUnauthorized
	}
	return ErrBusinessCode
}

// parseLimit keeps the fields the panel shows. An unknown type is kept as
// the endpoint spelled it instead of being guessed into a known one.
func parseLimit(wire limitWire) (Limit, bool) {
	if wire.Type == nil {
		return Limit{}, false
	}
	name := strings.TrimSpace(*wire.Type)
	if name == "" {
		return Limit{}, false
	}
	limit := Limit{Type: name}
	switch {
	case wire.Percentage != nil:
		limit.Percent, limit.HasPercent = ClampPercent(*wire.Percentage), true
	case wire.Remaining != nil && wire.Number != nil && *wire.Number > 0:
		limit.Percent, limit.HasPercent, limit.Derived = ClampPercent(100 - *wire.Remaining / *wire.Number * 100), true, true
	}
	if at, ok := UnixTime(wire.NextResetTime); ok {
		limit.ResetAt, limit.HasReset = at, true
	}
	return limit, true
}

// PickPrimary mirrors the ZCode CLI: the TIME_LIMIT bucket, else the first
// bucket with a share, else the first bucket.
func PickPrimary(limits []Limit) (Limit, bool) {
	for _, limit := range limits {
		if limit.Type == PrimaryLimitType {
			return limit, true
		}
	}
	for _, limit := range limits {
		if limit.HasPercent {
			return limit, true
		}
	}
	if len(limits) == 0 {
		return Limit{}, false
	}
	return limits[0], true
}

// ClampPercent pins a used share to 0..100. The endpoint reports 0..100, so
// a larger value means the field changed; clamping keeps the bar inside its
// track instead of drawing past it.
func ClampPercent(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return math.Max(0, math.Min(100, v))
}

// UnixTime converts an epoch that may be seconds or milliseconds. The ZCode
// source reads nextResetTime as milliseconds without converting, which is
// unverified; a value below that range is read as seconds so a seconds value
// is corrected rather than landing near 1970.
func UnixTime(v *float64) (time.Time, bool) {
	if v == nil {
		return time.Time{}, false
	}
	n := *v
	if math.IsNaN(n) || n <= 0 {
		return time.Time{}, false
	}
	switch {
	case n >= 1e11:
		return time.UnixMilli(int64(n)).UTC(), true
	case n >= 1e9:
		return time.Unix(int64(n), 0).UTC(), true
	}
	return time.Time{}, false
}

// FormatPercent is the panel form: one decimal at most, no trailing zero.
func FormatPercent(v float64) string {
	rounded := math.Round(v*10) / 10
	if rounded == math.Trunc(rounded) {
		return formatFixed(rounded, 0)
	}
	return formatFixed(rounded, 1)
}
