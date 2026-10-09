package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	socksPort = 10808
	httpPort  = 10809
)

// xrayBinary locates the bundled xray executable next to this engine binary.
func xrayBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	candidates := []string{
		filepath.Join(dir, "xray"),
		filepath.Join(dir, "..", "Resources", "xray"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("xray binary not found near %s", dir)
}

// buildXrayConfig turns a Vless link into an Xray client config with local
// SOCKS/HTTP inbounds.
func buildXrayConfig(v *Vless) map[string]interface{} {
	stream := map[string]interface{}{
		"network":  v.Network,
		"security": v.Security,
	}
	switch v.Security {
	case "tls":
		tls := map[string]interface{}{"serverName": v.SNI}
		if v.Fingerprint != "" {
			tls["fingerprint"] = v.Fingerprint
		}
		if v.ALPN != "" {
			tls["alpn"] = strings.Split(v.ALPN, ",")
		}
		if v.AllowInsecure {
			tls["allowInsecure"] = true
		}
		stream["tlsSettings"] = tls
	case "reality":
		stream["realitySettings"] = map[string]interface{}{
			"serverName":  v.SNI,
			"fingerprint": defaultStr(v.Fingerprint, "chrome"),
			"publicKey":   v.PublicKey,
			"shortId":     v.ShortID,
			"spiderX":     v.SpiderX,
		}
	}
	switch v.Network {
	case "ws":
		ws := map[string]interface{}{"path": defaultStr(v.Path, "/")}
		if v.Host != "" {
			ws["headers"] = map[string]string{"Host": v.Host}
		}
		stream["wsSettings"] = ws
	case "grpc":
		stream["grpcSettings"] = map[string]interface{}{"serviceName": v.ServiceName}
	case "http", "h2":
		stream["httpSettings"] = map[string]interface{}{
			"path": defaultStr(v.Path, "/"),
			"host": []string{v.Host},
		}
	case "tcp":
		if v.HeaderType == "http" {
			stream["tcpSettings"] = map[string]interface{}{
				"header": map[string]interface{}{"type": "http"},
			}
		}
	}

	user := map[string]interface{}{"id": v.UUID, "encryption": "none"}
	if v.Flow != "" {
		user["flow"] = v.Flow
	}

	return map[string]interface{}{
		"log": map[string]interface{}{"loglevel": "warning"},
		"inbounds": []map[string]interface{}{
			{"tag": "socks", "listen": "127.0.0.1", "port": socksPort, "protocol": "socks",
				"settings": map[string]interface{}{"udp": true}},
			{"tag": "http", "listen": "127.0.0.1", "port": httpPort, "protocol": "http"},
		},
		"outbounds": []map[string]interface{}{
			{
				"tag": "proxy", "protocol": "vless",
				"settings": map[string]interface{}{"vnext": []map[string]interface{}{
					{"address": v.Address, "port": v.Port, "users": []interface{}{user}},
				}},
				"streamSettings": stream,
			},
		},
	}
}

// writeXrayConfig serializes the config for v into dir/config.json.
func writeXrayConfig(dir string, v *Vless) (string, error) {
	data, err := json.MarshalIndent(buildXrayConfig(v), "", "  ")
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return "", err
	}
	return p, nil
}
