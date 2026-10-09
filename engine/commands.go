package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func cmdHwid() error {
	s, err := loadState()
	if err != nil {
		return err
	}
	h, err := ensureHwid(s)
	if err != nil {
		return err
	}
	fmt.Println(h)
	return nil
}

func cmdSubAdd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sub-add <url>")
	}
	if strings.HasPrefix(strings.TrimSpace(args[0]), "vless://") {
		return cmdAdd(args)
	}
	s, err := loadState()
	if err != nil {
		return err
	}
	s.SubURL = args[0]
	if _, err := ensureHwid(s); err != nil {
		return err
	}
	if err := saveState(s); err != nil {
		return err
	}
	return refreshSubscription(s)
}

func cmdSubUpdate() error {
	s, err := loadState()
	if err != nil {
		return err
	}
	if s.SubURL == "" {
		return fmt.Errorf("no subscription set; run: sub-add <url>")
	}
	return refreshSubscription(s)
}

func refreshSubscription(s *State) error {
	if _, err := ensureHwid(s); err != nil {
		return err
	}
	res, err := FetchSubscription(s.SubURL, s.Hwid)
	if err != nil {
		return err
	}
	s.Announce = res.Announce
	s.ProfileURL = res.ProfileURL
	s.UserInfo = res.UserInfo
	s.HwidActive = res.HwidActive
	s.Warning = res.Warning
	s.LastUpdate = time.Now().Format(time.RFC3339)

	if res.HwidNotSupp {
		s.HwidNotSupp = true
		_ = saveState(s)
		return fmt.Errorf("panel requires HWID (x-hwid-not-supported); hwid=%s", s.Hwid)
	}
	if res.HwidLimit {
		s.HwidLimit = true
		_ = saveState(s)
		return fmt.Errorf("%s", firstNonEmpty(res.Warning, "Достигнут лимит устройств (x-hwid-max-devices-reached)"))
	}
	s.HwidNotSupp, s.HwidLimit = false, false

	var servers []Server
	for _, link := range res.Links {
		v, err := ParseVless(link)
		if err != nil {
			continue
		}
		name := v.Remark
		if name == "" {
			name = v.Address + ":" + strconv.Itoa(v.Port)
		}
		servers = append(servers, Server{
			Name: name, URI: link, Address: v.Address, Port: v.Port, Network: v.Network,
		})
	}
	s.Servers = mergeServers(servers, manualServers(s.Servers))
	if s.Selected >= len(s.Servers) {
		s.Selected = 0
	}
	if err := saveState(s); err != nil {
		return err
	}
	if len(servers) == 0 && s.Warning != "" {
		return fmt.Errorf("%s", s.Warning)
	}
	fmt.Printf("updated: %d servers, hwid=%s\n", len(servers), s.Hwid)
	if s.UserInfo != "" {
		fmt.Println("userinfo:", s.UserInfo)
	}
	if s.Announce != "" {
		fmt.Println("announce:", s.Announce)
	}
	if s.Warning != "" {
		fmt.Println("warning:", s.Warning)
	}
	return nil
}

// cmdAdd appends one or more vless:// share links as manual servers. Manual
// servers are user-pinned and survive subscription refreshes.
func cmdAdd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: add <vless://...>")
	}
	s, err := loadState()
	if err != nil {
		return err
	}
	added := 0
	for _, raw := range args {
		raw = strings.TrimSpace(raw)
		v, err := ParseVless(raw)
		if err != nil {
			return err
		}
		if hasServerURI(s.Servers, raw) {
			continue
		}
		name := v.Remark
		if name == "" {
			name = v.Address + ":" + strconv.Itoa(v.Port)
		}
		s.Servers = append(s.Servers, Server{
			Name: name, URI: raw, Address: v.Address, Port: v.Port,
			Network: v.Network, Manual: true,
		})
		added++
	}
	if err := saveState(s); err != nil {
		return err
	}
	fmt.Printf("added: %d (total %d)\n", added, len(s.Servers))
	return nil
}

