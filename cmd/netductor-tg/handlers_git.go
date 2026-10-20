package main

import (
	"strings"
)

func handleGitCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:git" && !strings.HasPrefix(data, "m:git:") {
		return false
	}
	if handleGitHub(token, chat, msgID, data) {
		return true
	}
	rest := strings.TrimPrefix(data, "m:git:")
	ru := getLang() != "en"
	if handleGitBuildHost(token, chat, msgID, rest) {
		return true
	}
	if handleGitShip(token, chat, msgID, rest) {
		return true
	}
	if handleGitProj(token, chat, msgID, rest, ru) {
		return true
	}
	if handleGitRepo(token, chat, msgID, rest, ru) {
		return true
	}
	return true
}
