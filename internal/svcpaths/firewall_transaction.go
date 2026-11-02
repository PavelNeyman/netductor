package svcpaths

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/PavelNeyman/netductor/internal/firewall"
)

// firewallMutation applies one narrowly scoped peer rule and supplies an inverse
// only when this transaction actually created the rule.
type firewallMutation struct {
	apply    func() (bool, error)
	rollback func() error
	snapshot func() (bool, error)
	image    bootstrapFirewallImage
}

func applyFirewallMutations(mutations []firewallMutation, persist func() error) error {
	return applyFirewallMutationsWithTransaction(nil, mutations, persist)
}

func applyFirewallMutationsWithTransaction(tx *bootstrapFileTransaction, mutations []firewallMutation, persist func() error) error {
	if tx != nil {
		for i, mutation := range mutations {
			if mutation.snapshot == nil {
				return fmt.Errorf("firewall mutation %d has no durable before-image inspector", i)
			}
			exists, err := mutation.snapshot()
			if err != nil { return fmt.Errorf("inspect firewall mutation %d before-image: %w", i, err) }
			if err := tx.SnapshotFirewallRule(mutation.image, exists); err != nil {
				return fmt.Errorf("journal firewall mutation %d: %w", i, err)
			}
		}
	}
	return applyFirewallMutationsCore(mutations, persist)
}

func applyFirewallMutationsCore(mutations []firewallMutation, persist func() error) error {
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
			rollbackErr := rollback()
			// A failed save may have partially replaced the persistent ruleset.
			// Persist the restored runtime state as well; report both failures.
			restorePersistErr := persist()
			if restorePersistErr != nil {
				restorePersistErr = fmt.Errorf("persist restored firewall state: %w", restorePersistErr)
			}
			return errors.Join(fmt.Errorf("persist firewall changes: %w", err), rollbackErr, restorePersistErr)
		}
	}
	return nil
}

// ufwStatusHasPeerRule matches one complete numbered UFW rule. Matching exact
// fields prevents a different source address, port, action, or comment from
// being mistaken for the peer rule being managed.
func ufwStatusHasPeerRule(output, source string, port int, comment string) bool {
	wantPort := strconv.Itoa(port) + "/tcp"
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 8 && fields[0] == "[" && strings.HasSuffix(fields[1], "]") {
			if _, err := strconv.Atoi(strings.TrimSuffix(fields[1], "]")); err != nil { continue }
			fields = fields[2:]
		} else if len(fields) == 7 && strings.HasPrefix(fields[0], "[") && strings.HasSuffix(fields[0], "]") {
			if _, err := strconv.Atoi(strings.Trim(fields[0], "[]")); err != nil { continue }
			fields = fields[1:]
		} else {
			continue
		}
		if len(fields) == 6 && fields[0] == wantPort && fields[1] == "ALLOW" &&
			fields[2] == "IN" && fields[3] == source && fields[4] == "#" && fields[5] == comment {
			return true
		}
	}
	return false
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
		image: bootstrapFirewallImage{Backend: "ufw", Source: source, Port: port, Comment: comment},
		snapshot: func() (bool, error) {
			status, err := exec.Command("ufw", "status", "numbered").CombinedOutput()
			if err != nil { return false, fmt.Errorf("inspect UFW before-image: %w: %s", err, strings.TrimSpace(string(status))) }
			return ufwStatusHasPeerRule(string(status), source, port, comment), nil
		},
		apply: func() (bool, error) {
			status, err := exec.Command("ufw", "status", "numbered").CombinedOutput()
			if err != nil { return false, fmt.Errorf("inspect UFW rules: %w: %s", err, strings.TrimSpace(string(status))) }
			if ufwStatusHasPeerRule(string(status), source, port, comment) { return false, nil }
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
		image: bootstrapFirewallImage{Backend: "iptables", Chain: chain, Source: source, Port: port},
		snapshot: func() (bool, error) {
			out, err := exec.Command("iptables", check...).CombinedOutput()
			if err == nil { return true, nil }
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 { return false, nil }
			return false, fmt.Errorf("inspect iptables before-image: %w: %s", err, strings.TrimSpace(string(out)))
		},
		apply: func() (bool, error) {
			checkOut, checkErr := exec.Command("iptables", check...).CombinedOutput()
			if checkErr == nil { return false, nil }
			var exitErr *exec.ExitError
			if !errors.As(checkErr, &exitErr) || exitErr.ExitCode() != 1 {
				return false, fmt.Errorf("inspect iptables peer rule: %w: %s", checkErr, strings.TrimSpace(string(checkOut)))
			}
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
	return parseNetductorInputChain(string(out))
}

func parseNetductorInputChain(output string) (string, error) {
	var chain string
	count := 0
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 4 && fields[0] == "-A" && fields[1] == "INPUT" && fields[2] == "-j" &&
			(fields[3] == "NETDUCTOR_A" || fields[3] == "NETDUCTOR_B") {
			count++
			chain = fields[3]
		}
	}
	if count != 1 { return "", fmt.Errorf("expected exactly one Netductor INPUT jump, found %d", count) }
	return chain, nil
}

