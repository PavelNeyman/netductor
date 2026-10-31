package main

import (
	"context"
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/migrate"
)

func runMigrate(args []string) {
	cmd := "status"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "run":
		res, err := migrate.RunAll(context.Background())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("ran=%v skipped=%v\n", res.Ran, res.Skipped)
	case "status", "list":
		j, err := migrate.LoadJournalPublic()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if len(j) == 0 {
			fmt.Println("(no migration journal)")
			return
		}
		for id, rec := range j {
			fmt.Printf("%s status=%s ver=%s err=%s\n", id, rec.Status, rec.AppVersion, rec.Error)
		}
	case "clear-failed", "reset-failed":
		if err := migrate.ClearFailedRecords(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("cleared failed/running migration records")
	case "help", "-h", "--help":
		fmt.Println("netductor migrate status|run|clear-failed")
	default:
		fmt.Fprintln(os.Stderr, "unknown migrate subcommand")
		os.Exit(2)
	}
}
