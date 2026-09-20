package main

// Core TUI — wizards in tui_deploy_wizards.go
// Temporary minimal restore so main builds; full UI follows in next commit.

import (
	"fmt"
	"os"
)

func tuiMainPlaceholder() {
	fmt.Fprintln(os.Stderr, "tui core loading")
}