// hasServerURI reports whether a server with the exact URI is already stored.
func hasServerURI(servers []Server, uri string) bool {
	for _, srv := range servers {
		if srv.URI == uri {
			return true
		}
	}
	return false
}

// manualServers returns the user-pinned (manually added) servers from servers.
func manualServers(servers []Server) []Server {
	var out []Server
	for _, srv := range servers {
		if srv.Manual {
			out = append(out, srv)
		}
	}
	return out
}

// mergeServers concatenates fetched and manual servers, dropping manual
// entries whose URI is already present among the fetched ones.
func mergeServers(fetched, manual []Server) []Server {
	out := append([]Server{}, fetched...)
	for _, m := range manual {
		if !hasServerURI(out, m.URI) {
			out = append(out, m)
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func cmdList() error {
	s, err := loadState()
	if err != nil {
		return err
	}
	for i, srv := range s.Servers {
		mark := " "
		if i == s.Selected {
			mark = "*"
		}
		fmt.Printf("%s%d\t%s\n", mark, i, srv.Name)
	}
	return nil
}

func cmdSet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: set <index>")
	}
	idx, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	s, err := loadState()
	if err != nil {
		return err
	}
	if idx < 0 || idx >= len(s.Servers) {
		return fmt.Errorf("index out of range (0..%d)", len(s.Servers)-1)
	}
	s.Selected = idx
	return saveState(s)
}

func cmdUp() error {
	s, err := loadState()
	if err != nil {
		return err
	}
	if len(s.Servers) == 0 {
		return fmt.Errorf("no servers; run: sub-update")
	}
	v, err := ParseVless(s.Servers[s.Selected].URI)
	if err != nil {
		return err
	}
	stopXray()
	if err := startXray(v, s.AllowLAN); err != nil {
		return err
	}
	if err := proxyOn(); err != nil {
		fmt.Fprintln(os.Stderr, "warning: system proxy set failed:", err)
	}
	fmt.Printf("up: %s (socks 127.0.0.1:%d, http 127.0.0.1:%d)\n",
		s.Servers[s.Selected].Name, socksPort, httpPort)
	if s.AllowLAN {
		fmt.Printf("LAN: другие устройства могут использовать HTTP-прокси %s\n", lanGatewayURL())
	}
	return nil
}

// cmdConn reports public IP (direct), exit IP via proxy if tunnel is up,
// and ping to selected server.
func cmdConn() error {
	lines := []string{}

	// 1) Public IP directly (no proxy, no tunnel)
	if ip, err := getPublicIP(); err == nil {
		lines = append(lines, "Ваш IP (напрямую): "+ip)
	} else {
		lines = append(lines, "Ваш IP (напрямую): ошибка — "+err.Error())
	}

	// 2) Exit IP via proxy (only if tunnel is running)
	if xrayRunning() {
		if ip, lat, err := getExitIPViaProxy(); err == nil {
			lines = append(lines, fmt.Sprintf("Exit IP (через туннель): %s (латентность %s)", ip, lat))
		} else {
			lines = append(lines, "Exit IP (через туннель): ошибка — "+err.Error())
		}
	}

	// 3) Ping selected server (no tunnel needed)
	if out, err := pingSelectedServer(); err == nil {
		lines = append(lines, out)
	} else {
		lines = append(lines, "Пинг сервера: "+err.Error())
	}

	fmt.Println(strings.Join(lines, "\n"))
	return nil
}

// cmdPing pings the selected server directly (no tunnel required).
func cmdPing() error {
	out, err := pingSelectedServer()
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

// cmdPingThroughTunnel measures latency through the VPN tunnel (HTTP via
// local proxy). Requires the tunnel to be up.
func cmdPingThroughTunnel() error {
	if !xrayRunning() {
		return fmt.Errorf("туннель не запущен (нажмите Подключить)")
	}
	s, _ := loadState()
	srvName := "(нет сервера)"
	if s.Selected >= 0 && s.Selected < len(s.Servers) {
		srvName = s.Servers[s.Selected].Name
	}

	proxyURL, err := url.Parse("http://127.0.0.1:" + strconv.Itoa(httpPort))
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			Proxy: func(*http.Request) (*url.URL, error) { return proxyURL, nil },
		},
	}

	// Warm-up + measure a few times for a stable number.
	var total time.Duration
	var n int
	var lastIP string
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest(http.MethodGet, "https://ipinfo.io/ip", nil)
		req.Header.Set("User-Agent", "VLessBar/"+version)
		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			if i == 0 {
				return fmt.Errorf("запрос через туннель не удался: %v", err)
			}
			break
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		total += time.Since(start)
		n++
		lastIP = strings.TrimSpace(string(body))
	}
	if n == 0 {
		return fmt.Errorf("нет успешных запросов")
	}
	avg := (total / time.Duration(n)).Round(time.Millisecond)
	fmt.Printf("Пинг через туннель: %s (сервер: %s, запросов: %d)\n", avg, srvName, n)
	if lastIP != "" {
		fmt.Println("Exit IP:", lastIP)
	}
	return nil
}

