package main

import (
	"fmt"
	"os"
)

const version = "0.1.0"

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
	fmt.Printf(`vlessbar engine %s  -  minimal VLESS client for macOS 10.13+

usage:
  hwid                 print this device HWID
  sub-add <url>        set subscription URL and fetch servers
  sub-update           re-fetch subscription
  list                 list servers as "<idx><TAB><name>" ('*' marks selected)
  set <idx>            select server by index
  up                   start tunnel for selected server + set system proxy
  down                 stop tunnel + clear system proxy
  status               print status as JSON
  version              print version
`, version)
}
