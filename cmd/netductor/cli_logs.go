package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/PavelNeyman/netductor/internal/logs"
)

func runLogs(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: netductor logs schedule|schedule set <h> <m> [utc|local]|rotate|export [--hours N]")
		os.Exit(2)
	}
	switch args[0] {
	case "schedule":
		if len(args) >= 2 && args[1] == "set" {
			if len(args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: netductor logs schedule set <hour> <minute> [utc|local]")
				os.Exit(2)
			}
			h, _ := strconv.Atoi(args[2])
			m, _ := strconv.Atoi(args[3])
			s := logs.LoadSchedule()
			s.Hour, s.Minute = h, m
			if len(args) >= 5 && args[4] == "local" {
				s.UTC = false
			} else {
				s.UTC = true
			}
			if err := logs.SaveSchedule(s); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("ok", logs.FormatSchedule(s))
			return
		}
		fmt.Println(logs.FormatSchedule(logs.LoadSchedule()))
	case "rotate":
		out, err := logs.Rotate()
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "export":
		hours := 1
		for i := 1; i < len(args); i++ {
			if args[i] == "--hours" && i+1 < len(args) {
				hours, _ = strconv.Atoi(args[i+1])
			}
		}
		path, err := logs.ExportHours(hours)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(path)
	default:
		fmt.Fprintln(os.Stderr, "usage: netductor logs schedule|rotate|export")
		os.Exit(2)
	}
}
