package main

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

func runSSHHostsTUI() {
	action := "list"
	kind := "all"
	id := ""
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("SSH known hosts (TOFU)").Description(
				"After MT/VPS reinstall, forget the old key then reconnect once from a trusted network.",
			),
			huh.NewSelect[string]().Title("Action").Options(
				huh.NewOption("List", "list"),
				huh.NewOption("Forget one", "forget"),
				huh.NewOption("Clear all", "clear"),
			).Value(&action),
			huh.NewSelect[string]().Title("Kind").Options(
				huh.NewOption("All", "all"),
				huh.NewOption("MikroTik", "mt"),
				huh.NewOption("Relay", "relay"),
			).Value(&kind),
			huh.NewInput().Title("ID to forget (host or host:port)").Value(&id),
		),
	).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil {
		return
	}
	switch action {
	case "list":
		runSSHHosts([]string{"list", "--kind", kind})
	case "forget":
		if id == "" {
			fmt.Println(errStyle.Render("id required"))
			return
		}
		runSSHHosts([]string{"forget", "--kind", kind, id})
	case "clear":
		ok := false
		cf := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().Title("Clear known hosts (" + kind + ")?").Affirmative("Yes").Negative("No").Value(&ok),
		)).WithTheme(huh.ThemeCharm())
		_ = cf.Run()
		if ok {
			runSSHHosts([]string{"clear", "--kind", kind})
		}
	}
}
