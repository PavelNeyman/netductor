package firewall

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// HosterAllowPatterns are common VPS panel / stock image ALLOW rules that block
// secondary baseline. Never delete without explicit operator confirm.
var HosterAllowPatterns = []string{
	"80/tcp",
	"ispmanager",
	"1500/tcp",
	"1500",
}

// ListHosterNoiseRules returns numbered UFW rule numbers that look like hoster noise
// (WAN ALLOW on restricted admin ports / panel profiles). Does not mutate.
func ListHosterNoiseRules() ([]int, []string, error) {
	out, err := exec.Command("ufw", "status", "numbered").CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("ufw status: %v (%s)", err, truncate(string(out), 160))
	}
	var nums []int
	var lines []string
	for _, rule := range parseUfwNumbered(string(out)) {
		if !ufwRuleAllowsTraffic(rule.Action) {
			continue
		}
		if !ufwFromIsWAN(rule.From) {
			continue
		}
		raw := strings.ToLower(rule.Raw)
		noise := false
		if ufwRuleMatchesPortProtocol(rule, "80", "tcp") {
			noise = true
		}
		if ufwRuleMatchesPortProtocol(rule, "1500", "tcp") {
			noise = true
		}
		if strings.Contains(raw, "ispmanager") {
			noise = true
		}
		// Never touch netductor-owned rules.
		if strings.Contains(raw, "netductor") {
			noise = false
		}
		if noise {
			nums = append(nums, rule.Num)
			lines = append(lines, strings.TrimSpace(rule.Raw))
		}
	}
	return nums, lines, nil
}

// PurgeHosterNoiseRules deletes hoster noise ALLOW rules. confirm must be true
// (CLI --confirm or env). Safe for multi-release jumps: only documented patterns.
func PurgeHosterNoiseRules(confirm bool) (deleted int, lines []string, err error) {
	if !confirm {
		return 0, nil, fmt.Errorf("refusing hoster purge without confirm (pass --confirm)")
	}
	for round := 0; round < 32; round++ {
		nums, desc, err := ListHosterNoiseRules()
		if err != nil {
			return deleted, lines, err
		}
		if len(nums) == 0 {
			return deleted, lines, nil
		}
		sort.Sort(sort.Reverse(sort.IntSlice(nums)))
		n := nums[0]
		out, delErr := exec.Command("ufw", "--force", "delete", strconv.Itoa(n)).CombinedOutput()
		if delErr != nil {
			return deleted, lines, fmt.Errorf("ufw delete %d: %v (%s)", n, delErr, truncate(string(out), 120))
		}
		deleted++
		if len(desc) > 0 {
			lines = append(lines, desc[0])
		}
	}
	return deleted, lines, fmt.Errorf("hoster purge: too many matching rules")
}
