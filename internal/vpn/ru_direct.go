package vpn

// RuDirectSuffixes — domains that must NOT exit via foreign primary.
// Used by client profiles and secondary relay routing.
// Matching is domain_suffix (gosuslugi.ru matches *.gosuslugi.ru).
func RuDirectSuffixes() []string {
	return []string{
		// TLDs
		"ru", "su", "xn--p1ai", "xn--p1acf",
		// VK / Mail
		"vk.com", "vk.ru", "vk.me", "userapi.com", "vkuservideo.net", "vk-cdn.net",
		"mail.ru", "imgsmail.ru", "ok.ru", "odnoklassniki.ru", "mycdn.me",
		// Yandex
		"yandex.ru", "yandex.net", "yandex.com", "ya.ru", "yastatic.net", "yandex.cloud",
		"yandex.by", "yandex.kz", "auto.ru",
		// State / ESIA / regions
		"gosuslugi.ru", "gu-st.ru", "pos.gosuslugi.ru", "gosuslugi-online.ru",
		"mos.ru", "mosreg.ru", "nalog.ru", "cbr.ru", "gov.ru", "government.ru",
		"emias.ru", "rzd.ru", "pochta.ru", "sfr.gov.ru", "pfr.gov.ru",
		"edu.ru", "edu.gov.ru", "myschool.edu.ru", "gup.education",
		// Banks
		"sberbank.ru", "sber.ru", "sberbank.com", "tinkoff.ru", "tbank.ru",
		"vtb.ru", "alfabank.ru", "gazprombank.ru", "open.ru", "raiffeisen.ru",
		"nspk.ru", "mironline.ru", "sbp.nspk.ru",
		// Marketplaces / retail
		"wildberries.ru", "wb.ru", "wbbasket.ru", "ozon.ru", "ozonusercontent.com",
		"avito.ru", "dns-shop.ru", "citilink.ru", "mvideo.ru", "lamoda.ru",
		// Telco
		"mts.ru", "megafon.ru", "beeline.ru", "tele2.ru", "yota.ru", "t2.ru",
		// Maps / media
		"2gis.com", "2gis.ru", "rutube.ru", "ivi.ru", "kinopoisk.ru", "hd.kinopoisk.ru",
		"okko.tv", "wink.ru",
	}
}

// RuDirectKeywords — domain_keyword match (covers odd subdomains / non-.ru hosts).
func RuDirectKeywords() []string {
	return []string{
		"gosuslugi",
		"esia",
		"emias",
		"sberbank",
		"sber",
		"tinkoff",
		"tbank",
		"wildberries",
		"ozon",
	}
}

// ShadowrocketRuDirectRules returns [Rule] lines for Shadowrocket (before FINAL).
func ShadowrocketRuDirectRules() []string {
	var lines []string
	lines = append(lines, "# netductor RU/gov direct — keep foreign exit off these")
	for _, k := range RuDirectKeywords() {
		lines = append(lines, "DOMAIN-KEYWORD,"+k+",DIRECT")
	}
	for _, s := range RuDirectSuffixes() {
		// Shadowrocket: DOMAIN-SUFFIX,ru matches *.ru
		lines = append(lines, "DOMAIN-SUFFIX,"+s+",DIRECT")
	}
	lines = append(lines, "GEOIP,RU,DIRECT")
	return lines
}
