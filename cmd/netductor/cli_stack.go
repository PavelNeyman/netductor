package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/stack"
)

func runStack(args []string) {
	o := parseCLIOpts(args)
	if len(o.Rest) == 0 || o.Help || (len(o.Rest) > 0 && (o.Rest[0] == "help")) {
		fmt.Print(`netductor stack — release apply on this node (CF-18.3)

  netductor stack status [--json]
  netductor stack preflight [vX.Y.Z]
  netductor stack apply [vX.Y.Z] [--dry-run] [--chain]
  netductor stack schedule [vX.Y.Z]
  netductor stack rollback | heal | promote
  netductor stack pin [reason] | unpin
  netductor stack watchdog | watchdog-install
`)
		return
	}
	args = o.Rest
	switch args[0] {
	case "status":
		st := stack.Collect()
		if o.JSON || true { // status is always structured; --json kept for canon
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(st)
		}
	case "preflight":
		tag := ""
		if len(args) > 1 {
			tag = args[1]
		}
		pf := stack.Preflight(tag)
		fmt.Print(stack.FormatPreflight(pf))
		if !pf.OK {
			os.Exit(1)
		}
		return
	case "apply":
		tag := ""
		if len(args) > 1 {
			tag = args[1]
		}
		if o.DryRun {
			pf := stack.Preflight(tag)
			fmt.Print(stack.FormatPreflight(pf))
			if o.Chain {
				if steps, err := stack.ResolveApplyPath(tag); err == nil {
					fmt.Println("chain:", stack.FormatApplyPath(steps))
				}
			}
			fmt.Println("dry-run: stack apply not executed")
			if !pf.OK {
				os.Exit(1)
			}
			return
		}
		var err error
		if o.Chain {
			err = stack.ApplyChain(tag)
		} else {
			err = stack.Apply(tag)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		installBashCompletionBestEffort()
	case "rollback":
		if err := stack.Rollback(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "heal":
		if err := stack.HealVersion(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		st := stack.Collect()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(st)
	case "watchdog":
		stack.WatchdogOnce()
		st := stack.Collect()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(st)
	case "promote":
		if !stack.PromoteLastGood() {
			fmt.Println("promote: nothing to do (or refused dirty prev/pin)")
			return
		}
		fmt.Println("promoted")
		return
	case "schedule":
		tag := ""
		if len(args) > 1 {
			tag = args[1]
		}
		if err := stack.ScheduleApply(tag); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("scheduled", tag)
		return
	case "pin":
		reason := "manual"
		if len(args) > 1 {
			reason = strings.Join(args[1:], " ")
		}
		if err := stack.Pin(reason); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("stack pinned")
		return
	case "unpin":
		_ = stack.Unpin()
		fmt.Println("stack unpinned")
		return
	case "watchdog-install":
		if err := stack.InstallWatchdogTimer(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("stack watchdog timer enabled")
	default:
		fmt.Fprintln(os.Stderr, "unknown stack subcommand:", args[0])
		os.Exit(2)
	}
}
