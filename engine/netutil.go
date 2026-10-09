package main

import (
	"net"
	"sort"
	"strconv"
	"strings"
)

// nonLANPrefixes are virtual interface name prefixes that never represent the
// real local network (Docker, WSL, Hyper-V, tunnels...).
var nonLANPrefixes = []string{
	"docker", "br-", "virbr", "veth", "vethernet", "wsl",
	"tun", "tap", "bridge", "hyper-v", "loopback", "isatap", "teredo", "cellular",
}

func isVirtualIFace(name string) bool {
	l := strings.ToLower(name)
	for _, p := range nonLANPrefixes {
		if strings.HasPrefix(l, p) {
			return true
		}
	}
	return false
}

// rank scores an IPv4 address for use as the LAN gateway address: private
// ranges typical of home/office LANs score highest, -1 means "never use".
func rank(ip net.IP) int {
	s := ip.String()
	switch {
	case ip.IsLoopback(), ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast():
		return -1
	case strings.HasPrefix(s, "192.168."):
		return 100
	case strings.HasPrefix(s, "10."):
		return 90
	case strings.HasPrefix(s, "172.16."), strings.HasPrefix(s, "172.17."):
		// 172.16/17 — Docker/WSL territory, use only as a fallback.
		return 20
	}
	return 40
}

// localIPv4 returns the best non-loopback IPv4 address of this machine, or ""
// when none is found. Virtual interfaces (docker/wsl/...) are skipped when a
// real LAN address exists.
func localIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	type cand struct {
		ip    string
		score int
	}
	var cands []cand

	for _, iface := range ifaces {
		if isVirtualIFace(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil {
				continue
			}
			if s := rank(ip4); s >= 0 {
				cands = append(cands, cand{ip: ip4.String(), score: s})
			}
		}
	}
	if len(cands) == 0 {
		return ""
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	return cands[0].ip
}

// lanGatewayURL is the HTTP proxy URL that other devices on the LAN should use
// when "allow LAN" is enabled.
func lanGatewayURL() string {
	ip := localIPv4()
	if ip == "" {
		ip = "127.0.0.1"
	}
	return "http://" + ip + ":" + strconv.Itoa(httpPort)
}
