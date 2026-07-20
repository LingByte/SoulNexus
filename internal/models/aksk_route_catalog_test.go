package models

import "testing"

func TestAKSKCatalogEntryByID(t *testing.T) {
	entry, ok := AKSKCatalogEntryByID("assistants.get")
	if !ok || entry.Method != "GET" {
		t.Fatalf("entry=%+v ok=%v", entry, ok)
	}
	_, ok = AKSKCatalogEntryByID("not-a-real-route-id")
	if ok {
		t.Fatal("unknown id should miss")
	}
}

func TestNormalizeAKSKRouteIDs(t *testing.T) {
	got := NormalizeAKSKRouteIDs([]string{" assistants.get ", "assistants.get", "unknown"})
	if len(got) != 1 || got[0] != "assistants.get" {
		t.Fatalf("%v", got)
	}
}

func TestRoutePatternsForIDs(t *testing.T) {
	patterns := RoutePatternsForIDs([]string{"assistants.get"})
	if len(patterns) == 0 {
		t.Fatal("expected pattern")
	}
	if !CredentialRoutesAllowed([]string{"assistants.get"}, "GET", "/assistants/42") {
		t.Fatal("should match catalog route")
	}
}

func TestValidateCredentialRouteIDs_empty(t *testing.T) {
	if _, err := ValidateCredentialRouteIDs(nil); err == nil {
		t.Fatal("empty ids")
	}
}

func TestParseMarshalAKSKRoutePolicy(t *testing.T) {
	raw, err := MarshalAKSKRoutePolicy(AKSKRoutePolicy{RouteIDs: []string{"assistants.get"}})
	if err != nil {
		t.Fatal(err)
	}
	policy, err := ParseAKSKRoutePolicyJSON(raw)
	if err != nil || len(policy.RouteIDs) != 1 {
		t.Fatalf("policy=%+v err=%v", policy, err)
	}
}
