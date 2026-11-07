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
		fmt.Print(opBashCompletion())
	case "zsh":
		fmt.Print(opZshCompletion())
	default:
		fmt.Fprintf(os.Stderr, "usage: netductor-op completion bash|zsh\n")
		os.Exit(2)
	}
}

func opTop() string {
	return strings.Join([]string{
		"deploy", "serve", "credentials", "tunnel", "session", "release", "project",
		"update", "vps-check", "vpscheck", "ssh-trust", "ssh-hosts-trust", "ssh-hosts",
		"tui", "menu", "job", "jobs", "version", "help", "completion",
	}, " ")
}

func opBashCompletion() string {
	top := opTop()
	return `# bash completion for netductor-op (CF-18.2)
_netductor_op() {
  local cur
  if declare -F _init_completion >/dev/null 2>&1; then
    _init_completion || return
  else
    cur="${COMP_WORDS[COMP_CWORD]}"
  fi
  if [[ ${COMP_CWORD} -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "` + top + `" -- "$cur") )
    return
  fi
  case "${COMP_WORDS[1]}" in
    deploy) COMPREPLY=( $(compgen -W "primary secondary fleet edge build-host buildhost" -- "$cur") ) ;;
    release) COMPREPLY=( $(compgen -W "list-local tags mirror-fetch build" -- "$cur") ) ;;
    ssh-hosts) COMPREPLY=( $(compgen -W "list show trust forget" -- "$cur") ) ;;
    completion) COMPREPLY=( $(compgen -W "bash zsh" -- "$cur") ) ;;
  esac
}
complete -F _netductor_op netductor-op
`
}

func opZshCompletion() string {
	top := strings.ReplaceAll(opTop(), " ", "\n  ")
	return `#compdef netductor-op
_netductor_op() {
  local -a top
  top=(
  ` + top + `
  )
  if (( CURRENT == 2 )); then
    _describe 'command' top
    return
  fi
  case $words[2] in
    deploy) _values 'target' primary secondary fleet edge build-host buildhost ;;
    release) _values 'release' list-local tags mirror-fetch build ;;
    ssh-hosts) _values 'ssh-hosts' list show trust forget ;;
    completion) _values 'shell' bash zsh ;;
  esac
}
_netductor_op
`
}
