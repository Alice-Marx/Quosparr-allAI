package opencode

import (
	"encoding/json"
	"os"
	"time"
)

// maxAuthBytes bounds the legacy credential file this package will read.
const maxAuthBytes = 1 << 20

// CredentialKind is how a stored login authenticates.
type CredentialKind int

const (
	// CredentialOAuth is a Console device-code login with an access token.
	CredentialOAuth CredentialKind = iota
	// CredentialKey is a plain API key, which has no organization scope.
	CredentialKey
)

// Credential is the stored OpenCode Console login. The token itself is
// unexported so that no caller can print it by accident; the panel only needs
// the identity fields.
type Credential struct {
	access    string
	Kind      CredentialKind
	Expires   time.Time
	HasExpiry bool
	OrgID     string
	OrgName   string
	Email     string
}

// token is what the Authorization header carries.
func (c Credential) token() string { return c.access }

// Usable reports whether the login can be sent now. A login without an
// expiry is treated as usable, because the CLI also accepts one.
func (c Credential) Usable(now time.Time) bool {
	if c.access == "" {
		return false
	}
	if !c.HasExpiry {
		return true
	}
	return now.Before(c.Expires)
}

// Expired reports a login that is stored but past its expiry. This package
// does not refresh it: the CLI owns the refresh, and a second refresher would
// invalidate the CLI's login.
func (c Credential) Expired(now time.Time) bool {
	return c.access != "" && c.HasExpiry && !now.Before(c.Expires)
}

// HasOrg reports an organization scope, which the budget endpoint needs.
func (c Credential) HasOrg() bool { return c.OrgID != "" }

type rawMetadata struct {
	OrgID   string `json:"orgID"`
	OrgName string `json:"orgName"`
	Email   string `json:"email"`
	Server  string `json:"server"`
}

type rawCredential struct {
	Type     string      `json:"type"`
	MethodID string      `json:"methodID"`
	Access   string      `json:"access"`
	Key      string      `json:"key"`
	Refresh  string      `json:"refresh"`
	Expires  json.Number `json:"expires"`
	Metadata rawMetadata `json:"metadata"`
}

// parseCredential reads one stored credential. The shape is the v2 database
// one ({type:"oauth",access,expires,metadata}); the legacy auth.json uses the
// same value under each integration id. An unknown type is read as whichever
// token field it carries, which is conservative: a login that works is better
// than a login rejected over a type name this package does not know.
func parseCredential(raw []byte) (Credential, error) {
	var rc rawCredential
	if err := json.Unmarshal(raw, &rc); err != nil {
		return Credential{}, ErrCredential
	}
	cred := Credential{
		Kind:    CredentialOAuth,
		OrgID:   rc.Metadata.OrgID,
		OrgName: rc.Metadata.OrgName,
		Email:   rc.Metadata.Email,
	}
	switch rc.Type {
	case "oauth", "":
		cred.access = rc.Access
		if cred.access == "" {
			cred.access, cred.Kind = rc.Key, CredentialKey
		}
	case "key", "api":
		cred.Kind, cred.access = CredentialKey, rc.Key
		if cred.access == "" {
			cred.access = rc.Access
		}
	default:
		cred.access = rc.Access
		if cred.access == "" {
			cred.access, cred.Kind = rc.Key, CredentialKey
		}
	}
	if cred.access == "" {
		return Credential{}, ErrCredential
	}
	if at, ok := parseEpoch(rc.Expires); ok {
		cred.Expires, cred.HasExpiry = at, true
	}
	return cred, nil
}

// parseEpoch reads the stored expiry. OpenCode writes epoch milliseconds; a
// value too small to be milliseconds is read as seconds so that an older
// auth.json still yields a sane expiry instead of 1970.
func parseEpoch(n json.Number) (time.Time, bool) {
	if n == "" {
		return time.Time{}, false
	}
	ms, err := n.Int64()
	if err != nil {
		f, ferr := n.Float64()
		if ferr != nil {
			return time.Time{}, false
		}
		ms = int64(f)
	}
	if ms <= 0 {
		return time.Time{}, false
	}
	if ms < 1e11 {
		return time.Unix(ms, 0), true
	}
	return time.UnixMilli(ms), true
}

// parseAuthFile reads the legacy credential file: an object keyed by
// integration id.
func parseAuthFile(data []byte) (Credential, error) {
	var file map[string]json.RawMessage
	if err := json.Unmarshal(data, &file); err != nil {
		return Credential{}, ErrCredential
	}
	raw, ok := file[IntegrationID]
	if !ok {
		return Credential{}, ErrNoCredential
	}
	return parseCredential(raw)
}

// readAuthFile reads the first legacy credential file that parses.
func readAuthFile(home string) (Credential, bool) {
	for _, path := range authPaths(home) {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxAuthBytes {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if cred, err := parseAuthFile(data); err == nil {
			return cred, true
		}
	}
	return Credential{}, false
}
