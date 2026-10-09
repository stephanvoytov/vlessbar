package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Vless holds the fields of a vless:// share link that matter to Xray client config.
type Vless struct {
	UUID          string
	Address       string
	Port          int
	Security      string // none | tls | reality
	Network       string // tcp | ws | grpc | h2 | http
	Flow          string
	SNI           string
	Fingerprint   string
	PublicKey     string
	ShortID       string
	SpiderX       string
	Path          string
	Host          string
	ServiceName   string
	ALPN          string
	HeaderType    string
	AllowInsecure bool
	Remark        string
}

// ParseVless parses a vless:// URI into a Vless struct.
func ParseVless(raw string) (*Vless, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "vless" {
		return nil, fmt.Errorf("not a vless link (%q)", u.Scheme)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return nil, fmt.Errorf("bad port %q: %w", u.Port(), err)
	}
	q := u.Query()
	v := &Vless{
		UUID:          u.User.Username(),
		Address:       u.Hostname(),
		Port:          port,
		Security:      defaultStr(q.Get("security"), "none"),
		Network:       defaultStr(q.Get("type"), "tcp"),
		Flow:          q.Get("flow"),
		SNI:           q.Get("sni"),
		Fingerprint:   q.Get("fp"),
		PublicKey:     q.Get("pbk"),
		ShortID:       q.Get("sid"),
		SpiderX:       q.Get("spx"),
		Path:          q.Get("path"),
		Host:          q.Get("host"),
		ServiceName:   q.Get("serviceName"),
		ALPN:          q.Get("alpn"),
		HeaderType:    q.Get("headerType"),
		AllowInsecure: q.Get("allowInsecure") == "1",
		Remark:        u.Fragment,
	}
	if v.SNI == "" {
		v.SNI = v.Host
	}
	if v.UUID == "" {
		return nil, fmt.Errorf("missing uuid")
	}
	return v, nil
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
