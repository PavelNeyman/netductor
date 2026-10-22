package vpscheck

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Geoblock-style probes: can this VPS reach services that often geo-restrict?
// Not a clone of censorcheck — structured reachability for common targets.

func checkGeoblock(ctx context.Context, profile Profile) []Result {
	var targets []struct {
		id, en, ru, url string
	}
	// Abroad-facing (from RU secondary: expected often blocked or throttled)
	abroad := []struct{ id, en, ru, url string }{
		{"geo_netflix", "Netflix CDN", "Netflix CDN", "https://www.netflix.com/title/80057281"},
		{"geo_youtube", "YouTube", "YouTube", "https://www.youtube.com/"},
		{"geo_twitter", "X/Twitter", "X/Twitter", "https://x.com/"},
		{"geo_discord", "Discord", "Discord", "https://discord.com/"},
		{"geo_chatgpt", "OpenAI", "OpenAI", "https://api.openai.com/"},
		{"geo_instagram", "Instagram", "Instagram", "https://www.instagram.com/"},
	}
	// RU services (from abroad primary: may be limited)
	ru := []struct{ id, en, ru, url string }{
		{"geo_yandex", "Yandex", "Яндекс", "https://ya.ru/"},
		{"geo_vk", "VK", "VK", "https://vk.com/"},
		{"geo_gosuslugi", "Gosuslugi", "Госуслуги", "https://www.gosuslugi.ru/"},
		{"geo_rutube", "Rutube", "Rutube", "https://rutube.ru/"},
	}
	switch profile {
	case ProfileSecondary:
		targets = abroad // what RU VPS can reach outward
		targets = append(targets, ru...)
	case ProfilePrimary:
		targets = abroad
		targets = append(targets, ru...)
	default:
		targets = append(abroad, ru...)
	}
	var out []Result
	for _, t := range targets {
		out = append(out, probeHTTP(ctx, t.id, t.en, t.ru, t.url))
	}
	return out
}

func probeHTTP(ctx context.Context, id, en, ru, url string) Result {
	start := time.Now()
	r := Result{ID: id, TitleEN: en, TitleRU: ru}
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		r.Status = StatusWarn
		r.Detail = err.Error()
		return r
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; netductor-vpscheck/1.0)")
	resp, err := client.Do(req)
	r.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		r.Status = StatusWarn
		r.Detail = err.Error()
		return r
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32<<10))
	code := resp.StatusCode
	// 200-399 ok; 403/451 often geo; 5xx warn
	switch {
	case code >= 200 && code < 400:
		r.Status = StatusOK
		r.Detail = fmt.Sprintf("%d %dms", code, r.DurationMs)
	case code == 403 || code == 451:
		r.Status = StatusWarn
		r.Detail = fmt.Sprintf("%d geo/forbidden %dms", code, r.DurationMs)
	case code >= 500:
		r.Status = StatusWarn
		r.Detail = fmt.Sprintf("%d server %dms", code, r.DurationMs)
	default:
		r.Status = StatusWarn
		r.Detail = fmt.Sprintf("%d %dms", code, r.DurationMs)
	}
	_ = strings.TrimSpace
	return r
}
