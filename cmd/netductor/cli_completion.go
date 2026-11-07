package main

import (
	"fmt"
	"os"
	"strings"
)

func runCompletion(args []string) {
	shell := "bash"
	if len(args) > 0 {
		shell = strings.ToLower(strings.TrimSpace(args[0]))
	}
	switch shell {
	case "bash":
		fmt.Print(bashCompletionScript())
	case "zsh":
		fmt.Print(zshCompletionScript())
	default:
		fmt.Fprintf(os.Stderr, "usage: netductor completion bash|zsh\n")
		os.Exit(2)
	}
}

func topLevelCmds() string {
	return strings.Join([]string{
		"install", "serve", "firewall", "host-audit", "hoster-audit", "vps-check", "vpscheck",
		"host-baseline", "cleanup", "cleanup-legacy", "tg-alerts", "servicenet", "service-net",
		"doctor", "status", "vpn", "services", "policy", "logs", "channel", "channels",
		"secondary", "inventory", "svc-paths", "api-public", "edge", "mtls", "nvr",
		"stack", "redirect-serve", "import-redirect", "domain", "fleet", "backup", "restore",
		"sites", "nodes", "ssh-hosts", "known-hosts", "migrate", "aptguard", "apt-guard",
		"job", "jobs", "addons", "addon", "git", "registry", "ci", "dns", "recovery", "recover",
		"self-install", "project", "projects", "build-host", "buildhost", "release", "update",
		"probe", "collect", "audit", "smoke", "opcatalog", "catalog-matrix", "uplink-mux", "tls",
		"version", "help", "completion",
	}, " ")
}

func bashCompletionScript() string {
	top := topLevelCmds()
	return `# bash completion for netductor (CF-18.2)
_netductor() {
  local cur prev
  if declare -F _init_completion >/dev/null 2>&1; then
    _init_completion || return
  else
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
  fi
  if [[ ${COMP_CWORD} -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "` + top + `" -- "$cur") )
    return
  fi
  case "${COMP_WORDS[1]}" in
    svc-paths) COMPREPLY=( $(compgen -W "status apply peer failover doctor" -- "$cur") ) ;;
    stack) COMPREPLY=( $(compgen -W "status apply schedule rollback preflight heal pin unpin promote watchdog --json --dry-run -h" -- "$cur") ) ;;
    secondary) COMPREPLY=( $(compgen -W "status list cmd upgrade join remove export --json -h" -- "$cur") ) ;;
    migrate) COMPREPLY=( $(compgen -W "status run clear-failed list --json -h --help" -- "$cur") ) ;;
    vpn) COMPREPLY=( $(compgen -W "list add remove status" -- "$cur") ) ;;
    edge) COMPREPLY=( $(compgen -W "list status enroll policy" -- "$cur") ) ;;
    firewall) COMPREPLY=( $(compgen -W "status apply" -- "$cur") ) ;;
    git) COMPREPLY=( $(compgen -W "status sync ship mirror" -- "$cur") ) ;;
    host-baseline) COMPREPLY=( $(compgen -W "apply status repair" -- "$cur") ) ;;
    completion) COMPREPLY=( $(compgen -W "bash zsh" -- "$cur") ) ;;
  esac
}
complete -F _netductor netductor
`
}

func zshCompletionScript() string {
	top := strings.ReplaceAll(topLevelCmds(), " ", "\n  ")
	return `#compdef netductor
_netductor() {
  local -a top
  top=(
  ` + top + `
  )
  if (( CURRENT == 2 )); then
    _describe 'command' top
    return
  fi
  case $words[2] in
    svc-paths) _values 'svc-paths' status apply peer failover doctor ;;
    stack) _values 'stack' status apply schedule rollback ;;
    secondary) _values 'secondary' status list cmd upgrade join remove ;;
    migrate) _values 'migrate' status run clear-failed list --json ;;
    vpn) _values 'vpn' list add remove status ;;
    edge) _values 'edge' list status enroll policy ;;
    firewall) _values 'firewall' status apply ;;
    git) _values 'git' status sync ship mirror ;;
    host-baseline) _values 'host-baseline' apply status repair ;;
    completion) _values 'shell' bash zsh ;;
  esac
}
_netductor
`
}

func installBashCompletionBestEffort() {
	dir := "/usr/share/bash-completion/completions"
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return
	}
	path := dir + "/netductor"
	if err := os.WriteFile(path, []byte(bashCompletionScript()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "bash completion install:", err)
		return
	}
	fmt.Fprintln(os.Stderr, "bash completion: installed", path)
}
