package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

func runProject(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor project jobs [--json] [--state S]
  netductor project job <id> [--json]
  netductor project job rm <id>
  netductor project next [name]
  netductor project ship <name> [--ref REF] [--tag vX]
  netductor project pause <id>
  netductor project resume <id>
  netductor project complete <id> --from <dir>
  netductor project job log <id> [--tail N]
  netductor project secret list|set|rm <project> [key] [value]`)
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
		if args[1] == "log" {
			if len(args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: project job log <id> [--tail N]")
				return 2
			}
			tail := 0
			for i := 3; i < len(args); i++ {
				if args[i] == "--tail" && i+1 < len(args) {
					fmt.Sscanf(args[i+1], "%d", &tail)
					i++
				}
			}
			max := 0
			if tail > 0 {
				max = tail
			}
			s, err := gitstore.ReadShipLog(args[2], max)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Print(s)
			if !strings.HasSuffix(s, "\n") {
				fmt.Println()
			}
			return 0
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
		fmt.Printf("id=%s project=%s state=%s vps_slot=%v tag=%s ref=%s cursor=%d reason=%s pause_req=%v\n",
			j.ID, j.Project, j.State, j.VpsSlot, j.Tag, j.Ref, j.Cursor, j.Reason, j.PauseRequested)
		for i, s := range j.Steps {
			mark := " "
			if i == j.Cursor {
				mark = "*"
			}
			fmt.Printf("  %s%d. %s\t%s\n", mark, i, s.Name, s.Status)
		}
		if j.PublishTarget != "" {
			fmt.Println("publish:", j.PublishTarget)
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
		fmt.Println(gitstore.NextProjectTag(name))
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
		if name == "netductor" {
			got, dst, logPath, err := ndupdate.Ship(ndupdate.ShipOpts{Tag: tag})
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
		if tag == "" {
			tag = gitstore.NextProjectTag(name)
			fmt.Fprintln(os.Stderr, "auto tag", tag)
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
	case "pause":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor project pause <id>")
			return 2
		}
		if err := gitstore.RequestShipPause(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("pause requested", args[1], "(applies after current step)")
		return 0
	case "resume":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor project resume <id>")
			return 2
		}
		job, out, err := gitstore.ResumeShip(args[1])
		fmt.Print(out)
		if job != nil {
			fmt.Printf("job id=%s state=%s\n", job.ID, job.State)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "complete":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor project complete <id> --from <dir>")
			return 2
		}
		id := args[1]
		from := ""
		for i := 2; i < len(args); i++ {
			if args[i] == "--from" && i+1 < len(args) {
				from = args[i+1]
				i++
			}
		}
		job, out, err := gitstore.CompleteShip(id, from)
		fmt.Print(out)
		if job != nil {
			fmt.Printf("job id=%s state=%s\n", job.ID, job.State)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "secret", "secrets":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: project secret list <proj> | set <proj> <KEY> <value> | rm <proj> <KEY>")
			return 2
		}
		switch args[1] {
		case "list", "ls":
			if len(args) < 3 {
				return 2
			}
			keys, err := gitstore.ListCISecretKeys(args[2])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			for _, k := range keys {
				fmt.Println(k)
			}
			return 0
		case "set":
			if len(args) < 5 {
				fmt.Fprintln(os.Stderr, "usage: project secret set <proj> KEY value")
				return 2
			}
			if err := gitstore.SetCISecret(args[2], args[3], args[4]); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println("ok", args[3])
			return 0
		case "rm", "delete":
			if len(args) < 4 {
				return 2
			}
			if err := gitstore.DeleteCISecret(args[2], args[3]); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println("removed", args[3])
			return 0
		default:
			return 2
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown project subcommand:", args[0])
		return 2
	}
}
