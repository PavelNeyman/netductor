package main

import "strings"

func handleCBHubs(token string, chat int64, msgID int, data string) bool {
	if data == "m:git" || strings.HasPrefix(data, "m:git:") {
		handleGitCB(token, chat, msgID, data)
		return true
	}
	if data == "m:nvr" || strings.HasPrefix(data, "m:nvr:") {
		handleNVRCB(token, chat, msgID, data)
		return true
	}
	if data == "m:release" || strings.HasPrefix(data, "m:release:") {
		// Legacy menu: always Versions hub (multi-component), not old primary-only release UI
		handleVersionsCB(token, chat, msgID, "m:versions")
		return true
	}
	if data == "m:updates" || data == "m:versions" || strings.HasPrefix(data, "m:ver:") {
		handleVersionsCB(token, chat, msgID, data)
		return true
	}
	if strings.HasPrefix(data, "m:updates:") {
		handleUpdatesCB(token, chat, msgID, data)
		return true
	}
	if handleEdgeStorageCB(token, chat, msgID, data) {
		return true
	}
	if handleEdgeGuestCB(token, chat, msgID, data) {
		return true
	}
	if handleEdgeLuciCB(token, chat, msgID, data) {
		return true
	}
	if handleEdgeTemplateCB(token, chat, msgID, data) {
		return true
	}
	if handleCertsCB(token, chat, msgID, data) {
		return true
	}
	if handleRegistryCallback(token, chat, msgID, data) {
		return true
	}
	if handleGitCB(token, chat, msgID, data) {
		return true
	}
	if data == "m:guest" || strings.HasPrefix(data, "m:guest:") {
		handleGuestCB(token, chat, msgID, data)
		return true
	}
	if data == "m:dns" || strings.HasPrefix(data, "m:dns:") {
		handleDNSCB(token, chat, msgID, data)
		return true
	}
	if data == "m:logs" || strings.HasPrefix(data, "m:logs:") {
		handleLogsCB(token, chat, msgID, data)
		return true
	}
	if data == "m:backup" || strings.HasPrefix(data, "m:backup:") {
		handleBackupCB(token, chat, msgID, data)
		return true
	}
	if data == "m:loc" || strings.HasPrefix(data, "m:loc:") {
		handleLocationCB(token, chat, msgID, data)
		return true
	}
	if strings.HasPrefix(data, "m:quota:") {
		handleQuotaCB(token, chat, msgID, data)
		return true
	}
	if strings.HasPrefix(data, "u:policy") || strings.HasPrefix(data, "u:pol:") {
		handleUserPolicyCB(token, chat, msgID, data)
		return true
	}
	if strings.HasPrefix(data, "e:policy") || strings.HasPrefix(data, "e:pol:") {
		handleEdgePolicyCB(token, chat, msgID, data)
		return true
	}
	return false
}
