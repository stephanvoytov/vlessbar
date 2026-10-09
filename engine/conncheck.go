package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// pingServer measures RTT to the server via TCP connect (no tunnel needed).
// Returns latency and error if unreachable.
func pingServer(addr string, port int, timeout time.Duration) (time.Duration, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(addr, strconv.Itoa(port)), timeout)
	if err != nil {
		return 0, err
	}
	conn.Close()
	return time.Since(start), nil
}

// pingSelectedServer pings the currently selected server directly.
func pingSelectedServer() (string, error) {
	s, err := loadState()
	if err != nil {
		return "", err
	}
	if s.Selected < 0 || s.Selected >= len(s.Servers) {
		return "", fmt.Errorf("сервер не выбран")
	}
	srv := s.Servers[s.Selected]
	rtt, err := pingServer(srv.Address, srv.Port, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("%s (%s:%d) недоступен: %v", srv.Name, srv.Address, srv.Port, err)
	}
	return fmt.Sprintf("Сервер: %s (%s:%d)\nRTT: %s", srv.Name, srv.Address, srv.Port, rtt.Round(time.Millisecond)), nil
}

// getPublicIP returns the current public IP directly (no proxy, no tunnel).
func getPublicIP() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "https://ipinfo.io/ip", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "VLessBar/"+version)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return "", fmt.Errorf("пустой ответ")
	}
	return ip, nil
}

// getExitIPViaProxy returns the exit IP through the local HTTP proxy (tunnel).
func getExitIPViaProxy() (string, time.Duration, error) {
	if !xrayRunning() {
		return "", 0, fmt.Errorf("туннель не запущен")
	}
	proxyURL, err := url.Parse("http://127.0.0.1:" + strconv.Itoa(httpPort))
	if err != nil {
		return "", 0, err
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			Proxy: func(*http.Request) (*url.URL, error) { return proxyURL, nil },
		},
	}
	start := time.Now()
	req, err := http.NewRequest(http.MethodGet, "https://ipinfo.io/ip", nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", "VLessBar/"+version)
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	lat := time.Since(start).Round(time.Millisecond)
	body, _ := io.ReadAll(resp.Body)
	ip := strings.TrimSpace(string(body))
	return ip, lat, nil
}
