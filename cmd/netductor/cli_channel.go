package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/PavelNeyman/netductor/internal/channels"
)

func runChannel(args []string) {
	o := parseCLIOpts(args)
	if o.Help {
		fmt.Fprintln(os.Stderr, `usage:
  netductor channel status [--json]
  netductor channel history [--minutes N] [--id C2] [--json]
  netductor channel sample [--json]   # one sample + append history (T1)`)
		os.Exit(0)
	}
	sub := "status"
	if len(o.Rest) > 0 {
		sub = o.Rest[0]
	}
	switch sub {
	case "status", "health":
		r := channels.Collect()
		if o.JSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(r)
			return
		}
		fmt.Print(channels.FormatText(r))
	case "sample":
		pt, err := channels.SampleOnce()
		if err != nil {
			fmt.Fprintln(os.Stderr, "sample:", err)
			os.Exit(1)
		}
		if o.JSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(pt)
			return
		}
		fmt.Print(channels.FormatHistoryText([]channels.SamplePoint{pt}))
	case "history":
		mins := 60
		idFilter := ""
		for i := 0; i < len(o.Rest); i++ {
			a := o.Rest[i]
			if a == "--minutes" && i+1 < len(o.Rest) {
				i++
				if n, err := strconv.Atoi(o.Rest[i]); err == nil && n > 0 {
					mins = n
				}
			}
			if a == "--id" && i+1 < len(o.Rest) {
				i++
				idFilter = o.Rest[i]
			}
			if len(a) > 10 && a[:10] == "--minutes=" {
				if n, err := strconv.Atoi(a[10:]); err == nil && n > 0 {
					mins = n
				}
			}
			if len(a) > 5 && a[:5] == "--id=" {
				idFilter = a[5:]
			}
		}
		since := time.Now().UTC().Add(-time.Duration(mins) * time.Minute)
		pts, err := channels.ReadHistory(since, idFilter)
		if err != nil {
			fmt.Fprintln(os.Stderr, "history:", err)
			os.Exit(1)
		}
		if o.JSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(pts)
			return
		}
		fmt.Print(channels.FormatHistoryText(pts))
	default:
		fmt.Fprintln(os.Stderr, "usage: netductor channel status|sample|history [--json]")
		os.Exit(2)
	}
}
