package opencode

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// offlineClient fails the test if anything reaches the network.
func offlineClient(t *testing.T) Client {
	t.Helper()
	return Client{HTTP: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("Load reached the network without a usable login")
		return nil, ErrUnavailable
	})}}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLoadWithoutADatabaseReportsNoCredential(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	snap := Load(context.Background(), home, offlineClient(t), time.Now())
	if snap.State != StateNoCredential {
		t.Fatalf("state = %v err=%v", snap.State, snap.Err)
	}
	if snap.DatabasePath == "" || snap.HasCost {
		t.Fatalf("snapshot = %+v", snap)
	}
	if snap.LocalErr != nil {
		t.Errorf("a missing database is not an error: %v", snap.LocalErr)
	}
}

func TestIdentityFallsBackToTheOrganizationID(t *testing.T) {
	if got := identityOf(Credential{OrgName: "Team", OrgID: "org_1"}).Name(); got != "Team" {
		t.Errorf("name = %q", got)
	}
	if got := identityOf(Credential{OrgID: "org_1"}).Name(); got != "org_1" {
		t.Errorf("name = %q", got)
	}
	if got := identityOf(Credential{}).Name(); got != "" {
		t.Errorf("name = %q", got)
	}
	id := identityOf(Credential{OrgID: "org_1"})
	if id.UsageURL != "https://opencode.ai/console/org_1/activity/overview" {
		t.Errorf("url = %q", id.UsageURL)
	}
	if identityOf(Credential{}).UsageURL != "" {
		t.Error("no organization, no usage page")
	}
}
