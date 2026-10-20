package main

import "net/http"

func registerEdgeAPI(mux *http.ServeMux) {
	registerEdgeMeta(mux)
	registerEdgeTemplates(mux)
	registerEdgeEnroll(mux)
	registerEdgeStorage(mux)
	registerEdgeAgent(mux)
	registerEdgeAdmin(mux)
}