var bootstrapFirewallLock = firewall.AcquireMutationLock
var bootstrapFirewallExec = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func restoreBootstrapFirewallRules(images []bootstrapFirewallImage) error {
	needed := false
	for _, image := range images { if !image.BeforeExists { needed = true; break } }
	if !needed { return nil }
	unlock, err := bootstrapFirewallLock()
	if err != nil { return fmt.Errorf("acquire firewall lock for bootstrap rollback: %w", err) }
	defer unlock()
	var errs []error
	iptablesNeedsPersist := false
	for i := len(images)-1; i >= 0; i-- {
		rule := images[i]
		if rule.BeforeExists { continue }
		switch rule.Backend {
		case "ufw":
			status, err := bootstrapFirewallExec("ufw", "status", "numbered")
			if err != nil {
				errs = append(errs, fmt.Errorf("inspect UFW rollback rule %s:%d: %w: %s", rule.Source, rule.Port, err, strings.TrimSpace(string(status))))
				continue
			}
			if !ufwStatusHasPeerRule(string(status), rule.Source, rule.Port, rule.Comment) { continue }
			args := []string{"--force", "delete", "allow", "from", rule.Source, "to", "any", "port", strconv.Itoa(rule.Port), "proto", "tcp", "comment", rule.Comment}
			out, err := bootstrapFirewallExec("ufw", args...)
			if err != nil {
				errs = append(errs, fmt.Errorf("remove transaction-created UFW rule %s:%d: %w: %s", rule.Source, rule.Port, err, strings.TrimSpace(string(out))))
				continue
			}
			verify, err := bootstrapFirewallExec("ufw", "status", "numbered")
			if err != nil || ufwStatusHasPeerRule(string(verify), rule.Source, rule.Port, rule.Comment) {
				errs = append(errs, fmt.Errorf("verify removal of transaction-created UFW rule %s:%d (err=%v): %s", rule.Source, rule.Port, err, strings.TrimSpace(string(verify))))
			}
		case "iptables":
			iptablesNeedsPersist = true
			check := []string{"-C", rule.Chain, "-p", "tcp", "-s", rule.Source, "--dport", strconv.Itoa(rule.Port), "-j", "ACCEPT"}
			out, err := bootstrapFirewallExec("iptables", check...)
			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 { continue }
				errs = append(errs, fmt.Errorf("inspect transaction-created iptables rule %s:%d: %w: %s", rule.Source, rule.Port, err, strings.TrimSpace(string(out))))
				continue
			}
			remove := []string{"-D", rule.Chain, "-p", "tcp", "-s", rule.Source, "--dport", strconv.Itoa(rule.Port), "-j", "ACCEPT"}
			out, err = bootstrapFirewallExec("iptables", remove...)
			if err != nil {
				errs = append(errs, fmt.Errorf("remove transaction-created iptables rule %s:%d: %w: %s", rule.Source, rule.Port, err, strings.TrimSpace(string(out))))
				continue
			}
			// The runtime rule is now absent; persist even if an earlier
			// failed save left the on-disk ruleset stale.
		default:
			errs = append(errs, fmt.Errorf("unsupported firewall backend %q in rollback journal", rule.Backend))
		}
	}
	if iptablesNeedsPersist {
		if err := persistNetfilterRules(); err != nil { errs = append(errs, fmt.Errorf("persist iptables rollback: %w", err)) }
	}
	return errors.Join(errs...)
}
