package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type tuiLang int

const (
	langEN tuiLang = iota
	langRU
)

func localeLooksRU(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "_")
	return strings.HasPrefix(s, "ru") || strings.Contains(s, "ru_ru") || strings.Contains(s, "russian")
}

// detectTUILang: ru if locale/system UI is Russian, else en.
func detectTUILang() tuiLang {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			continue
		}
		for _, part := range strings.Split(v, ":") {
			if localeLooksRU(part) {
				return langRU
			}
		}
		if localeLooksRU(v) {
			return langRU
		}
	}
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output(); err == nil {
			if localeLooksRU(string(out)) {
				return langRU
			}
		}
		if out, err := exec.Command("defaults", "read", "-g", "AppleLanguages").Output(); err == nil {
			if strings.Contains(strings.ToLower(string(out)), "ru") {
				return langRU
			}
		}
	}
	return langEN
}

type tuiL10n struct {
	AppTitle, Suggested, SelectMode, ModeHint string
	Output, Back, Quit, HelpBar, HelpMode string
	Wizard, Tools, ChangeMode string
	SuggestedMark string
}

func l10n(lang tuiLang) tuiL10n {
	if lang == langRU {
		return tuiL10n{
			AppTitle: "Netductor", Suggested: "Рекомендуется", SelectMode: "Выбор режима",
			ModeHint: "↑↓ / мышь · Enter · Ctrl+C выход · L язык",
			Output: "Вывод", Back: "Enter/Esc — назад", Quit: "Выход",
			HelpBar: "↑↓ выбор  Enter открыть  Esc/q назад  1–9 быстрый выбор  L язык  Ctrl+C выход",
			HelpMode: "Выберите режим · ничего не ставится до подтверждения",
			Wizard: "★ Мастер настройки…", Tools: "── Инструменты ──", ChangeMode: "Сменить режим…",
			SuggestedMark: "  ← рекомендуется",
		}
	}
	return tuiL10n{
		AppTitle: "Netductor", Suggested: "Suggested", SelectMode: "Select mode",
		ModeHint: "↑↓ / mouse · Enter · Ctrl+C quit · L language",
		Output: "Output", Back: "enter/esc back", Quit: "Quit",
		HelpBar: "↑↓ select  Enter open  Esc/q back  1–9 jump  L language  Ctrl+C quit",
		HelpMode: "Pick a mode · nothing installs until you confirm",
		Wizard: "★ Setup wizard…", Tools: "── Tools ──", ChangeMode: "Change mode…",
		SuggestedMark: "  ← suggested",
	}
}
