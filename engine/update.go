package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// App self-update: checks this project's GitHub releases and can replace the
// running executable with a newer build. It never touches the Xray core (the
// core is pinned on purpose so the client keeps working on old systems).

const releasesAPI = "https://api.github.com/repos/stephanvoytov/vlessbar/releases/latest"

type ghRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// compareVersions compares dotted numeric versions (e.g. 0.3.0). Returns -1, 0
// or 1. Non-numeric segments count as 0.
func compareVersions(a, b string) int {
	pa := strings.Split(normalizeVersion(a), ".")
	pb := strings.Split(normalizeVersion(b), ".")
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var va, vb int
		if i < len(pa) {
			va, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			vb, _ = strconv.Atoi(pb[i])
		}
		if va != vb {
			if va < vb {
				return -1
			}
			return 1
		}
	}
	return 0
}

func fetchLatestRelease() (*ghRelease, error) {
	req, err := http.NewRequest(http.MethodGet, releasesAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "VLessBar/"+version)
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github HTTP %d", resp.StatusCode)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// pickAsset returns the name and URL of the asset matching this platform.
func (r *ghRelease) pickAsset() (string, string) {
	for _, a := range r.Assets {
		n := strings.ToLower(a.Name)
		switch runtime.GOOS {
		case "windows":
			if strings.Contains(n, "win") {
				return a.Name, a.URL
			}
		case "darwin":
			if strings.Contains(n, "mac") || strings.Contains(n, "osx") || strings.Contains(n, "darwin") {
				return a.Name, a.URL
			}
		default:
			if strings.Contains(n, "linux") {
				return a.Name, a.URL
			}
		}
	}
	return "", ""
}

// checkUpdate returns a human-readable status and, when an update is available,
// the asset URL and its version.
func checkUpdate() (status, url, ver string) {
	rel, err := fetchLatestRelease()
	if err != nil {
		return "Не удалось проверить обновление: " + err.Error(), "", ""
	}
	latest := normalizeVersion(rel.TagName)
	switch compareVersions(version, latest) {
	case 0:
		return fmt.Sprintf("У вас последняя версия (%s).", version), "", ""
	case 1:
		return fmt.Sprintf("У вас версия новее последнего релиза (%s).", version), "", ""
	}
	name, assetURL := rel.pickAsset()
	if assetURL == "" {
		return fmt.Sprintf("Доступна версия %s, но нет сборки для %s.", latest, runtime.GOOS), "", ""
	}
	return fmt.Sprintf("Доступна версия %s (%s).", latest, name), assetURL, latest
}

func downloadFile(url, dst string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "VLessBar/"+version)
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download HTTP %d", resp.StatusCode)
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// applyUpdate downloads the new build and schedules replacing this executable
// after the current process exits.
func applyUpdate(url string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	newPath := exe + ".new"
	if err := downloadFile(url, newPath); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return scheduleReplaceWindows(newPath, exe)
	}
	return scheduleReplaceUnix(newPath, exe)
}

func scheduleReplaceWindows(newPath, exe string) error {
	base := filepath.Base(exe)
	bat := filepath.Join(os.TempDir(), "vlessbar-update.bat")
	script := "@echo off\r\n" +
		":wait\r\n" +
		"tasklist /fi \"imagename eq " + base + "\" | find /i \"" + base + "\" >nul && (ping -n 2 127.0.0.1 >nul & goto wait)\r\n" +
		"move /y \"" + newPath + "\" \"" + exe + "\"\r\n" +
		"start \"\" \"" + exe + "\"\r\n" +
		"(goto) 2>nul & del \"%~f0\"\r\n"
	if err := os.WriteFile(bat, []byte(script), 0o600); err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/c", "start", "", "/min", bat)
	return cmd.Start()
}

func scheduleReplaceUnix(newPath, exe string) error {
	sh := filepath.Join(os.TempDir(), "vlessbar-update.sh")
	script := "#!/bin/sh\n" +
		"while pgrep -f \"$(basename \"" + exe + "\")\" >/dev/null 2>&1; do sleep 1; done\n" +
		"chmod +x \"" + newPath + "\"\n" +
		"mv -f \"" + newPath + "\" \"" + exe + "\"\n" +
		"open \"" + exe + "\" 2>/dev/null || \"" + exe + "\" >/dev/null 2>&1 &\n" +
		"rm -f \"$0\"\n"
	if err := os.WriteFile(sh, []byte(script), 0o700); err != nil {
		return err
	}
	return exec.Command("/bin/sh", sh).Start()
}

// xrayVersion returns the version line of the bundled Xray core (short form).
func xrayVersion() string {
	bin, err := xrayBinary()
	if err != nil {
		return "не найдено"
	}
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		return "неизвестно"
	}
	fields := strings.Fields(strings.SplitN(string(out), "\n", 2)[0])
	if len(fields) >= 2 {
		return fields[0] + " " + fields[1]
	}
	return strings.TrimSpace(string(out))
}
