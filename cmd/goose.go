package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/konveyor/tackle2-hub/shared/addon/command"
	"github.com/konveyor/tackle2-hub/shared/env"
)

var (
	GooseBin   = env.Get("GOOSE_BIN", "/usr/local/bin/goose")
	RecipePath = env.Get("RECIPE_PATH", "/opt/goose/recipes/migration.yaml")
)

// Goose wraps the goose CLI for recipe execution.

// TODO: Support custom prompts. The recipe's `prompt:` field handles the
// default prompt via --params, but --recipe and --text are mutually exclusive
// in goose. Custom prompts may require a wrapper recipe or --instructions.
type Goose struct {
	SourceTech   string
	TargetTech   string
	InputPath    string
	RulePaths    []string
	WorkspaceDir string
}

// Run executes the goose recipe.
func (g *Goose) Run() (err error) {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM)
	defer cancel()

	cmd := command.New(GooseBin)
	cmd.Options = g.options()
	cmd.Dir = g.WorkspaceDir

	// Wrap Begin to tee output to local stdout while preserving the hub writer
	// that Begin installs.
	origBegin := cmd.Begin
	cmd.Begin = func() (bErr error) {
		bErr = origBegin()
		if bErr != nil {
			return
		}
		cmd.Writer = io.MultiWriter(cmd.Writer, os.Stdout)
		return
	}
	err = cmd.RunWith(ctx)
	return
}

// options builds the command-line arguments for goose.
func (g *Goose) options() (options command.Options) {
	options = command.Options{
		"run",
		"--recipe",
		RecipePath,
	}
	options.Add("--params", fmt.Sprintf("source_tech=%s", g.SourceTech))
	options.Add("--params", fmt.Sprintf("target_tech=%s", g.TargetTech))
	options.Add("--params", fmt.Sprintf("input_path=%s", g.InputPath))
	options.Add("--params", fmt.Sprintf("workspace_dir=%s", g.WorkspaceDir))
	if len(g.RulePaths) > 0 {
		options.Add("--params",
			fmt.Sprintf("rules=%s", strings.Join(g.RulePaths, ",")))
	}
	return
}

// ReportPath returns the expected report output path.
func (g *Goose) ReportPath() string {
	return path.Join(g.WorkspaceDir, "report.html")
}
