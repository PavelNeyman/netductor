package main

import (
	"net/http"
)

func registerGitAPI(mux *http.ServeMux) {
	registerGitRepos(mux)
	registerGitProjects(mux)
	registerGitShip(mux)
}
