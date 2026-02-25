package main

import (
	"os"
	"path"

	hub "github.com/konveyor/tackle2-hub/shared/addon"
	"github.com/konveyor/tackle2-hub/shared/api"
	"github.com/konveyor/tackle2-hub/shared/env"
)

var (
	addon        = hub.Addon
	Dir          = ""
	SourceDir    = ""
	RuleDir      = ""
	WorkspaceDir = ""
)

// Data task data.
type Data struct {
	// Profile analysis profile reference.
	Profile    api.Ref `json:"profile"`
	SourceTech string  `json:"sourceTech"`
	TargetTech string  `json:"targetTech"`
	Rules      Rules   `json:"rules"`
}

func init() {
	Dir = env.Get(hub.EnvSharedDir, "/tmp/shared")
	SourceDir = path.Join(Dir, "source")
	RuleDir = path.Join(Dir, "rules")
	WorkspaceDir = path.Join(Dir, "workspace")
}

func main() {
	addon.Run(func() (err error) {
		d := &Data{}
		err = addon.DataWith(d)
		if err != nil {
			return
		}

		err = applyProfile(d)
		if err != nil {
			return
		}

		// Create directories.
		for _, dir := range []string{RuleDir} {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				return
			}
		}

		addon.Activity("Fetching application.")
		application, err := addon.Task.Application()
		if err != nil {
			return
		}

		addon.Activity("Fetching repository.")
		err = FetchRepository(application)
		if err != nil {
			return
		}

		addon.Activity("Fetching rules.")
		err = d.Rules.Build()
		if err != nil {
			return
		}

		addon.Activity("Running goose migration.")
		err = os.MkdirAll(WorkspaceDir, 0777)
		if err != nil {
			return
		}
		goose := &Goose{
			SourceTech:   d.SourceTech,
			TargetTech:   d.TargetTech,
			InputPath:    SourceDir,
			RulePaths:    d.Rules.Paths(),
			WorkspaceDir: WorkspaceDir,
		}
		err = goose.Run()
		if err != nil {
			return
		}

		addon.Activity("Uploading report.")
		err = uploadReport(application, goose)
		if err != nil {
			return
		}

		return
	})
}

// applyProfile fetches and applies an analysis profile when specified.
func applyProfile(d *Data) (err error) {
	if d.Profile.ID == 0 {
		return
	}

	p, err := addon.AnalysisProfile.Get(d.Profile.ID)
	if err != nil {
		return
	}

	addon.Activity(
		"Using profile (id=%d): %s",
		p.ID,
		p.Name,
	)

	d.Rules.With(&p.Rules)
	return
}

// uploadReport uploads the goose report and stores it as a fact.
func uploadReport(application *api.Application, g *Goose) (err error) {
	reportPath := g.ReportPath()
	if _, stErr := os.Stat(reportPath); stErr != nil {
		addon.Activity("No report file found at %s, skipping upload.", reportPath)
		return
	}
	f, err := addon.File.Post(reportPath)
	if err != nil {
		return
	}
	addon.Attach(f)

	facts := api.Map{
		"report": map[string]interface{}{
			"fileId": f.ID,
			"name":   f.Name,
		},
	}

	err = addon.Application.Select(application.ID).
		Fact.
		Source("ai-migrator").
		Replace(facts)
	if err != nil {
		return
	}
	addon.Activity("Report uploaded and facts updated.")
	return
}
