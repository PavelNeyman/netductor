package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSONStatusSetsCodeBeforeBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/t", func(w http.ResponseWriter, r *http.Request) {
		writeJSONStatus(w, http.StatusConflict, map[string]any{"ok": false})
	})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/t", nil))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d want 409", rec.Code)
	}
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["ok"] != false {
		t.Fatalf("%v", m)
	}
}
