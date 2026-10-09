package main

import (
	"fmt"
	"os"
)

const version = "0.3.0"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		if err := launchGUI(); err != nil {
			usage()
		}
		return
	}
	if err := dispatch(args[0], args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func dispatch(cmd string, args []string) error {
	switch cmd {
	case "version", "-v", "--version":
		fmt.Println(version)
	case "help", "-h", "--help":
		usage()
	case "hwid":
		return cmdHwid()
	case "sub-add":
		return cmdSubAdd(args)
	case "add":
		return cmdAdd(args)
	case "sub-update":
		return cmdSubUpdate()
	case "list":
		return cmdList()
	case "set":
		return cmdSet(args)
	case "up":
		return cmdUp()
	case "down":
		return cmdDown()
	case "lan":
		return cmdLan(args)
	case "check-update":
		return cmdCheckUpdate()
	case "conn", "conn-check":
		return cmdConn()
	case "ping":
		return cmdPing()
	case "ping-tunnel":
		return cmdPingThroughTunnel()
	case "ip":
		return cmdIP()
	case "status":
		return cmdStatus()
	case "gui-state":
		return cmdGuiState()
	case "gui-servers":
		return cmdGuiServers()
	default:
		return fmt.Errorf("unknown command: %s (try: help)", cmd)
	}
	return nil
}

func usage() {
	fmt.Printf(`vlessbar engine %s  -  minimal VLESS client (macOS 10.13+ / Windows 7+)

usage:
  hwid                 print this device HWID
  sub-add <url>        set subscription URL and fetch servers
  add <vless://...>    add a single server manually (kept across updates)
  sub-update           re-fetch subscription
  list                 list servers as "<idx><TAB><name>" ('*' marks selected)
  set <idx>            select server by index
  up                   start tunnel for selected server + set system proxy
  down                 stop tunnel + clear system proxy
  lan on|off           allow other devices on the LAN to use this proxy
  check-update         check if a newer app build is available
  conn                 проверить IP + пинг сервера (без туннеля)
  ping                 пинг выбранного сервера напрямую
  ping-tunnel          пинг через VPN-туннель (нужен запущенный туннель)
  ip                   показать текущий публичный IP
  hwid                 print this device HWID (CLI only)
  status               print status as JSON (CLI only)
  version              print version
`, version)
}
