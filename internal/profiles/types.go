package profiles

// Profile defines a set of SNMP metrics to poll for a device class.
// It is intentionally simple for Phase 6 MVP (no cache, no discovery).
type Profile struct {
	Name    string   `yaml:"name"`
	Match   Match    `yaml:"match"`
	Metrics []Metric `yaml:"metrics"`
}

// Match selects which devices the profile applies to.
// Empty values mean "wildcard".
type Match struct {
	Vendor string `yaml:"vendor"`
	Model  string `yaml:"model"`
}

// Metric describes one polled OID mapping.
// If Index is true, the OID is walked and the interface index is added as tag `ifIndex`.
type Metric struct {
	Metric string `yaml:"metric"`
	OID    string `yaml:"oid"`
	Type   string `yaml:"type"` // "get" or "walk"
	Unit   string `yaml:"unit"`
	Index  bool   `yaml:"index"`
}
