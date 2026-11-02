package svcpaths

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// firewallMutation applies one narrowly scoped peer rule and supplies an inverse
// only when this transaction actually created the rule.
type firewallMutation struct {
	apply    func() (bool, error)
	rollback func() error
}

func applyFirewallMutations(mutations []firewallMutation, persist func() error) error {
	applied := make([]firewallMutation, 0, len(mutations))
	rollback := func() error {
		var errs []error
		for i := len(applied)-1; i >= 0; i-- {
			if err := applied[i].rollback(); err != nil {
				errs = append(errs, fmt.Errorf("rollback firewall mutation %d: %w", i, err))
			}
		}
		return errors.Join(errs...)
	}
	for i, mutation := range mutations {
		changed, err := mutation.apply()
		if err != nil {
			return errors.Join(fmt.Errorf("apply firewall mutation %d: %w", i, err), rollback())
		}
		if changed {
			applied = append(applied, mutation)
		}
	}
	if persist != nil {
		if err := persist(); err != nil {
			return errors.Join(fmt.Errorf("persist firewall changes: %w", err), rollback())
		}
	}
	return nil
}

func ufwPeerRuleMutation(source string, port int, comment string) (firewallMutation, error) {
	ip := net.ParseIP(source)
	if ip == nil || ip.To4() == nil || port < 1 || port > 65535 ||
		(comment != "nd-wss-sp" && comment != "nd-wss-ps" && comment != "nd-svc-sp") {
		return firewallMutation{}, fmt.Errorf("invalid peer UFW rule tuple")
	}
	source = ip.To4().String()
	args := []string{"allow", "from", source, "to", "any", "port", strconv.Itoa(port), "proto", "tcp", "comment", comment}
	deleteArgs := append([]string{"--force", "delete"}, args...)
	return firewallMutation{
		apply: func() (bool, error) {
			status, err := exec.Command("ufw", "status", "numbered").CombinedOutput()
			if err != nil { return false, fmt.Errorf("inspect UFW rules: %w: %s", err, strings.TrimSpace(string(status))) }
			lineRE := regexp.MustCompile("(?m)^.*" + regexp.QuoteMeta(strconv.Itoa(port)+"/tcp") + ".*" + regexp.QuoteMeta(source) + ".*# " + regexp.QuoteMeta(comment) + "\\s*$")
			if lineRE.Match(status) { return false, nil }
			out, err := exec.Command("ufw", args...).CombinedOutput()
			if err != nil { return false, fmt.Errorf("add UFW rule: %w: %s", err, strings.TrimSpace(string(out))) }
			return true, nil
		},
		rollback: func() error {
			out, err := exec.Command("ufw", deleteArgs...).CombinedOutput()
			if err != nil { return fmt.Errorf("delete exact UFW rule: %w: %s", err, strings.TrimSpace(string(out))) }
			return nil
		},
	}, nil
}

func iptablesPeerRuleMutation(chain, source string, port int) (firewallMutation, error) {
	ip := net.ParseIP(source)
	if (chain != "NETDUCTOR_A" && chain != "NETDUCTOR_B") || ip == nil || ip.To4() == nil || port < 1 || port > 65535 {
		return firewallMutation{}, fmt.Errorf("invalid peer iptables rule tuple")
	}
	source = ip.To4().String()
	check := []string{"-C", chain, "-p", "tcp", "-s", source, "--dport", strconv.Itoa(port), "-j", "ACCEPT"}
	insert := []string{"-I", chain, "3", "-p", "tcp", "-s", source, "--dport", strconv.Itoa(port), "-j", "ACCEPT"}
	remove := []string{"-D", chain, "-p", "tcp", "-s", source, "--dport", strconv.Itoa(port), "-j", "ACCEPT"}
	return firewallMutation{
		apply: func() (bool, error) {
			if err := exec.Command("iptables", check...).Run(); err == nil { return false, nil }
			out, err := exec.Command("iptables", insert...).CombinedOutput()
			if err != nil { return false, fmt.Errorf("insert iptables peer rule: %w: %s", err, strings.TrimSpace(string(out))) }
			return true, nil
		},
		rollback: func() error {
			out, err := exec.Command("iptables", remove...).CombinedOutput()
			if err != nil { return fmt.Errorf("delete exact iptables peer rule: %w: %s", err, strings.TrimSpace(string(out))) }
			return nil
		},
	}, nil
}

func persistNetfilterRules() error {
	out, err := exec.Command("netfilter-persistent", "save").CombinedOutput()
	if err != nil { return fmt.Errorf("netfilter-persistent save: %w: %s", err, strings.TrimSpace(string(out))) }
	return nil
}

func netductorInputChain() (string, error) {
	out, err := exec.Command("iptables", "-S", "INPUT").CombinedOutput()
	if err != nil { return "", fmt.Errorf("inspect INPUT chain: %w: %s", err, strings.TrimSpace(string(out))) }
	seen := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 4 && fields[0] == "-A" && fields[1] == "INPUT" && fields[2] == "-j" &&
			(fields[3] == "NETDUCTOR_A" || fields[3] == "NETDUCTOR_B") { seen[fields[3]] = true }
	}
	if len(seen) != 1 { return "", fmt.Errorf("expected exactly one Netductor INPUT chain, found %d", len(seen)) }
	for chain := range seen { return chain, nil }
	return "", fmt.Errorf("Netductor INPUT chain unresolved")
}
