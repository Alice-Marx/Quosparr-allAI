package glm

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeStore writes a credential store into a temporary home. The values are
// obviously fake and are never printed.
func writeStore(t *testing.T, body string) string {
	t.Helper()
	home := t.TempDir()
	path := CredentialPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

func clearEnvKeys(t *testing.T) {
	t.Helper()
	t.Setenv("ZCODE_BIGMODEL_USAGE_API_KEY", "")
	t.Setenv("BIGMODEL_USAGE_API_KEY", "")
}

func TestLoadReadsCodingPlanKeysAndOAuthTokens(t *testing.T) {
	clearEnvKeys(t)
	home := writeStore(t, `{
	  "oauth:bigmodel:access_token": "oauth-value",
	  "oauth:active_provider": "bigmodel",
	  "account-provider:coding-plan:account:bigmodel-individual-coding-plan:account:42:api-key": "individual",
	  "account-provider:coding-plan:account:bigmodel-team-coding-plan:account:42:api-key": "team",
	  "account-provider:coding-plan:account:zai-individual-coding-plan:account:42:api-key": "zai"
	}`)
	store, err := Load(home)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		kind Kind
		plan string
		key  string
		team bool
		org  string
	}{
		{KindCodingPlan, "bigmodel-individual-coding-plan", "individual", false, "42"},
		{KindCodingPlan, "bigmodel-team-coding-plan", "team", true, "42"},
		{KindOAuth, "bigmodel", "oauth-value", false, ""},
		{KindCodingPlan, "zai-individual-coding-plan", "zai", false, "42"},
	}
	if len(store.Candidates) != len(want) {
		t.Fatalf("candidates = %d, want %d", len(store.Candidates), len(want))
	}
	for i, w := range want {
		got := store.Candidates[i]
		if got.Kind != w.kind || got.Plan != w.plan || got.Team != w.team || got.Organization != w.org {
			t.Fatalf("candidate %d = %+v, want %+v", i, got.Plan, w)
		}
		if got.Key != w.key {
			t.Fatalf("candidate %d key mismatch", i)
		}
	}
	if store.Candidates[3].Region != Zai {
		t.Fatal("zai plan was not placed on the international host")
	}
}

func TestLoadPrefersTheEnvironmentKey(t *testing.T) {
	home := writeStore(t, `{"account-provider:coding-plan:account:bigmodel-individual-coding-plan:account:42:api-key": "stored"}`)
	t.Setenv("BIGMODEL_USAGE_API_KEY", "")
	t.Setenv("ZCODE_BIGMODEL_USAGE_API_KEY", "from-env")
	store, err := Load(home)
	if err != nil {
		t.Fatal(err)
	}
	first := store.Candidates[0]
	if first.Kind != KindEnvKey || first.Key != "from-env" || first.Plan != "ZCODE_BIGMODEL_USAGE_API_KEY" {
		t.Fatalf("first candidate = %s", first)
	}
	if len(store.Candidates) != 2 {
		t.Fatalf("candidates = %d", len(store.Candidates))
	}
}

func TestLoadFallsBackToThePlainEnvironmentKey(t *testing.T) {
	t.Setenv("ZCODE_BIGMODEL_USAGE_API_KEY", "")
	t.Setenv("BIGMODEL_USAGE_API_KEY", "plain")
	store, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if store.Candidates[0].Key != "plain" {
		t.Fatal("the plain environment key was not used")
	}
}

func TestLoadMissingStoreIsNoCredentials(t *testing.T) {
	t.Setenv("ZCODE_BIGMODEL_USAGE_API_KEY", "")
	t.Setenv("BIGMODEL_USAGE_API_KEY", "")
	store, err := Load(t.TempDir())
	if !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("err = %v", err)
	}
	if store.Usable() {
		t.Fatal("a missing store reported a candidate")
	}
	if store.Path == "" || !strings.HasSuffix(store.Path, "credentials.json") {
		t.Fatalf("path = %q", store.Path)
	}
}

func TestLoadToleratesUnexpectedValues(t *testing.T) {
	home := writeStore(t, `{"oauth:bigmodel:access_token": 7, "zcodejwttoken": "x",
	  "account-provider:coding-plan:account:bigmodel-individual-coding-plan:account:42:api-key": "  "}`)
	store, err := Load(home)
	if !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("err = %v", err)
	}
	if store.Usable() {
		t.Fatal("blank keys and non-strings produced a candidate")
	}
}

func TestLoadUnreadableStoreIsAFormatError(t *testing.T) {
	home := writeStore(t, "not json")
	if _, err := Load(home); !errors.Is(err, ErrCredentialFormat) {
		t.Fatalf("err = %v", err)
	}
}

func TestCandidateStringNeverHoldsTheKey(t *testing.T) {
	cand := Candidate{Key: "SECRET-KEY", Kind: KindCodingPlan, Plan: "bigmodel-individual-coding-plan"}
	for _, text := range []string{cand.String(), strings.TrimSpace(fmtCandidate(cand))} {
		if strings.Contains(text, "SECRET-KEY") {
			t.Fatalf("formatted credential leaked the key: %q", text)
		}
	}
}

// fmtCandidate formats the candidate the sloppiest way a log line would.
func fmtCandidate(cand Candidate) string {
	return strings.Join([]string{cand.String(), cand.Region.Host()}, " ")
}