// cmdIP prints the current public IP directly (no proxy).
func cmdIP() error {
	ip, err := getPublicIP()
	if err != nil {
		return err
	}
	fmt.Println(ip)
	return nil
}

// cmdLan toggles LAN gateway mode (listen on 0.0.0.0 instead of loopback).
func cmdLan(args []string) error {
	s, err := loadState()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		fmt.Printf("allow_lan=%v\n", s.AllowLAN)
		if s.AllowLAN {
			fmt.Println("LAN:", lanGatewayURL())
		}
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "on", "1", "true", "yes":
		s.AllowLAN = true
	case "off", "0", "false", "no":
		s.AllowLAN = false
	default:
		return fmt.Errorf("usage: lan on|off")
	}
	if err := saveState(s); err != nil {
		return err
	}
	fmt.Printf("allow_lan=%v\n", s.AllowLAN)
	if s.AllowLAN {
		fmt.Println("LAN:", lanGatewayURL())
	}
	return nil
}

// cmdCheckUpdate prints whether a newer app build is available.
func cmdCheckUpdate() error {
	status, url, ver := checkUpdate()
	fmt.Println(status)
	if url != "" {
		fmt.Printf("version=%s\nurl=%s\n", ver, url)
	}
	return nil
}

// cmdUpdate downloads the newer build and schedules a restart (app only).
func cmdUpdate() error {
	_, url, ver := checkUpdate()
	if url == "" {
		return fmt.Errorf("нет доступного обновления")
	}
	if runtime.GOOS != "windows" {
		// The macOS asset is a zip of the whole .app bundle; swapping a single
		// binary inside Contents/MacOS would corrupt it. Open the release page
		// so the user replaces the app the normal way.
		if err := openURL(releasePageURL); err != nil {
			return err
		}
		fmt.Printf("открыта страница релиза %s — скачайте и замените VLessBar.app\n", ver)
		return nil
	}
	if err := applyUpdate(url); err != nil {
		return err
	}
	fmt.Printf("обновление до %s запущено, приложение закроется\n", ver)
	return nil
}

func cmdDown() error {
	stopXray()
	if err := proxyOff(); err != nil {
		fmt.Fprintln(os.Stderr, "warning: proxy clear failed:", err)
	}
	fmt.Println("down")
	return nil
}

func cmdStatus() error {
	s, err := loadState()
	if err != nil {
		return err
	}
	st := map[string]interface{}{
		"running":     xrayRunning(),
		"selected":    s.Selected,
		"servers":     len(s.Servers),
		"hwid":        s.Hwid,
		"sub_url":     s.SubURL,
		"user_info":   s.UserInfo,
		"last_update": s.LastUpdate,
		"allow_lan":   s.AllowLAN,
	}
	if s.AllowLAN {
		st["lan_url"] = lanGatewayURL()
	}
	if s.AllowLAN {
		st["lan_url"] = lanGatewayURL()
	}
	if s.Selected >= 0 && s.Selected < len(s.Servers) {
		st["selected_name"] = s.Servers[s.Selected].Name
	}
	out, _ := json.MarshalIndent(st, "", "  ")
	fmt.Println(string(out))
	return nil
}
