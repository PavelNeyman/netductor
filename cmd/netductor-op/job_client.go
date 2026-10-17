package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func opBaseURL() string {
	return strings.TrimRight(envOr("NETDUCTOR_OP_URL", "http://127.0.0.1:7373"), "/")
}

func opToken() string {
	if t := envOr("NETDUCTOR_OP_TOKEN", os.Getenv("ND_TOKEN")); t != "" {
		return t
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(home, ".netductor", "operator_token"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func opHealthOK(base string) bool {
	c := &http.Client{Timeout: 800 * time.Millisecond}
	res, err := c.Get(base + "/v1/health")
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.StatusCode == 200
}

func opPOSTJSON(path string, body any) (map[string]any, error) {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, err := http.NewRequest(http.MethodPost, opBaseURL()+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if t := opToken(); t != "" {
		req.Header.Set("X-Netductor-Token", t)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if res.StatusCode >= 300 {
		errMsg, _ := m["error"].(string)
		if errMsg == "" {
			errMsg = string(b)
		}
		return m, fmt.Errorf("%s", errMsg)
	}
	return m, nil
}

func opGETJSON(path string) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, opBaseURL()+path, nil)
	if err != nil {
		return nil, err
	}
	if t := opToken(); t != "" {
		req.Header.Set("X-Netductor-Token", t)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if res.StatusCode >= 300 {
		return m, fmt.Errorf("%s", string(b))
	}
	return m, nil
}

// startAndWaitJob posts a job and polls until terminal; streams log to stderr for TUI capture.
func startAndWaitJob(typ string, params map[string]any) (string, error) {
	if err := ensureOpServe(); err != nil {
		return "", err
	}
	started, err := opPOSTJSON("/v1/jobs", map[string]any{"type": typ, "params": params})
	if err != nil {
		return "", err
	}
	id, _ := started["id"].(string)
	if id == "" {
		return "", fmt.Errorf("job start: no id")
	}
	fmt.Fprintf(os.Stderr, "job %s id=%s\n", typ, id)
	var lastLog string
	for {
		j, err := opGETJSON("/v1/jobs/" + id)
		if err != nil {
			return "", err
		}
		if log, _ := j["log"].(string); log != "" && log != lastLog {
			// print only new suffix
			if strings.HasPrefix(log, lastLog) {
				fmt.Fprint(os.Stderr, log[len(lastLog):])
			} else {
				fmt.Fprint(os.Stderr, log)
			}
			if !strings.HasSuffix(log, "\n") {
				fmt.Fprintln(os.Stderr)
			}
			lastLog = log
		}
		if steps, ok := j["steps"].([]any); ok {
			for _, s := range steps {
				sm, _ := s.(map[string]any)
				id, _ := sm["id"].(string)
				msg, _ := sm["message"].(string)
				done, _ := sm["done"].(bool)
				errS, _ := sm["error"].(string)
				if errS != "" {
					fmt.Fprintf(os.Stderr, "step %s ERROR: %s\n", id, errS)
				} else if done {
					fmt.Fprintf(os.Stderr, "step %s done: %s\n", id, msg)
				} else if msg != "" {
					fmt.Fprintf(os.Stderr, "step %s: %s\n", id, msg)
				}
			}
		}
		st, _ := j["status"].(string)
		switch st {
		case "succeeded":
			return fmt.Sprintf("job %s ok", typ), nil
		case "failed", "cancelled", "interrupted":
			errMsg, _ := j["error"].(string)
			if errMsg == "" {
				errMsg = st
			}
			return "", fmt.Errorf("%s", errMsg)
		}
		time.Sleep(800 * time.Millisecond)
	}
}

func structToParams(v any) map[string]any {
	b, _ := json.Marshal(v)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m == nil {
		m = map[string]any{}
	}
	return m
}
