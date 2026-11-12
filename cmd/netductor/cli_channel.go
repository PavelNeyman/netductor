package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/channels"
)

func runChannel(args []string) {
	o := parseCLIOpts(args)
	if o.Help {
		fmt.Fprintln(os.Stderr, `usage:
  netductor channel status [--json]
  netductor channel history [--minutes N] [--id C2] [--json]
  netductor channel sample [--json]
  netductor channel probe [--bw] [--target host[:port]] [--json]
  netductor channel users [--json]`)
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
		if sp := channels.SparkRTT(24); sp != "" {
			fmt.Printf("rtt_spark_c2 %s\n", sp)
		}
		ur := channels.CollectUsers()
		if !ur.Available {
			fmt.Printf("users: %s\n", ur.Note)
		} else {
			fmt.Print(channels.FormatUsersText(ur))
		}
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
			if strings.HasPrefix(a, "--minutes=") {
				if n, err := strconv.Atoi(a[len("--minutes="):]); err == nil && n > 0 {
					mins = n
				}
			}
			if strings.HasPrefix(a, "--id=") {
				idFilter = a[len("--id="):]
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
		if sp := channels.Sparkline(channels.HistorySeries(pts, idFilter, "", "rtt", 48)); sp != "" {
			fmt.Printf("spark %s\n", sp)
		}
	case "probe":
		withBW := false
		target := ""
		for i := 0; i < len(o.Rest); i++ {
			a := o.Rest[i]
			if a == "--bw" {
				withBW = true
			}
			if a == "--target" && i+1 < len(o.Rest) {
				i++
				target = o.Rest[i]
			}
			if strings.HasPrefix(a, "--target=") {
				target = a[len("--target="):]
			}
		}
		var list []channels.ProbeResult
		if target != "" {
			host, port := channels.ParseProbeTarget(target)
			if withBW {
				list = []channels.ProbeResult{channels.ProbeBW(host, port, 2 * time.Second)}
			} else {
				list = []channels.ProbeResult{channels.ProbeTCP(host, port, 3 * time.Second)}
			}
		} else {
			list = channels.ProbeSecondaries(withBW)
			if len(list) == 0 && withBW {
				list = []channels.ProbeResult{channels.ProbeBW("", 443, 2 * time.Second)}
			}
		}
		if o.JSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(list)
			return
		}
		fmt.Print(channels.FormatProbeText(list))
	case "users":
		ur := channels.CollectUsers()
		if o.JSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(ur)
			return
		}
		fmt.Print(channels.FormatUsersText(ur))
	default:
		fmt.Fprintln(os.Stderr, "usage: netductor channel status|sample|history|probe|users [--json]")
		os.Exit(2)
	}
}
