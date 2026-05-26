package profiles

import "testing"

func TestValidateAll_RequiresStandardProfile(t *testing.T) {
	err := ValidateAll([]Profile{{
		Name:    "v1",
		Match:   Match{Vendor: "cisco", Model: "2900"},
		Metrics: []Metric{{Metric: "m", OID: "1.3.6.1.2.1.1.3.0", Type: "get"}},
	}})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestSelectProfile_Precedence(t *testing.T) {
	profiles := []Profile{
		{Name: "standard", Match: Match{}, Metrics: []Metric{{Metric: "m", OID: "1.3.6", Type: "get"}}},
		{Name: "vendor-default", Match: Match{Vendor: "example"}, Metrics: []Metric{{Metric: "m", OID: "1.3.6", Type: "get"}}},
		{Name: "exact", Match: Match{Vendor: "example", Model: "r1"}, Metrics: []Metric{{Metric: "m", OID: "1.3.6", Type: "get"}}},
	}
	if _, err := func() (Profile, error) {
		if err := ValidateAll(profiles); err != nil {
			return Profile{}, err
		}
		p, _ := SelectProfile(profiles, "example", "r1")
		return p, nil
	}(); err != nil {
		t.Fatalf("ValidateAll: %v", err)
	}

	p, ok := SelectProfile(profiles, "example", "r1")
	if !ok || p.Name != "exact" {
		t.Fatalf("expected exact, got ok=%v name=%q", ok, p.Name)
	}
	p2, ok := SelectProfile(profiles, "example", "other")
	if !ok || p2.Name != "vendor-default" {
		t.Fatalf("expected vendor-default, got ok=%v name=%q", ok, p2.Name)
	}
	p3, ok := SelectProfile(profiles, "unknown", "x")
	if !ok || p3.Name != "standard" {
		t.Fatalf("expected standard, got ok=%v name=%q", ok, p3.Name)
	}
}
