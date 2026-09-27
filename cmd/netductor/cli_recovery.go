package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/PavelNeyman/netductor/internal/secondary"
)

func runRecovery(args []string) {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, `usage:
  netductor recovery arm [--ttl 30m] [--detach]
  netductor recovery disarm
  netductor recovery status

Recovery HTTP :8790 is OFF until arm. Run on secondary over SSH, then recover from new primary.
Without --detach, arm blocks until TTL expires or SIGINT/SIGTERM (keeps the HTTP server alive).
Prefer pull from primary via service path: https://10.87.11.2:8790 (nd-svc-ps).
`)
		os.Exit(2)
	}
	switch strings.ToLower(args[0]) {
	case "arm":
		ttl := 30 * time.Minute
		detach := false
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--ttl":
				if i+1 >= len(args) {
					fmt.Fprintln(os.Stderr, "--ttl needs duration")
					os.Exit(1)
				}
				d, err := time.ParseDuration(args[i+1])
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				ttl = d
				i++
			case "--detach":
				detach = true
			}
		}
		if err := secondary.ArmRecovery(ttl); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		armed, until := secondary.RecoveryStatus()
		fmt.Printf("armed=%v until=%s\n", armed, until.Format(time.RFC3339))
		if detach {
			fmt.Fprintln(os.Stderr, "recovery-serve: --detach set; process will exit — server stops with process (use without --detach or under systemd)")
			return
		}
		// Keep process alive so ListenAndServe goroutine stays up.
		fmt.Fprintln(os.Stderr, "recovery-serve: blocking until TTL or signal (Ctrl-C disarms)")
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		timer := time.NewTimer(time.Until(until) + time.Second)
		defer timer.Stop()
		select {
		case <-timer.C:
			secondary.DisarmRecovery()
			fmt.Println("disarmed (ttl)")
		case s := <-sig:
			secondary.DisarmRecovery()
			fmt.Println("disarmed (", s, ")")
		}
	case "disarm", "off":
		secondary.DisarmRecovery()
		fmt.Println("disarmed")
	case "status":
		armed, until := secondary.RecoveryStatus()
		if !armed {
			fmt.Println("armed=false")
			return
		}
		fmt.Printf("armed=true until=%s\n", until.Format(time.RFC3339))
	default:
		fmt.Fprintln(os.Stderr, "unknown recovery subcommand (knock removed; use arm over SSH)")
		os.Exit(2)
	}
}
