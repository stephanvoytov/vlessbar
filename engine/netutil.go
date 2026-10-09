package main

import (
	"net"
	"strconv"
)

// localIPv4 returns the first non-loopback IPv4 address of this machine, or ""
// when none is found. Used to display the LAN gateway address.
func localIPv4() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			return ip4.String()
		}
	}
	return ""
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
