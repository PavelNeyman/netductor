package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PavelNeyman/netductor/internal/notify"
)

func rememberFromChatMember(m *chatMemberUpd) {
	if m == nil {
		return
	}
	st := ""
	if m.NewChatMember != nil {
		st = m.NewChatMember.Status
	}
	// only remember when bot is (or becomes) admin/member of channel/group
	if st != "" && st != "administrator" && st != "member" && st != "creator" {
		return
	}
	notify.RememberChat(m.Chat.ID, m.Chat.Title, m.Chat.Type)
}

func rememberFromMessage(m *message) {
	if m == nil {
		return
	}
	notify.RememberChat(m.Chat.ID, m.Chat.Title, m.Chat.Type)
	if m.ForwardFromChat != nil && m.ForwardFromChat.ID != 0 {
		notify.RememberChat(m.ForwardFromChat.ID, m.ForwardFromChat.Title, m.ForwardFromChat.Type)
	}
	if m.ForwardOrigin != nil {
		if m.ForwardOrigin.Chat != nil && m.ForwardOrigin.Chat.ID != 0 {
			notify.RememberChat(m.ForwardOrigin.Chat.ID, m.ForwardOrigin.Chat.Title, m.ForwardOrigin.Chat.Type)
		}
		if m.ForwardOrigin.SenderChat != nil && m.ForwardOrigin.SenderChat.ID != 0 {
			notify.RememberChat(m.ForwardOrigin.SenderChat.ID, m.ForwardOrigin.SenderChat.Title, m.ForwardOrigin.SenderChat.Type)
		}
	}
}

func extractForwardedChannelID(m *message) (int64, string) {
	if m == nil {
		return 0, ""
	}
	if m.ForwardFromChat != nil && m.ForwardFromChat.ID != 0 {
		return m.ForwardFromChat.ID, m.ForwardFromChat.Title
	}
	if m.ForwardOrigin != nil {
		if m.ForwardOrigin.Chat != nil && m.ForwardOrigin.Chat.ID != 0 {
			return m.ForwardOrigin.Chat.ID, m.ForwardOrigin.Chat.Title
		}
		if m.ForwardOrigin.SenderChat != nil && m.ForwardOrigin.SenderChat.ID != 0 {
			return m.ForwardOrigin.SenderChat.ID, m.ForwardOrigin.SenderChat.Title
		}
	}
	return 0, ""
}

func alertsChatPromptHTML(ru bool) string {
	st := notify.GetAlertsRoutingStatus()
	known := notify.ListKnownChats()
	if ru {
		var b strings.Builder
		b.WriteString("📢 <b>Канал алертов</b>\n\n")
		if st.AlertsChatID != "" {
			b.WriteString("Сейчас: <code>" + st.AlertsChatID + "</code>\n\n")
		} else {
			b.WriteString("Сейчас: <i>не задан</i> (алерты в топиках/личке)\n\n")
		}
		b.WriteString("Как привязать:\n")
		b.WriteString("1. Добавьте бота <b>админом</b> канала\n")
		b.WriteString("2. Нажмите «Задать» и <b>перешлите</b> любой пост из канала сюда\n")
		b.WriteString("   <i>или</i> пришлите числовой id (−100…)\n")
		b.WriteString("3. Либо выберите канал из списка ниже (если бот его уже видел)\n\n")
		b.WriteString("<i>Бот не знает все каналы «сам» — только те, куда его добавили или откуда переслали пост.</i>\n")
		if len(known) > 0 {
			b.WriteString("\n<b>Известные чаты:</b>\n")
			for i, k := range known {
				if i >= 8 {
					break
				}
				title := k.Title
				if title == "" {
					title = k.Type
				}
				if title == "" {
					title = "chat"
				}
				b.WriteString(fmt.Sprintf("• %s <code>%d</code>\n", esc(title), k.ID))
			}
		}
		return b.String()
	}
	var b strings.Builder
	b.WriteString("📢 <b>Alerts channel</b>\n\n")
	if st.AlertsChatID != "" {
		b.WriteString("Current: <code>" + st.AlertsChatID + "</code>\n\n")
	} else {
		b.WriteString("Current: <i>not set</i> (alerts → topics/DM)\n\n")
	}
	b.WriteString("How to bind:\n")
	b.WriteString("1. Add the bot as <b>channel admin</b>\n")
	b.WriteString("2. Tap Set and <b>forward</b> any post from the channel here\n")
	b.WriteString("   <i>or</i> send the numeric id (−100…)\n")
	b.WriteString("3. Or pick a known chat below (if the bot has seen it)\n\n")
	b.WriteString("<i>Bots do not list all admin channels — only chats they received updates from.</i>\n")
	if len(known) > 0 {
		b.WriteString("\n<b>Known chats:</b>\n")
		for i, k := range known {
			if i >= 8 {
				break
			}
			title := k.Title
			if title == "" {
				title = k.Type
			}
			if title == "" {
				title = "chat"
			}
			b.WriteString(fmt.Sprintf("• %s <code>%d</code>\n", esc(title), k.ID))
		}
	}
	return b.String()
}

func alertsChatKeyboard() map[string]any {
	ru := getLang() != "en"
	setL, clearL, testL, refreshL, menu := "✏️ Set", "🗑 Clear", "🔔 Test", "🔄 Refresh", T("main_menu")
	if ru {
		setL, clearL, testL, refreshL = "✏️ Задать", "🗑 Сбросить", "🔔 Тест", "🔄 Обновить"
	}
	rows := [][]map[string]any{
		{btn(setL, "m:alerts-chat:set", "primary"), btn(testL, "m:alerts-chat:test", "")},
		{btn(clearL, "m:alerts-chat:clear", ""), btn(refreshL, "m:alerts-chat", "")},
	}
	for i, k := range notify.ListKnownChats() {
		if i >= 6 {
			break
		}
		label := k.Title
		if label == "" {
			label = strconv.FormatInt(k.ID, 10)
		}
		if len(label) > 28 {
			label = label[:28] + "…"
		}
		rows = append(rows, []map[string]any{
			btn("✓ "+label, "m:alerts-chat:pick:"+strconv.FormatInt(k.ID, 10), ""),
		})
	}
	rows = append(rows, []map[string]any{btn(menu, "m:menu", "")})
	return map[string]any{"inline_keyboard": rows}
}

func applyAlertsChatID(token string, chat int64, msgID int, idStr string) {
	id, err := notify.ParseChatID(idStr)
	if err != nil {
		reply(token, chat, msgID, "❌ "+esc(err.Error()), alertsChatKeyboard())
		return
	}
	if err := notify.SetAlertsChatID(strconv.FormatInt(id, 10)); err != nil {
		reply(token, chat, msgID, "❌ "+esc(err.Error()), alertsChatKeyboard())
		return
	}
	notify.RememberChat(id, "", "channel")
	ok := "✅ alerts channel → <code>" + strconv.FormatInt(id, 10) + "</code>\n\n"
	reply(token, chat, msgID, ok+alertsChatPromptHTML(getLang() != "en"), alertsChatKeyboard())
}
