package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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
	s.Servers = servers
	if s.Selected >= len(servers) {
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
	if err := startXray(v); err != nil {
		return err
	}
	if err := proxyOn(); err != nil {
		fmt.Fprintln(os.Stderr, "warning: system proxy set failed:", err)
	}
	fmt.Printf("up: %s (socks 127.0.0.1:%d, http 127.0.0.1:%d)\n",
		s.Servers[s.Selected].Name, socksPort, httpPort)
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
	}
	if s.Selected >= 0 && s.Selected < len(s.Servers) {
		st["selected_name"] = s.Servers[s.Selected].Name
	}
	out, _ := json.MarshalIndent(st, "", "  ")
	fmt.Println(string(out))
	return nil
}
