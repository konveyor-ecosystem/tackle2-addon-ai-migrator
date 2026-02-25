package main

import (
	"errors"
	"path"
	"strings"

	"github.com/konveyor/tackle2-hub/shared/addon/scm"
	"github.com/konveyor/tackle2-hub/shared/api"
)

// FetchRepository gets SCM repository.
func FetchRepository(application *api.Application) (err error) {
	if application.Repository == nil {
		err = errors.New("application repository not defined")
		return
	}
	identity, _, err :=
		addon.Application.Identity(application.ID).Search().
			Direct("source").
			Indirect("source").
			Find()
	if err != nil {
		return
	}
	SourceDir = path.Join(
		SourceDir,
		strings.Split(
			path.Base(
				application.Repository.URL),
			".")[0])
	var rp scm.SCM
	rp, err = scm.New(
		SourceDir,
		*application.Repository,
		identity)
	if err != nil {
		return
	}
	err = rp.Fetch()
	return
}
