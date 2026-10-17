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
	return formatJobsList(m, false)
}

func listOpJobsTextRunning() string {
	if err := ensureOpServe(); err != nil {
		return err.Error()
	}
	m, err := opGETJSON("/v1/jobs?status=running")
	if err != nil {
		return err.Error()
	}
	return formatJobsList(m, true)
}

func formatJobsList(m map[string]any, runningOnly bool) string {
	jobs, _ := m["jobs"].([]any)
	var b strings.Builder
	title := "Operator jobs (serve)"
	if runningOnly {
		title = "Running jobs"
	}
	b.WriteString(title + "\n")
	b.WriteString("Cancel: menu → Jobs cancel  or  netductor-op job cancel <id>\n\n")
	b.WriteString(fmt.Sprintf("count: %d\n", len(jobs)))
	for _, j := range jobs {
		jm, _ := j.(map[string]any)
		id, _ := jm["id"].(string)
		typ, _ := jm["type"].(string)
		st, _ := jm["status"].(string)
		b.WriteString(fmt.Sprintf("• [%s] %s  %s\n", st, typ, id))
		if e, _ := jm["error"].(string); e != "" {
			b.WriteString("  error: " + e + "\n")
		}
	}
	if len(jobs) == 0 {
		b.WriteString("(none)\n")
	}
	return b.String()
}

func cancelOpJobText(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "job id required"
	}
	if err := ensureOpServe(); err != nil {
		return err.Error()
	}
	m, err := opPOSTJSON("/v1/jobs/"+id+"/cancel", nil)
	if err != nil {
		return err.Error()
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return "cancel ok\n" + string(b)
}
