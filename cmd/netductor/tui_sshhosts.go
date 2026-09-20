package main

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

func runSSHHostsTUI() {
	lang := detectLang()
	action := "list"
	kind := "all"
	id := ""
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title(TT(lang, "SSH known hosts (TOFU)", "SSH known hosts (TOFU)")).Description(
				TT(lang,
					"After MT/VPS reinstall, forget the old key then reconnect once from a trusted network.",
					"После переустановки MT/VPS: forget старый ключ, затем один reconnect из доверенной сети.",
				),
			),
			huh.NewSelect[string]().Title(TT(lang, "Action", "Действие")).Options(
				huh.NewOption(TT(lang, "List", "Список"), "list"),
				huh.NewOption(TT(lang, "Forget one", "Забыть один"), "forget"),
				huh.NewOption(TT(lang, "Clear all", "Очистить все"), "clear"),
			).Value(&action),
			huh.NewSelect[string]().Title(TT(lang, "Kind", "Тип")).Options(
				huh.NewOption(TT(lang, "All", "Все"), "all"),
				huh.NewOption("MikroTik", "mt"),
				huh.NewOption("Secondary", "secondary"),
			).Value(&kind),
			huh.NewInput().Title(TT(lang, "ID to forget (host or host:port)", "ID для forget (host или host:port)")).Value(&id),
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
			fmt.Println(errStyle.Render(TT(lang, "id required", "нужен id")))
			return
		}
		runSSHHosts([]string{"forget", "--kind", kind, id})
	case "clear":
		ok := false
		cf := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().Title(TT(lang, "Clear known hosts ("+kind+")?", "Очистить known hosts ("+kind+")?")).
				Affirmative(TT(lang, "Yes", "Да")).Negative(TT(lang, "No", "Нет")).Value(&ok),
		)).WithTheme(huh.ThemeCharm())
		_ = cf.Run()
		if ok {
			runSSHHosts([]string{"clear", "--kind", kind})
		}
	}
}
