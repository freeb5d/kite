package profile

import (
	"net/http"
	"net/url"
	"strings"
)

// SupportURL returns the provider's support link from a subscription
// response: the Support-URL header (Profile-Web-Page-URL as a fallback).
// Telegram handles ("@name") become t.me links; only http(s) and tg: links
// are accepted, so a subscription can't make the UI open anything else.
func SupportURL(h http.Header) string {
	for _, key := range []string{"Support-Url", "Profile-Web-Page-Url"} {
		v := strings.TrimSpace(h.Get(key))
		if v == "" {
			continue
		}
		if strings.HasPrefix(v, "@") {
			v = "https://t.me/" + strings.TrimPrefix(v, "@")
		}
		u, err := url.Parse(v)
		if err != nil || u.Host == "" && u.Scheme != "tg" {
			continue
		}
		switch strings.ToLower(u.Scheme) {
		case "http", "https", "tg":
			return v
		}
	}
	return ""
}
