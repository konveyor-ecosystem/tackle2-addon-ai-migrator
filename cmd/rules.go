package main

import (
	"os"
	"path"
	"strconv"

	"github.com/konveyor/tackle2-hub/shared/addon/scm"
	"github.com/konveyor/tackle2-hub/shared/api"
)

// Rules settings for hub-managed rulesets.
type Rules struct {
	// Path within the application bucket containing rules.
	Path string `json:"path,omitempty"`
	// Repository containing rules.
	Repository *api.Repository `json:"repository,omitempty"`
	// Identity for repository credentials.
	Identity *api.Ref `json:"identity,omitempty"`
	// RuleSets selected by ID.
	RuleSets []api.Ref `json:"ruleSets,omitempty"`
	// ruleFiles uploaded to the hub.
	ruleFiles []api.Ref
	// rules directories on disk after Build().
	rules []string
}

// With populates rules from an analysis profile.
func (r *Rules) With(p *api.ApRules) {
	r.Identity = p.Identity
	r.Repository = p.Repository
	r.ruleFiles = p.Files
}

// Build fetches all rules from the hub to local disk.
func (r *Rules) Build() (err error) {
	err = r.addFiles()
	if err != nil {
		return
	}

	err = r.addRepository()
	if err != nil {
		return
	}

	err = r.addRuleSets()
	if err != nil {
		return
	}

	return
}

// Paths returns the local directories containing fetched rules.
func (r *Rules) Paths() []string {
	return r.rules
}

// addFiles fetches uploaded rule files from the hub.
func (r *Rules) addFiles() (err error) {
	ruleDir := path.Join(RuleDir, "files")
	err = os.MkdirAll(ruleDir, 0755)
	if err != nil {
		return
	}
	if len(r.ruleFiles) > 0 {
		for _, ref := range r.ruleFiles {
			fileId := strconv.Itoa(int(ref.ID))
			dest := path.Join(ruleDir, fileId)
			addon.Activity(
				"[RULES] fetching file: (id=%d) => %s",
				ref.ID,
				dest)
			err = addon.File.Get(ref.ID, dest)
			if err != nil {
				return
			}
		}
		r.rules = append(r.rules, ruleDir)
	} else if r.Path != "" {
		addon.Activity(
			"[RULES] fetching bucket: %s",
			r.Path,
		)
		bucket := addon.Bucket()
		err = bucket.Get(r.Path, ruleDir)
		if err != nil {
			return
		}
		r.rules = append(r.rules, ruleDir)
	}
	return
}

// addRuleSets fetches rulesets and their dependencies from the hub.
func (r *Rules) addRuleSets() (err error) {
	history := make(map[uint]bool)
	for _, ref := range r.RuleSets {
		ruleSet, fErr := addon.RuleSet.Get(ref.ID)
		if fErr != nil {
			err = fErr
			return
		}
		if history[ruleSet.ID] {
			continue
		}
		addon.Activity(
			"[RULES] fetching ruleset: id=%d (%s)",
			ruleSet.ID,
			ruleSet.Name,
		)
		history[ruleSet.ID] = true
		err = r.addRuleSetRules(ruleSet)
		if err != nil {
			return
		}
		err = r.addRuleSetRepository(ruleSet)
		if err != nil {
			return
		}
		err = r.addDeps(ruleSet, history)
		if err != nil {
			return
		}
	}
	return
}

// addDeps fetches dependent rulesets recursively.
func (r *Rules) addDeps(ruleSet *api.RuleSet, history map[uint]bool) (err error) {
	for _, ref := range ruleSet.DependsOn {
		if history[ref.ID] {
			continue
		}
		history[ref.ID] = true
		dep, fErr := addon.RuleSet.Get(ref.ID)
		if fErr != nil {
			err = fErr
			return
		}
		addon.Activity(
			"[RULES] fetching ruleset (dep): id=%d (%s)",
			dep.ID,
			dep.Name,
		)
		err = r.addRuleSetRules(dep)
		if err != nil {
			return
		}
		err = r.addRuleSetRepository(dep)
		if err != nil {
			return
		}
		err = r.addDeps(dep, history)
		if err != nil {
			return
		}
	}
	return
}

// addRuleSetRules downloads individual rule files from a ruleset.
func (r *Rules) addRuleSetRules(ruleSet *api.RuleSet) (err error) {
	if len(ruleSet.Rules) == 0 {
		return
	}
	ruleDir := path.Join(
		RuleDir,
		"rulesets",
		strconv.Itoa(int(ruleSet.ID)),
		"rules",
	)
	err = os.MkdirAll(ruleDir, 0755)
	if err != nil {
		return
	}
	r.rules = append(r.rules, ruleDir)
	for _, rule := range ruleSet.Rules {
		if rule.File == nil {
			continue
		}
		err = addon.File.Get(
			rule.File.ID,
			path.Join(ruleDir, rule.File.Name))
		if err != nil {
			return
		}
	}
	return
}

// addRuleSetRepository clones a ruleset's associated git repository.
func (r *Rules) addRuleSetRepository(ruleSet *api.RuleSet) (err error) {
	if ruleSet.Repository == nil {
		return
	}
	rootDir := path.Join(
		RuleDir,
		"rulesets",
		strconv.Itoa(int(ruleSet.ID)),
		"repository",
	)
	err = os.MkdirAll(rootDir, 0755)
	if err != nil {
		return
	}
	var identity *api.Identity
	if ruleSet.Identity != nil {
		identity, err = addon.Identity.Get(ruleSet.Identity.ID)
		if err != nil {
			return
		}
	}
	rp, sErr := scm.New(
		rootDir,
		*ruleSet.Repository,
		identity)
	if sErr != nil {
		err = sErr
		return
	}
	err = rp.Fetch()
	if err != nil {
		return
	}
	ruleDir := path.Join(rootDir, ruleSet.Repository.Path)
	r.rules = append(r.rules, ruleDir)
	return
}

// addRepository clones a custom rule repository.
func (r *Rules) addRepository() (err error) {
	if r.Repository == nil {
		return
	}

	rootDir := path.Join(RuleDir, "repository")
	err = os.MkdirAll(rootDir, 0755)
	if err != nil {
		return
	}

	var identity *api.Identity
	if r.Identity != nil {
		identity, err = addon.Identity.Get(r.Identity.ID)
		if err != nil {
			return
		}
	}

	rp, sErr := scm.New(
		rootDir,
		*r.Repository,
		identity,
	)

	if sErr != nil {
		err = sErr
		return
	}
	err = rp.Fetch()
	if err != nil {
		return
	}
	ruleDir := path.Join(rootDir, r.Repository.Path)
	r.rules = append(r.rules, ruleDir)
	return
}
