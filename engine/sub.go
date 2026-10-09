package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// SubResult is the parsed subscription response.
type SubResult struct {
	Links       []string
	Announce    string
	ProfileURL  string
	UserInfo    string
	HwidActive  bool
	HwidLimit   bool
	HwidNotSupp bool
	Warning     string
}

// FetchSubscription performs a Happ-compatible subscription request:
// it sends the HWID/device headers and parses both the body and the
// HWID-related response headers.
func FetchSubscription(subURL, hwid string) (*SubResult, error) {
	req, err := http.NewRequest(http.MethodGet, subURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Happ/3.13.0")
	req.Header.Set("x-hwid", hwid)
	req.Header.Set("x-device-os", "macOS")
	req.Header.Set("x-ver-os", deviceOSVersion())
	req.Header.Set("x-device-model", deviceModel())
	req.Header.Set("x-device-locale", "ru")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	res := &SubResult{
		Announce:    decodeMaybeBase64(resp.Header.Get("announce")),
		ProfileURL:  resp.Header.Get("profile-web-page-url"),
		UserInfo:    resp.Header.Get("subscription-userinfo"),
		HwidActive:  headerTrue(resp.Header.Get("x-hwid-active")),
		HwidLimit:   headerTrue(resp.Header.Get("x-hwid-max-devices-reached")) || headerTrue(resp.Header.Get("x-hwid-limit")),
		HwidNotSupp: headerTrue(resp.Header.Get("x-hwid-not-supported")),
	}
	if res.HwidNotSupp {
		res.Warning = "Панель не поддерживает HWID (x-hwid-not-supported)."
		return res, nil
	}
	if res.HwidLimit {
		res.Warning = res.Announce
		if res.Warning == "" {
			res.Warning = "Достигнут лимит устройств. Отключите лишнее устройство в личном кабинете, затем обновите подписку."
		}
		return res, nil
	}

	text := decodeMaybeBase64(string(body))
	all := extractLinks(text)
	res.Links = filterLinks(all)
	if len(res.Links) == 0 && len(all) > 0 {
		res.Warning = "Панель не вернула рабочих серверов (возможно, достигнут лимит устройств)."
	}
	if resp.StatusCode != http.StatusOK && len(res.Links) == 0 {
		return res, fmt.Errorf("subscription HTTP %d", resp.StatusCode)
	}
	return res, nil
}

func deviceOSVersion() string {
	if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
		if v := strings.TrimSpace(string(out)); v != "" {
			return v
		}
	}
	return "10.13"
}

func deviceModel() string {
	if out, err := exec.Command("sysctl", "-n", "hw.model").Output(); err == nil {
		if m := strings.TrimSpace(string(out)); m != "" {
			return m
		}
	}
	return "Mac"
}

func headerTrue(v string) bool {
	return v == "true" || v == "1"
}

// decodeMaybeBase64 returns the decoded string when s is valid base64 that
// yields link-like content, otherwise returns s unchanged.
func decodeMaybeBase64(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	encodings := []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	}
	for _, enc := range encodings {
		if b, err := enc.DecodeString(s); err == nil {
			d := string(b)
			if strings.Contains(d, "://") || strings.Contains(d, "\n") {
				return d
			}
		}
	}
	return s
}

// extractLinks pulls share links out of a subscription body, skipping blank
// lines and metadata/comment lines (#key: value).
func extractLinks(text string) []string {
	var links []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		if i := strings.Index(line, "://"); i > 0 {
			links = append(links, line)
		}
	}
	return links
}

// filterLinks drops non-usable entries such as the "device limit" placeholder
// some panels return (a zero UUID pointing at 0.0.0.0:1).
func filterLinks(links []string) []string {
	var out []string
	for _, l := range links {
		v, err := ParseVless(l)
		if err != nil {
			continue
		}
		if v.Address == "" || v.Address == "0.0.0.0" || v.Port <= 1 || isZeroUUID(v.UUID) {
			continue
		}
		out = append(out, l)
	}
	return out
}

func isZeroUUID(u string) bool {
	return strings.ReplaceAll(u, "-", "") == strings.Repeat("0", 32)
}
