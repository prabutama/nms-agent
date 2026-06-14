package discovery

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"nms-agent/internal/config"
	"nms-agent/internal/profiles"
)

const (
	SkipReasonSNMPProbeFailed = "SNMP_PROBE_FAILED"
	DefaultMaxNewDevices      = 50
	UnlimitedMaxNewDevices    = -1
)

type Service struct {
	Provider CandidateProvider
	Prober   Prober
	Explorer Explorer
}

func (s Service) RunOnce(ctx context.Context, configPath string, loaded config.Loaded) (Result, error) {
	return s.run(ctx, configPath, loaded, true)
}

func (s Service) PreviewOnce(ctx context.Context, configPath string, loaded config.Loaded) (Result, error) {
	return s.run(ctx, configPath, loaded, false)
}

func (s Service) run(ctx context.Context, configPath string, loaded config.Loaded, apply bool) (Result, error) {
	var res Result
	if s.Provider == nil {
		return res, fmt.Errorf("discovery provider is required")
	}
	if s.Prober == nil {
		s.Prober = SNMPProber{}
	}
	candidates, err := s.Provider.Candidates(ctx, loaded)
	if err != nil {
		return res, err
	}
	res.CandidatesFound = len(candidates)
	existingByAddress := map[string]struct{}{}
	usedIDs := map[string]struct{}{}
	for _, d := range loaded.Devices {
		existingByAddress[strings.TrimSpace(strings.ToLower(d.Address))] = struct{}{}
		usedIDs[strings.TrimSpace(strings.ToLower(d.ID))] = struct{}{}
	}
	baseDir := filepath.Dir(configPath)
	profs, err := profiles.LoadDir(filepath.Clean(loaded.ProfilesDir))
	if err != nil {
		return res, err
	}
	if err := profiles.ValidateAll(profs); err != nil {
		return res, err
	}

	filtered := make([]Candidate, 0, len(candidates))
	for _, c := range candidates {
		if _, ok := existingByAddress[strings.ToLower(strings.TrimSpace(c.Address))]; ok {
			res.ExistingSkipped++
			res.SkippedReasons = append(res.SkippedReasons, fmt.Sprintf("existing address: %s", c.Address))
			continue
		}
		filtered = append(filtered, c)
	}

	probeResults := s.probeCandidates(ctx, filtered, loaded.Root.Discovery.SNMP)
	maxNewDevices := NormalizeMaxNewDevicesLimit(loaded.Root.Discovery.AutoPromote.MaxNewDevicesPerCycle)
	for _, probe := range probeResults {
		if probe.err != nil {
			addr := strings.TrimSpace(probe.candidate.Address)
			if addr == "" {
				addr = "unknown"
			}
			res.SkippedReasons = append(res.SkippedReasons, fmt.Sprintf("%s address=%s error=%v", SkipReasonSNMPProbeFailed, addr, probe.err))
			continue
		}
		fp := probe.fp
		if fp.SNMPOK {
			res.SNMPOK++
		} else if loaded.Root.Discovery.AutoPromote.RequireSNMPOK {
			res.SkippedReasons = append(res.SkippedReasons, fmt.Sprintf("snmp not ok: %s", fp.Address))
			continue
		}
		if loaded.Root.Discovery.AutoPromote.RequireSysObjectID && strings.TrimSpace(fp.SysObjectID) == "" {
			res.SkippedReasons = append(res.SkippedReasons, fmt.Sprintf("missing sysObjectID: %s", fp.Address))
			continue
		}
		fp.Vendor, fp.Model = ResolveFingerprintVendorModel(fp)
		matched := false
		if (fp.Vendor != "" || fp.Model != "") && func() bool {
			profile, ok := profiles.SelectProfile(profs, fp.Vendor, fp.Model)
			if ok {
				fp.ProfileName = profile.Name
				res.ProfileMatched++
			}
			return ok
		}() {
			matched = true
		}
		if !matched && loaded.Root.Discovery.Exploration.Enabled && strings.TrimSpace(loaded.Root.Discovery.Exploration.RunWhen) == "no_profile_match" && s.Explorer != nil {
			exploreLoaded := loaded
			if !apply {
				exploreLoaded.Root.Discovery.Exploration.AutoApproveGeneratedProfile = false
			}
			exploreRes, err := s.Explorer.Explore(ctx, configPath, exploreLoaded, fp)
			if err != nil {
				res.SkippedReasons = append(res.SkippedReasons, fmt.Sprintf("exploration failed %s: %v", fp.Address, err))
			} else if exploreRes.Generated || len(exploreRes.Profile.Metrics) > 0 {
				fp.Vendor = exploreRes.Vendor
				fp.Model = exploreRes.Model
				res.GeneratedProfiles++
				if exploreRes.Generated {
					res.Changed = true
				}
				if apply {
					profs, err = profiles.LoadDir(filepath.Clean(loaded.ProfilesDir))
					if err != nil {
						return res, err
					}
					if err := profiles.ValidateAll(profs); err != nil {
						return res, err
					}
					if profile, ok := profiles.SelectProfile(profs, fp.Vendor, fp.Model); ok {
						fp.ProfileName = profile.Name
						res.ProfileMatched++
						matched = true
					}
				} else {
					fp.ProfileName = exploreRes.Profile.Name
					res.ProfileMatched++
					matched = true
				}
			}
		}
		if !matched && loaded.Root.Discovery.AutoPromote.RequireProfileMatch {
			reason := fmt.Sprintf("no profile match: %s", fp.Address)
			if strings.TrimSpace(fp.SysObjectID) != "" {
				reason += " sysObjectID=" + fp.SysObjectID
			}
			if strings.TrimSpace(fp.SysDescr) != "" {
				reason += " sysDescr=" + strings.TrimSpace(fp.SysDescr)
			}
			res.SkippedReasons = append(res.SkippedReasons, reason)
			continue
		}
		if !loaded.Root.Discovery.AutoPromote.Enabled {
			continue
		}
		if maxNewDevices > 0 && res.Promoted >= maxNewDevices {
			res.SkippedReasons = append(res.SkippedReasons, fmt.Sprintf("promotion limit reached: %s", fp.Address))
			continue
		}
		if !apply {
			res.Promoted++
			continue
		}
		if _, err := writePromotedDevice(baseDir, loaded, fp, usedIDs); err != nil {
			res.SkippedReasons = append(res.SkippedReasons, fmt.Sprintf("promote failed %s: %v", fp.Address, err))
			continue
		}
		res.Promoted++
		res.Changed = true
	}
	sort.Strings(res.SkippedReasons)
	return res, nil
}

func NormalizeMaxNewDevicesLimit(limit int) int {
	if limit == UnlimitedMaxNewDevices {
		return UnlimitedMaxNewDevices
	}
	if limit <= 0 {
		return DefaultMaxNewDevices
	}
	return limit
}

type probeResult struct {
	idx       int
	candidate Candidate
	fp        Fingerprint
	err       error
}

func (s Service) probeCandidates(ctx context.Context, candidates []Candidate, cfg config.DiscoverySNMP) []probeResult {
	if len(candidates) == 0 {
		return nil
	}
	workerCount := cfg.Concurrency
	if workerCount <= 0 {
		workerCount = 1
	}
	jobs := make(chan int)
	results := make(chan probeResult, len(candidates))
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				fp, err := s.Prober.Probe(ctx, candidates[idx], cfg)
				if strings.TrimSpace(fp.Address) == "" {
					fp.Candidate = candidates[idx]
				}
				results <- probeResult{idx: idx, candidate: candidates[idx], fp: fp, err: err}
			}
		}()
	}
	for i := range candidates {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	close(results)
	out := make([]probeResult, len(candidates))
	for r := range results {
		out[r.idx] = r
	}
	return out
}
