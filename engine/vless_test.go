package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseVlessReality(t *testing.T) {
	raw := "vless://11111111-2222-3333-4444-555555555555@example.com:443" +
		"?type=tcp&security=reality&pbk=PUBKEY&fp=chrome&sni=www.microsoft.com&sid=abcd&spx=%2F&flow=xtls-rprx-vision#My%20Server"
	v, err := ParseVless(raw)
	if err != nil {
		t.Fatalf("ParseVless: %v", err)
	}
	checks := map[string]string{
		"uuid": v.UUID, "addr": v.Address, "security": v.Security,
		"network": v.Network, "sni": v.SNI, "pbk": v.PublicKey,
		"sid": v.ShortID, "spx": v.SpiderX, "flow": v.Flow, "remark": v.Remark,
	}
	want := map[string]string{
		"uuid": "11111111-2222-3333-4444-555555555555", "addr": "example.com",
		"security": "reality", "network": "tcp", "sni": "www.microsoft.com",
		"pbk": "PUBKEY", "sid": "abcd", "spx": "/", "flow": "xtls-rprx-vision",
		"remark": "My Server",
	}
	for k, got := range checks {
		if got != want[k] {
			t.Errorf("%s = %q, want %q", k, got, want[k])
		}
	}
	if v.Port != 443 {
		t.Errorf("port = %d, want 443", v.Port)
	}
}

func TestParseVlessWS(t *testing.T) {
	raw := "vless://uuid@h.example.com:8443?type=ws&security=tls&sni=sni.example.com&path=%2Fws&host=cdn.example.com#WS"
	v, err := ParseVless(raw)
	if err != nil {
		t.Fatalf("ParseVless: %v", err)
	}
	if v.Network != "ws" || v.Path != "/ws" || v.Host != "cdn.example.com" {
		t.Errorf("ws parse wrong: %+v", v)
	}
	if v.Security != "tls" || v.SNI != "sni.example.com" {
		t.Errorf("tls parse wrong: %+v", v)
	}
}

func TestParseVlessRejectsNonVless(t *testing.T) {
	if _, err := ParseVless("vmess://abc@host:1"); err == nil {
		t.Error("expected error for non-vless scheme")
	}
	if _, err := ParseVless("vless://noport"); err == nil {
		t.Error("expected error for missing port")
	}
}

func TestBuildXrayConfigReality(t *testing.T) {
	v, _ := ParseVless("vless://u@host:443?type=tcp&security=reality&pbk=K&sid=S&sni=SNI&flow=xtls-rprx-vision")
	cfg := buildXrayConfig(v)
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	for _, want := range []string{`"realitySettings"`, `"publicKey":"K"`, `"shortId":"S"`, `"flow":"xtls-rprx-vision"`, `"encryption":"none"`, `"protocol":"socks"`, `"protocol":"http"`} {
		if !strings.Contains(s, want) {
			t.Errorf("config missing %s\n%s", want, s)
		}
	}
}

func TestDecodeMaybeBase64(t *testing.T) {
	// base64 of "vless://a@b:1\nvless://c@d:2"
	enc := "dmxlc3M6Ly9hQGI6MQp2bGVzczovL2NAZDoy"
	got := decodeMaybeBase64(enc)
	if !strings.Contains(got, "vless://a@b:1") || !strings.Contains(got, "vless://c@d:2") {
		t.Errorf("decode failed: %q", got)
	}
	plain := "vless://plain@host:443#x"
	if decodeMaybeBase64(plain) != plain {
		t.Errorf("plain text should pass through unchanged")
	}
}

func TestExtractLinks(t *testing.T) {
	body := "vless://a@b:1#one\n#subscription-userinfo: upload=1\n\n  vless://c@d:2#two  \n"
	links := extractLinks(body)
	if len(links) != 2 {
		t.Fatalf("got %d links, want 2: %v", len(links), links)
	}
	if links[0] != "vless://a@b:1#one" || links[1] != "vless://c@d:2#two" {
		t.Errorf("links = %v", links)
	}
}

func TestHwidFormat(t *testing.T) {
	for i := 0; i < 100; i++ {
		h := newHwid()
		if !hwidRe.MatchString(h) {
			t.Fatalf("hwid %q does not match %s", h, hwidRe)
		}
	}
}
