package main

import (
	"os"
	"path"

	hub "github.com/konveyor/tackle2-hub/shared/addon"
	// "github.com/konveyor/tackle2-hub/shared/addon/scm"
)

var (
	addon     = hub.Addon
	Dir       = ""
	SourceDir = ""
	Source    = "Discovery"
)

type Data struct {
	// Repository scm.SCM
	Source string
}

func init() {
	Dir, _ = os.Getwd()
	SourceDir = path.Join(Dir, "source")
}

func main() {
	addon.Run(func() (err error) {
		d := &Data{}
		err = addon.DataWith(d)
		if err != nil {
			return
		}
		if d.Source == "" {
			d.Source = Source
		}
		//
		// Fetch application.
		addon.Activity("Fetching application.")
		application, err := addon.Task.Application()
		if err != nil {
			return
		}

		// Failing for now after multiple runs
		// err = FetchRepository(application)
		// if err != nil {
		// 	return
		// }

		// Old code
		// err = Tag(application, d.Source)
		// if err != nil {
		// 	return
		// }

		// Just print out the data and application for now.
		addon.Log.Info("[test]", "data", d)
		addon.Log.Info("[test]", "application", application)

		return
	})
}
