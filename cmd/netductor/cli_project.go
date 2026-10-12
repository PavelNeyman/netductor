package main

import (
	"encoding/json"
	"fmt"
	"os"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

func runProject(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor project jobs [--json] [--state S]
  netductor project job <id> [--json]
  netductor project job rm <id>
  netductor project next <name>     # next auto tag (product/local)
  netductor project ship <name> …   # S1+ (not in this drop if stub)

Ship job store: PLAN-PROJECT-SHIP S0`)
		return 2
	}
	switch args[0] {
	case "jobs", "list":
		jobs, err := gitstore.ListShipJobs()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		stateFilter := ""
		asJSON := false
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--json":
				asJSON = true
			case "--state":
				if i+1 < len(args) {
					stateFilter = args[i+1]
					i++
				}
			}
		}
		if stateFilter != "" {
			var filtered []gitstore.ShipJob
			for _, j := range jobs {
				if j.State == stateFilter {
					filtered = append(filtered, j)
				}
			}
			jobs = filtered
		}
		if asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(jobs)
			return 0
		}
		fmt.Print(gitstore.FormatShipJobsText(jobs))
		c := gitstore.CountShipJobsByState()
		if len(c) > 0 {
			fmt.Printf("# counts:")
			for k, v := range c {
				fmt.Printf(" %s=%d", k, v)
			}
			fmt.Printf(" vps_slot_taken=%v\n", gitstore.VpsSlotTaken())
		}
		return 0
	case "job":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor project job <id> [--json] | job rm <id>")
			return 2
		}
		if args[1] == "rm" || args[1] == "delete" || args[1] == "remove" {
			if len(args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: netductor project job rm <id>")
				return 2
			}
			if err := gitstore.RemoveShipJob(args[2]); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println("removed", args[2])
			return 0
		}
		id := args[1]
		asJSON := false
		for _, a := range args[2:] {
			if a == "--json" {
				asJSON = true
			}
		}
		j, err := gitstore.GetShipJob(id)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(j)
			return 0
		}
		fmt.Printf("id=%s project=%s state=%s vps_slot=%v tag=%s ref=%s cursor=%d reason=%s\n",
			j.ID, j.Project, j.State, j.VpsSlot, j.Tag, j.Ref, j.Cursor, j.Reason)
		for i, s := range j.Steps {
			mark := " "
			if i == j.Cursor {
				mark = "*"
			}
			fmt.Printf("  %s%d. %s\t%s\n", mark, i, s.Name, s.Status)
		}
		if j.Error != "" {
			fmt.Println("error:", j.Error)
		}
		if j.WaitSince != "" {
			fmt.Println("wait_since:", j.WaitSince)
		}
		fmt.Println("updated:", j.UpdatedAt)
		return 0
	case "next":
		// Per-project next: for now product next when name is netductor; else v0.0.1 baseline
		name := ""
		if len(args) >= 2 {
			name = args[1]
		}
		if name == "" || name == "netductor" {
			t, err := ndupdate.NextReleaseTag()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println(t)
			return 0
		}
		// generic: look for tags under artifacts later; patch from 0.0.0
		fmt.Println("v0.0.1")
		return 0
	case "ship":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor project ship <name> [--ref REF] [--tag vX.Y.Z]")
			return 2
		}
		name := args[1]
		ref, tag := "", ""
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--ref":
				if i+1 < len(args) {
					ref = args[i+1]
					i++
				}
			case "--tag":
				if i+1 < len(args) {
					tag = args[i+1]
					i++
				}
			}
		}
		// netductor product matrix → update.Ship (S2 fold-in)
		if name == "netductor" {
			fmt.Fprintln(os.Stderr, "project ship netductor: use release path via update.Ship…")
			got, dst, logPath, err := ndupdate.Ship(ndupdate.ShipOpts{Tag: tag, SkipDarwin: false})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				if logPath != "" {
					fmt.Fprintln(os.Stderr, "log", logPath)
				}
				return 1
			}
			fmt.Println("ok", got, "→", dst)
			return 0
		}
		job, out, err := gitstore.ShipProject(gitstore.ShipOpts{Project: name, Ref: ref, Tag: tag})
		fmt.Print(out)
		if job != nil {
			fmt.Printf("job id=%s state=%s\n", job.ID, job.State)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "pause", "resume", "complete":
		fmt.Fprintln(os.Stderr, "project", args[0]+": implement in S3/S4. Job store: project jobs | project job rm")
		return 2
	default:
		fmt.Fprintln(os.Stderr, "unknown project subcommand:", args[0])
		return 2
	}
}
