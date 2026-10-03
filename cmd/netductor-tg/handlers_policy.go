package main

import (
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func handleUserPolicyCB(token string, chat int64, msgID int, data string) {
	if data == "u:policy" || strings.HasPrefix(data, "u:policy:") {
		name := strings.TrimPrefix(data, "u:policy:")
		name = strings.TrimPrefix(name, "u:policy")
		name = strings.Trim(name, ":")
		if name == "" {
			reply(token, chat, msgID, "❌ name required", usersListKeyboard())
			return
		}
		reply(token, chat, msgID, formatUserPolicyHTML(name), policyNavKeyboard("user", name))
		return
	}
	if !strings.HasPrefix(data, "u:pol:") {
		return
	}
	rest := strings.TrimPrefix(data, "u:pol:")
	name, action, ok := splitOnce(rest, ":")
	if !ok || name == "" || action == "" {
		reply(token, chat, msgID, "❌ bad policy callback", usersListKeyboard())
		return
	}
	p, err := vpn.GetUserPolicy(name)
	if err != nil {
		reply(token, chat, msgID, "❌ "+esc(err.Error()), usersListKeyboard())
		return
	}
	p = applyPolicyToggle(p, action)
	if err := vpn.SetUserPolicy(name, p, "tg"); err != nil {
		reply(token, chat, msgID, "❌ "+esc(err.Error()), policyNavKeyboard("user", name))
		return
	}
	reply(token, chat, msgID, formatUserPolicyHTML(name), policyNavKeyboard("user", name))
}

func handleEdgePolicyCB(token string, chat int64, msgID int, data string) {
	if data == "e:policy" || strings.HasPrefix(data, "e:policy:") {
		id := strings.TrimPrefix(data, "e:policy:")
		id = strings.TrimPrefix(id, "e:policy")
		id = strings.Trim(id, ":")
		if id == "" {
			reply(token, chat, msgID, "❌ device id required", backTo("routers"))
			return
		}
		reply(token, chat, msgID, formatEdgePolicyHTML(id), policyNavKeyboard("edge", id))
		return
	}
	if !strings.HasPrefix(data, "e:pol:") {
		return
	}
	rest := strings.TrimPrefix(data, "e:pol:")
	id, action, ok := splitOnce(rest, ":")
	if !ok || id == "" || action == "" {
		reply(token, chat, msgID, "❌ bad edge policy callback", backTo("routers"))
		return
	}
	p, err := edge.GetDevicePolicy(id)
	if err != nil {
		reply(token, chat, msgID, "❌ "+esc(err.Error()), backTo("routers"))
		return
	}
	p = applyPolicyToggle(p, action)
	if err := edge.SetDevicePolicy(id, p, "tg"); err != nil {
		reply(token, chat, msgID, "❌ "+esc(err.Error()), policyNavKeyboard("edge", id))
		return
	}
	reply(token, chat, msgID, formatEdgePolicyHTML(id), policyNavKeyboard("edge", id))
}

func splitOnce(s, sep string) (a, b string, ok bool) {
	i := strings.Index(s, sep)
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+len(sep):], true
}
