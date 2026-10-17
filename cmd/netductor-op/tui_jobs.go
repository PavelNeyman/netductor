package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func listOpJobsText() string {
	if err := ensureOpServe(); err != nil {
		return err.Error()
	}
	m, err := opGETJSON("/v1/jobs")
	if err != nil {
		return err.Error()
	}
	return formatJobsList(m)
}

func listOpJobsTextRunning() string {
	if err := ensureOpServe(); err != nil {
		return err.Error()
	}
	m, err := opGETJSON("/v1/jobs?status=running")
	if err != nil {
		return err.Error()
	}
	return formatJobsList(m)
}

func formatJobsList(m map[string]any) string {
	raw, _ := json.MarshalIndent(m, "", "  ")
	jobs, _ := m["jobs"].([]any)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("jobs: %d\n", len(jobs)))
	for _, j := range jobs {
		jm, _ := j.(map[string]any)
		b.WriteString(fmt.Sprintf("• %v  %v  %v\n", jm["id"], jm["type"], jm["status"]))
		if e, _ := jm["error"].(string); e != "" {
			b.WriteString("  error: " + e + "\n")
		}
	}
	if len(jobs) == 0 {
		b.WriteString("(none)\n")
	}
	b.WriteString("\n--- raw ---\n")
	b.Write(raw)
	b.WriteByte('\n')
	return b.String()
}
