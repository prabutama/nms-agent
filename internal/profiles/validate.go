package profiles

import (
	"fmt"
	"regexp"
	"strings"
)

var reOID = regexp.MustCompile(`^[0-9]+(\.[0-9]+)*$`)

// ValidateAll validates profiles and ensures selection rules are unambiguous.
func ValidateAll(profiles []Profile) error {
	if len(profiles) == 0 {
		return fmt.Errorf("no profiles loaded")
	}

	seenKeys := map[string]struct{}{}
	hasStandard := false
	for i, p := range profiles {
		if err := ValidateProfile(p); err != nil {
			return fmt.Errorf("profiles[%d] (%s): %w", i, p.Name, err)
		}
		k := selectionKey(p.Match.Vendor, p.Match.Model)
		if _, ok := seenKeys[k]; ok {
			return fmt.Errorf("duplicate profile match key: vendor=%q model=%q", p.Match.Vendor, p.Match.Model)
		}
		seenKeys[k] = struct{}{}
		if normalize(p.Match.Vendor) == "" && normalize(p.Match.Model) == "" {
			hasStandard = true
		}
	}
	if !hasStandard {
		return fmt.Errorf("missing standard profile (match.vendor and match.model must be empty)")
	}
	return nil
}

func ValidateProfile(p Profile) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}

	if len(p.Metrics) == 0 {
		return fmt.Errorf("metrics must not be empty")
	}

	seenMetric := map[string]struct{}{}
	seenOID := map[string]struct{}{}
	for i, m := range p.Metrics {
		metric := strings.TrimSpace(m.Metric)
		if metric == "" {
			return fmt.Errorf("metrics[%d].metric is required", i)
		}
		if _, ok := seenMetric[metric]; ok {
			return fmt.Errorf("duplicate metric name: %s", metric)
		}
		seenMetric[metric] = struct{}{}

		oid := strings.TrimSpace(m.OID)
		if oid == "" {
			return fmt.Errorf("metrics[%d].oid is required", i)
		}
		if !reOID.MatchString(oid) {
			return fmt.Errorf("metrics[%d].oid invalid: %q", i, oid)
		}
		if _, ok := seenOID[m.Type+":"+oid]; ok {
			return fmt.Errorf("duplicate oid: %s", oid)
		}
		seenOID[m.Type+":"+oid] = struct{}{}

		t := strings.ToLower(strings.TrimSpace(m.Type))
		if t != "get" && t != "walk" {
			return fmt.Errorf("metrics[%d].type must be 'get' or 'walk'", i)
		}
	}

	return nil
}

// SelectProfile chooses a profile using the Phase 6 MVP precedence:
// 1) exact vendor+model
// 2) vendor default (vendor + empty model)
// 3) standard (empty vendor + empty model)
func SelectProfile(profiles []Profile, vendor, model string) (Profile, bool) {
	v := normalize(vendor)
	m := normalize(model)

	// Exact.
	if p, ok := find(profiles, v, m); ok {
		return p, true
	}
	// Vendor default.
	if v != "" {
		if p, ok := find(profiles, v, ""); ok {
			return p, true
		}
	}
	// Standard.
	return find(profiles, "", "")
}

func find(profiles []Profile, vendor, model string) (Profile, bool) {
	for _, p := range profiles {
		if normalize(p.Match.Vendor) == vendor && normalize(p.Match.Model) == model {
			return p, true
		}
	}
	return Profile{}, false
}

func selectionKey(vendor, model string) string {
	return normalize(vendor) + "|" + normalize(model)
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
