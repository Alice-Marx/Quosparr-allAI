package opencode

import (
	"path/filepath"
	"testing"
	"time"
)

func TestParseCredentialReadsTheConsoleLogin(t *testing.T) {
	raw := `{"type":"oauth","methodID":"device","refresh":"REFRESH-SECRET","access":"ACCESS-SECRET",
 "expires":1797051604005,"metadata":{"server":"https://opencode.ai/console","accountID":"acc_1",
 "email":"a@example.com","orgID":"org_1","orgName":"Team"}}`
	cred, err := parseCredential([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if cred.Kind != CredentialOAuth || cred.OrgID != "org_1" || cred.OrgName != "Team" || cred.Email != "a@example.com" {
		t.Fatalf("credential = %+v", cred)
	}
	if !cred.HasExpiry || !cred.Expires.Equal(time.UnixMilli(1797051604005)) {
		t.Fatalf("expires = %v", cred.Expires)
	}
	if !cred.Usable(cred.Expires.Add(-time.Hour)) || cred.Expired(cred.Expires.Add(-time.Hour)) {
		t.Error("a login before its expiry must be usable")
	}
	if cred.Usable(cred.Expires) || !cred.Expired(cred.Expires) {
		t.Error("a login at its expiry must be expired")
	}
}

func TestParseCredentialAcceptsAnAPIKey(t *testing.T) {
	cred, err := parseCredential([]byte(`{"type":"key","key":"KEY-SECRET"}`))
	if err != nil {
		t.Fatal(err)
	}
	if cred.Kind != CredentialKey || cred.token() != "KEY-SECRET" {
		t.Fatalf("credential = %+v", cred)
	}
	if cred.HasExpiry || !cred.Usable(time.Now()) {
		t.Error("a key without an expiry is usable")
	}
	if cred.HasOrg() {
		t.Error("a key has no organization scope")
	}
}

func TestParseCredentialRejectsAnEmptyLogin(t *testing.T) {
	for _, raw := range []string{`{}`, `{"type":"oauth"}`, "not json"} {
		if _, err := parseCredential([]byte(raw)); err == nil {
			t.Errorf("%q was accepted", raw)
		}
	}
}

func TestParseEpochReadsMillisecondsAndSeconds(t *testing.T) {
	if at, ok := parseEpoch("1797051604005"); !ok || !at.Equal(time.UnixMilli(1797051604005)) {
		t.Errorf("milliseconds = %v %v", at, ok)
	}
	if at, ok := parseEpoch("1797051604"); !ok || !at.Equal(time.Unix(1797051604, 0)) {
		t.Errorf("seconds = %v %v", at, ok)
	}
	if _, ok := parseEpoch("0"); ok {
		t.Error("a zero expiry is not an expiry")
	}
}

func TestParseAuthFileReadsTheIntegrationEntry(t *testing.T) {
	raw := `{"other":{"type":"oauth","access":"OTHER"},"opencode":{"type":"oauth","access":"ACCESS","expires":1797051604005,"accountId":"acc_1"}}`
	cred, err := parseAuthFile([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if cred.token() != "ACCESS" {
		t.Fatalf("credential = %+v", cred)
	}
	if _, err := parseAuthFile([]byte(`{"other":{}}`)); err == nil {
		t.Error("a file without the opencode entry has no login")
	}
}

func TestDataDirFollowsXdg(t *testing.T) {
	home := filepath.Join("C:", "Users", "tester")
	t.Setenv("XDG_DATA_HOME", "")
	if got, want := DataDir(home), filepath.Join(home, ".local", "share", "opencode"); got != want {
		t.Errorf("DataDir = %q, want %q", got, want)
	}
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	if got, want := DataDir(home), filepath.Join(dir, "opencode"); got != want {
		t.Errorf("DataDir = %q, want %q", got, want)
	}
	if got, want := DatabasePath(home), filepath.Join(dir, "opencode", DatabaseName); got != want {
		t.Errorf("DatabasePath = %q, want %q", got, want)
	}
}
