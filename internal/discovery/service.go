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
	if !loaded.Root.Discovery.Enabled {
		return res, nil
	}
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

	fingerprints := s.probeCandidates(ctx, filtered, loaded.Root.Discovery.SNMP)
	for _, fp := range fingerprints {
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
			res.SkippedReasons = append(res.SkippedReasons, fmt.Sprintf("no profile match: %s", fp.Address))
			continue
		}
		if !loaded.Root.Discovery.AutoPromote.Enabled {
			continue
		}
		if loaded.Root.Discovery.AutoPromote.MaxNewDevicesPerCycle > 0 && res.Promoted >= loaded.Root.Discovery.AutoPromote.MaxNewDevicesPerCycle {
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

func (s Service) probeCandidates(ctx context.Context, candidates []Candidate, cfg config.DiscoverySNMP) []Fingerprint {
	if len(candidates) == 0 {
		return nil
	}
	workerCount := cfg.Concurrency
	if workerCount <= 0 {
		workerCount = 1
	}
	type item struct {
		idx int
		fp  Fingerprint
	}
	jobs := make(chan int)
	results := make(chan item, len(candidates))
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				fp, _ := s.Prober.Probe(ctx, candidates[idx], cfg)
				results <- item{idx: idx, fp: fp}
			}
		}()
	}
	for i := range candidates {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	close(results)
	out := make([]Fingerprint, len(candidates))
	for r := range results {
		out[r.idx] = r.fp
	}
	return out
}
