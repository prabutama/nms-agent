package discovery

import (
	"context"

	"nms-agent/internal/config"
	"nms-agent/internal/profiles"
)

type Candidate struct {
	Address   string
	MAC       string
	Interface string
	Source    string
}

type Fingerprint struct {
	Candidate
	SNMPOK      bool
	SysObjectID string
	SysName     string
	SysDescr    string
	Vendor      string
	Model       string
	ProfileName string
}

type Result struct {
	CandidatesFound   int
	ExistingSkipped   int
	SNMPOK            int
	ProfileMatched    int
	GeneratedProfiles int
	Promoted          int
	Changed           bool
	SkippedReasons    []string
}

type CandidateProvider interface {
	Candidates(ctx context.Context, loaded config.Loaded) ([]Candidate, error)
}

type Prober interface {
	Probe(ctx context.Context, candidate Candidate, cfg config.DiscoverySNMP) (Fingerprint, error)
}

type ExplorationResult struct {
	Generated   bool
	ProfilePath string
	Profile     profiles.Profile
	Vendor      string
	Model       string
}

type Explorer interface {
	Explore(ctx context.Context, configPath string, loaded config.Loaded, fp Fingerprint) (ExplorationResult, error)
}
