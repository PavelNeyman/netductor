package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/migrate"
)

// CF-18.1 + 18.3: migrate noun/verb with --json / -h and attention summary.

func runMigrate(args []string) {
	jsonOut := false
	var pos []string
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "-h", "--help", "help":
			fmt.Print(`netductor migrate — startup migration journal (CF-18)

  netductor migrate status [--json]
  netductor migrate run [--json]
  netductor migrate clear-failed [--json]

Flags:
  --json   machine-readable output
  -h       this help

Status includes attention=true when failed/running journal rows exist or any
builtin migration still reports Required (same gate as TG migrate buttons).
`)
			return
		default:
			if strings.HasPrefix(a, "-") {
				fmt.Fprintf(os.Stderr, "unknown flag: %s\n", a)
				os.Exit(2)
			}
			pos = append(pos, a)
		}
	}
	cmd := "status"
	if len(pos) > 0 {
		cmd = pos[0]
	}
	switch cmd {
	case "run":
		res, err := migrate.RunAll(context.Background())
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(map[string]any{
				"ran": res.Ran, "skipped": res.Skipped, "failed": res.Failed,
				"error": errString(err), "attention": migrate.NeedsOperatorAttention(),
			})
		} else {
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			fmt.Printf("ran=%v skipped=%v failed=%s attention=%v\n", res.Ran, res.Skipped, res.Failed, migrate.NeedsOperatorAttention())
		}
		if err != nil {
			os.Exit(1)
		}
	case "status", "list":
		j, err := migrate.LoadJournalPublic()
		if err != nil {
			if jsonOut {
				_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"error": err.Error(), "attention": true})
			} else {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(1)
		}
		pending, _ := migrate.PendingRequiredIDs(context.Background())
		attn := migrate.NeedsOperatorAttention()
		if jsonOut {
			type row struct {
				ID, Status, AppVersion, Error string
			}
			var rows []row
			for id, rec := range j {
				rows = append(rows, row{id, string(rec.Status), rec.AppVersion, rec.Error})
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(map[string]any{"attention": attn, "pending_required": pending, "journal": rows})
			return
		}
		fmt.Printf("attention=%v\n", attn)
		if len(pending) > 0 {
			fmt.Printf("pending_required=%s\n", strings.Join(pending, ","))
		}
		if len(j) == 0 {
			fmt.Println("(no migration journal)")
			return
		}
		for id, rec := range j {
			line := fmt.Sprintf("%s status=%s ver=%s", id, rec.Status, rec.AppVersion)
			if rec.Error != "" {
				line += " err=" + rec.Error
			}
			fmt.Println(line)
		}
	case "clear-failed", "reset-failed":
		if err := migrate.ClearFailedRecords(); err != nil {
			if jsonOut {
				_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"error": err.Error()})
			} else {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(1)
		}
		attn := migrate.NeedsOperatorAttention()
		if jsonOut {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": true, "attention": attn})
		} else {
			fmt.Println("cleared failed/running migration records")
			fmt.Printf("attention=%v\n", attn)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown migrate subcommand (status|run|clear-failed)")
		os.Exit(2)
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
