package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/firewall"
)

func runFirewall(args []string) {
	if len(args) < 1 {
		fmt.Print(`netductor firewall status [--json]
netductor firewall apply [primary|secondary]
netductor firewall heal [--force]
netductor firewall purge-hoster [--confirm]   # list or delete stock hoster ALLOW 80/1500/ispmanager
`)
		return
	}
	switch args[0] {
	case "status":
		st := firewall.Collect("")
		if len(args) > 1 && args[1] == "--json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(st)
			return
		}
		fmt.Printf("backend=%s active=%v ok=%v role=%s\n", st.Backend, st.Active, st.OK, st.Role)
		fmt.Printf("open_wan: %v\n", st.OpenWAN)
		fmt.Printf("deny_wan: %v\n", st.DenyWAN)
		if len(st.Restricted) > 0 {
			fmt.Printf("restricted: %v\n", st.Restricted)
		}
		for _, w := range st.Warnings {
			fmt.Println("WARN:", w)
		}
		if st.Detail != "" {
			fmt.Println("---")
			fmt.Println(st.Detail)
		}
		if !st.OK {
			os.Exit(1)
		}
	case "apply":
		role := ""
		if len(args) > 1 {
			role = args[1]
		}
		if err := firewall.ApplyRole(role); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("firewall applied")
		st := firewall.Collect(role)
		fmt.Printf("backend=%s active=%v ok=%v\n", st.Backend, st.Active, st.OK)
	case "heal":
		force := len(args) > 1 && args[1] == "--force"
		healed, err := firewall.HealIfNeeded("", force)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if healed {
			fmt.Println("firewall healed")
		} else {
			fmt.Println("firewall ok or autoheal disabled (use --force or NETDUCTOR_FW_AUTOHEAL=1)")
		}
	case "purge-hoster":
		confirm := false
		for _, a := range args[1:] {
			if a == "--confirm" {
				confirm = true
			}
		}
		nums, desc, err := firewall.ListHosterNoiseRules()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if len(nums) == 0 {
			fmt.Println("no hoster noise ALLOW rules found")
			return
		}
		fmt.Println("candidate hoster noise rules:")
		for i, line := range desc {
			fmt.Printf("  #%d %s\n", nums[i], line)
		}
		if !confirm {
			fmt.Println("dry-run only; re-run with --confirm to delete (never uses ufw reset)")
			return
		}
		n, lines, err := firewall.PurgeHosterNoiseRules(true)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("deleted %d hoster noise rule(s)\n", n)
		for _, line := range lines {
			fmt.Println(" -", line)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown firewall subcommand")
		os.Exit(2)
	}
}
