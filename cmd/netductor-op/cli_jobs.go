package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func runJobsCLI(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage: netductor-op job list|get|wait|cancel <id>
  Uses local serve http://127.0.0.1:7373 (NETDUCTOR_OP_URL, NETDUCTOR_OP_TOKEN)`)
		os.Exit(2)
	}
	base := strings.TrimRight(envOr("NETDUCTOR_OP_URL", "http://127.0.0.1:7373"), "/")
	token := envOr("NETDUCTOR_OP_TOKEN", os.Getenv("ND_TOKEN"))
	cmd := args[0]
	switch cmd {
	case "list":
		b, err := opGET(base, token, "/v1/jobs?"+strings.Join(args[1:], "&"))
		mustJob(err)
		fmt.Println(string(b))
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job get <id>")
			os.Exit(2)
		}
		b, err := opGET(base, token, "/v1/jobs/"+args[1])
		mustJob(err)
		fmt.Println(string(b))
	case "wait":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job wait <id>")
			os.Exit(2)
		}
		id := args[1]
		for {
			b, err := opGET(base, token, "/v1/jobs/"+id)
			mustJob(err)
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			st, _ := m["status"].(string)
			fmt.Fprintf(os.Stderr, "status=%s\n", st)
			if st == "succeeded" || st == "failed" || st == "cancelled" || st == "interrupted" {
				fmt.Println(string(b))
				if st != "succeeded" {
					os.Exit(1)
				}
				return
			}
			time.Sleep(time.Second)
		}
	case "cancel":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job cancel <id>")
			os.Exit(2)
		}
		b, err := opPOST(base, token, "/v1/jobs/"+args[1]+"/cancel", nil)
		mustJob(err)
		fmt.Println(string(b))
	default:
		fmt.Fprintln(os.Stderr, "unknown job subcommand")
		os.Exit(2)
	}
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func mustJob(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func opGET(base, token, path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, base+path, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("X-Netductor-Token", token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return b, fmt.Errorf("HTTP %d: %s", res.StatusCode, string(b))
	}
	return b, nil
}

func opPOST(base, token, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, base+path, body)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("X-Netductor-Token", token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return b, fmt.Errorf("HTTP %d: %s", res.StatusCode, string(b))
	}
	return b, nil
}
