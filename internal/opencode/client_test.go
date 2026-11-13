package opencode

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (Client, *[]*http.Request) {
	t.Helper()
	var seen []*http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return Client{
		HTTP:       srv.Client(),
		ZenURL:     srv.URL + "/zen/go/v1/usage",
		BudgetsURL: srv.URL + "/console/api/policies/current-budgets",
	}, &seen
}

func TestUsageSendsBothHeaders(t *testing.T) {
	client, seen := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, usageBody)
	})
	cred := Credential{access: "TOKEN-SECRET", OrgID: "org_1"}
	usage, ok, err := client.Usage(context.Background(), cred)
	if err != nil || !ok {
		t.Fatalf("usage = %+v ok=%v err=%v", usage, ok, err)
	}
	if usage.Monthly.Percent != 75 {
		t.Fatalf("usage = %+v", usage)
	}
	if len(*seen) != 1 {
		t.Fatalf("requests = %d", len(*seen))
	}
	req := (*seen)[0]
	if got := req.Header.Get("Authorization"); got != "Bearer TOKEN-SECRET" {
		t.Errorf("authorization = %q", got)
	}
	if got := req.Header.Get(usageOrgHeader); got != "org_1" {
		t.Errorf("usage org header = %q", got)
	}
	if got := req.Header.Get(budgetOrgHeader); got != "" {
		t.Errorf("budget org header leaked into the usage request: %q", got)
	}
}

func TestBudgetsUsesTheOtherOrgHeader(t *testing.T) {
	client, seen := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `[{"ruleID":"r1","limitDollars":50,"spentDollars":12.34,"scope":"org","per":"month"}]`)
	})
	cred := Credential{access: "TOKEN", OrgID: "org_1"}
	budgets, ok, err := client.Budgets(context.Background(), cred)
	if err != nil || !ok || len(budgets) != 1 {
		t.Fatalf("budgets = %+v ok=%v err=%v", budgets, ok, err)
	}
	if got := (*seen)[0].Header.Get(budgetOrgHeader); got != "org_1" {
		t.Errorf("budget org header = %q", got)
	}
}

func TestForbiddenIsNoDataNotAnError(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	cred := Credential{access: "TOKEN"}
	if _, ok, err := client.Usage(context.Background(), cred); ok || err != nil {
		t.Errorf("usage: ok=%v err=%v", ok, err)
	}
	if _, ok, err := client.Budgets(context.Background(), cred); ok || err != nil {
		t.Errorf("budgets: ok=%v err=%v", ok, err)
	}
}

func TestUnauthorizedAndServerErrors(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	cred := Credential{access: "TOKEN"}
	if _, _, err := client.Usage(context.Background(), cred); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("401 err = %v", err)
	}

	broken, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	if _, _, err := broken.Usage(context.Background(), cred); !errors.Is(err, ErrUnavailable) {
		t.Errorf("500 err = %v", err)
	}
	garbage, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "not json")
	})
	if _, _, err := garbage.Usage(context.Background(), cred); !errors.Is(err, ErrDecode) {
		t.Errorf("decode err = %v", err)
	}
}

func TestExpiredCredentialIsNotSent(t *testing.T) {
	client, seen := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("an expired login must not reach the network")
	})
	cred := Credential{access: "TOKEN", Expires: time.Now().Add(-time.Hour), HasExpiry: true}
	if _, _, err := client.Usage(context.Background(), cred); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("err = %v", err)
	}
	if len(*seen) != 0 {
		t.Errorf("requests = %d", len(*seen))
	}
}

func TestEmptyBudgetListIsAKnownAnswer(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "[]")
	})
	budgets, ok, err := client.Budgets(context.Background(), Credential{access: "TOKEN"})
	if err != nil || !ok || len(budgets) != 0 {
		t.Fatalf("budgets = %+v ok=%v err=%v", budgets, ok, err)
	}
}
