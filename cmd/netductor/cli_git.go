package main

import (
	"fmt"
	"github.com/PavelNeyman/netductor/internal/update"
	"os"
	"strconv"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
	"github.com/PavelNeyman/netductor/internal/hardening"
)

func runGit(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor git list
  netductor git init <name>
  netductor git log <name> [n]
  netductor git show <name> [rev]
  netductor git pipelines
  netductor git pipeline <repo> <name> [args...]
  netductor git delete <name>
  netductor git root
  netductor git artifacts [repo]
  netductor git artifact <rel-path>
  netductor git mirror-ensure [name] [upstream-url]
  netductor git mirror-fetch [name]
  netductor git tags [name]
  netductor git project list|add|rm|sync|build|queue|migrate-from-gh
    add <name> <url|org/repo> [--workflow path] [--pipeline name] [--build-on-fetch]
    sync <name> | build <name> [ref] | rm <name>`)
		return 2
	}
	switch args[0] {
	case "artifacts":
		repo := ""
		if len(args) >= 2 {
			repo = args[1]
		}
		list, err := gitstore.ListArtifacts(repo)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, n := range list {
			fmt.Println(n)
		}
		return 0
	case "artifact":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git artifact <rel-path>")
			return 2
		}
		out, err := gitstore.ReadArtifact(args[1])
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "root":
		fmt.Println(gitstore.Root())
		return 0
	case "project", "projects":
		return runGitProject(args[1:])
	case "mirror-ensure":
		name, up := "netductor", ""
		if len(args) >= 2 {
			name = args[1]
		}
		if len(args) >= 3 {
			up = args[2]
		}
		dir, err := gitstore.MirrorEnsure(name, up)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(dir)
		return 0
	case "mirror-fetch":
		name := "netductor"
		if len(args) >= 2 {
			name = args[1]
		}
		out, err := gitstore.MirrorFetch(name)
		if err == nil && (name == "netductor" || name == "") {
			_ = update.MaybeAutoBuildNewest()
		}
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "tags":
		name := "netductor"
		if len(args) >= 2 {
			name = args[1]
		}
		tags, err := gitstore.ListTags(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, t := range tags {
			fmt.Println(t)
		}
		return 0
	case "delete":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git delete <name>")
			return 2
		}
		if err := gitstore.Delete(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("deleted", args[1])
		return 0
	case "list":
		list, err := gitstore.List()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, n := range list {
			fmt.Println(n)
		}
		return 0
	case "init":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git init <name>")
			return 2
		}
		path, err := gitstore.Init(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(path)
		fmt.Fprintln(os.Stderr, "remote:", gitstore.RemoteHint(args[1], hardening.SSHPort()))
		return 0
	case "log":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git log <name> [n]")
			return 2
		}
		n := 20
		if len(args) >= 3 {
			n, _ = strconv.Atoi(args[2])
		}
		out, err := gitstore.Log(args[1], n)
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "show":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git show <name> [rev]")
			return 2
		}
		rev := "HEAD"
		if len(args) >= 3 {
			rev = args[2]
		}
		out, err := gitstore.Show(args[1], rev)
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "pipelines":
		_ = gitstore.EnsureSamplePipeline()
		list, err := gitstore.ListPipelines()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, n := range list {
			fmt.Println(n)
		}
		return 0
	case "workflow":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git workflow <repo> [path]")
			return 2
		}
		path := ""
		if len(args) > 2 {
			path = args[2]
		}
		out, err := gitstore.RunWorkflow(args[1], path)
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "pipeline":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor git pipeline <repo> <pipeline> [args...]")
			return 2
		}
		out, err := gitstore.RunPipeline(args[1], args[2], args[3:]...)
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown git subcommand")
		return 2
	}
}

func runGitProject(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: netductor git project list|add|rm|sync|sync-all|build|build-all|mirror-push|queue|timers|migrate-from-gh")
		return 2
	}
	switch args[0] {
	case "list", "ls":
		list, err := gitstore.LoadProjects()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, p := range list {
			fmt.Printf("%s\t%s\thost=%s\twf=%s\tpipe=%s\ton_fetch=%v\n", p.Name, p.Upstream, p.Host, p.Workflow, p.Pipeline, p.BuildOnFetch)
		}
		return 0
	case "add":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor git project add <name> <url|org/repo> [--workflow p] [--pipeline n] [--host vps|mac] [--build-on-fetch]")
			return 2
		}
		p := gitstore.Project{Name: args[1], Upstream: args[2]}
		for i := 3; i < len(args); i++ {
			switch args[i] {
			case "--workflow":
				if i+1 < len(args) {
					p.Workflow = args[i+1]
					i++
				}
			case "--pipeline":
				if i+1 < len(args) {
					p.Pipeline = args[i+1]
					i++
				}
			case "--build-on-fetch":
				p.BuildOnFetch = true
			case "--host":
				if i+1 < len(args) {
					p.Host = args[i+1]
					i++
				}
			}
		}
		if err := gitstore.AddProject(p, true); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("ok", p.Name, p.Upstream)
		return 0
	case "rm", "delete", "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git project rm <name>")
			return 2
		}
		if err := gitstore.RemoveProject(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("removed", args[1])
		return 0
	case "sync":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git project sync <name>")
			return 2
		}
		out, err := gitstore.SyncProject(args[1])
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "sync-all", "syncall":
		results, okN, failN := gitstore.SyncAll()
		for _, r := range results {
			status := "ok"
			if !r.OK {
				status = "FAIL"
			}
			fmt.Printf("%s\t%s\thost=%s\n", status, r.Name, r.Host)
			if r.Error != "" {
				fmt.Printf("  %s\n", r.Error)
			}
		}
		fmt.Printf("summary ok=%d fail=%d\n", okN, failN)
		if failN > 0 {
			return 1
		}
		return 0
	case "build":
		fmt.Fprintln(os.Stderr, "git project build removed — use: netductor project ship <name> [--ref REF] [--tag vX]")
		return 2
		// unreachable legacy:
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git project build <name> [ref]")
			return 2
		}
		ref := ""
		if len(args) >= 3 {
			ref = args[2]
		}
		out, err := gitstore.BuildProject(args[1], ref)
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "build-all", "buildall":
		vpsOnly := true
		for _, a := range args[1:] {
			if a == "--all-hosts" || a == "--include-mac" {
				vpsOnly = false
			}
		}
		results, okN, failN, skipN := gitstore.BuildAll(vpsOnly)
		for _, r := range results {
			status := "ok"
			if r.Skipped {
				status = "skip"
			} else if !r.OK {
				status = "FAIL"
			}
			fmt.Printf("%s	%s	host=%s\n", status, r.Name, r.Host)
			if r.Error != "" {
				fmt.Printf("  %s\n", r.Error)
			}
		}
		fmt.Printf("summary ok=%d fail=%d skip=%d\n", okN, failN, skipN)
		if failN > 0 {
			return 1
		}
		return 0
	case "queue":
		// list | done <id> | cancel <id>
		sub := "list"
		if len(args) >= 2 {
			sub = args[1]
		}
		switch sub {
		case "list", "ls", "":
			fmt.Println("== vps ==")
			for _, j := range gitstore.ListVpsQueue(false) {
				fmt.Printf("%s	%s	%s	%s	%s\n", j.ID, j.Status, j.Kind, j.Project, j.Ref)
			}
			fmt.Println("== mac ==")
			for _, j := range gitstore.ListMacQueue(false) {
				fmt.Printf("%s	%s	%s	%s	%s\n", j.ID, j.Status, j.Project, j.Ref, j.CLI)
			}
			return 0
		case "vps":
			for _, j := range gitstore.ListVpsQueue(false) {
				fmt.Printf("%s	%s	%s	%s	%s\n", j.ID, j.Status, j.Kind, j.Project, j.Ref)
				if j.Error != "" {
					fmt.Printf("  %s\n", j.Error)
				}
			}
			return 0
		case "pending":
			fmt.Println("== vps ==")
			for _, j := range gitstore.ListVpsQueue(true) {
				fmt.Printf("%s\t%s\t%s\t%s\n", j.ID, j.Status, j.Kind, j.Project)
			}
			fmt.Println("== mac ==")
			for _, j := range gitstore.ListMacQueue(true) {
				fmt.Printf("%s\t%s\t%s\t%s\n", j.ID, j.Project, j.Ref, j.CLI)
			}
			return 0
		case "done", "complete":
			if len(args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: netductor git project queue done <id>")
				return 2
			}
			if err := gitstore.CompleteMacBuild(args[2]); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println("done", args[2])
			return 0
		case "cancel":
			if len(args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: netductor git project queue cancel <id>")
				return 2
			}
			err := gitstore.CancelVpsJob(args[2])
			if err != nil {
				err = gitstore.CancelMacBuild(args[2])
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println("cancelled", args[2])
			return 0
		default:
			fmt.Fprintln(os.Stderr, "usage: netductor git project queue list|pending|done|cancel")
			return 2
		}
	case "mirror-push", "push-gh":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git project mirror-push <name>|all")
			return 2
		}
		if args[1] == "all" {
			list, err := gitstore.LoadProjects()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fail := 0
			for _, pr := range list {
				out, err := gitstore.MirrorPush(pr.Name)
				fmt.Printf("== %s ==\n%s", pr.Name, out)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					fail++
				}
			}
			if fail > 0 {
				return 1
			}
			return 0
		}
		out, err := gitstore.MirrorPush(args[1])
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "timers":
		if err := gitstore.EnsureGitOpsTimers(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("ok timers: sync-all (6h) + cleanup weekly")
		return 0
	case "migrate-from-gh", "migrate":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git project migrate-from-gh <owner> [--dry-run] [--sync]")
			return 2
		}
		opts := gitstore.MigrateOpts{Owner: args[1]}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--dry-run":
				opts.DryRun = true
			case "--sync":
				opts.Sync = true
			}
		}
		added, skipped, err := gitstore.MigrateFromGitHub(opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("added=%d skipped=%d dry_run=%v sync=%v\n", len(added), len(skipped), opts.DryRun, opts.Sync)
		for _, a := range added {
			fmt.Println("+", a)
		}
		for _, s := range skipped {
			fmt.Println("-", s)
		}
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown project subcommand")
		return 2
	}
}
