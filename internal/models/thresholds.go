package models

// ThresholdRule defines one threshold evaluation rule.
// It is loaded from thresholds.yml and applied during preprocessing.
type ThresholdRule struct {
	Metric   string            `yaml:"metric"`
	Operator string            `yaml:"operator"`
	Warning  *float64          `yaml:"warning"`
	Critical *float64          `yaml:"critical"`
	Tags     map[string]string `yaml:"tags"`
}
