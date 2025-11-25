package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultISOPath   = "/var/lib/vz/template/iso/"
	defaultCachePath = "/var/lib/vz/template/cache"
)

func ensureDirs(paths ...string) error {
	for _, p := range paths {
		if err := os.MkdirAll(p, 0755); err != nil {
			return err
		}
	}
	return nil
}

func defaultStatusPath(cachePath, name string) string {
	return filepath.Join(cachePath, name)
}

func parseWindowsVersion(v string) (int, error) {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "11", "win11", "windows11":
		return 0, nil
	case "10", "win10", "windows10":
		return 1, nil
	default:
		return -1, fmt.Errorf("unknown windows version: %s", v)
	}
}

func parseUbuntuVersion(v string) (int, error) {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "22.04-desktop", "22.04d", "2204d":
		return 0, nil
	case "22.04-server", "22.04s", "2204s":
		return 1, nil
	case "24.10-desktop", "24.10d", "2410d":
		return 2, nil
	case "24.10-server", "24.10s", "2410s":
		return 3, nil
	case "25.04-desktop", "25.04d", "2504d":
		return 4, nil
	case "25.04-server", "25.04s", "2504s":
		return 5, nil
	default:
		return -1, fmt.Errorf("unknown ubuntu version: %s", v)
	}
}

func parseIstoreVersion(v string) (int, error) {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "24.10", "2410", "24":
		return 0, nil
	case "22.03", "2203", "22":
		return 1, nil
	default:
		return -1, fmt.Errorf("unknown istore version: %s", v)
	}
}
