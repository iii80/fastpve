package quickget

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

//go:embed scripts/*
var scriptFiles embed.FS

func CreateQuickGet() (string, error) {
	f, err := scriptFiles.Open("scripts/quickget")
	if err != nil {
		return "", err
	}
	defer f.Close()

	tmpFile, err := os.CreateTemp("", "script-*.sh")
	if err != nil {
		return "", err
	}
	io.Copy(tmpFile, f)
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0700); err != nil {
		return "", err
	}
	return tmpFile.Name(), nil
}

func ParseLastURL(input string) (string, error) {
	if strings.Contains(input, "failing HTTP status code") {
		return "", fmt.Errorf("HTTP status code error")
	}

	// 使用正则表达式匹配最后一个 xxx: 后的 URL
	re := regexp.MustCompile(`[^:]+:\s+(https?://[^\s]+)\s*$`)

	// 查找所有匹配项
	matches := re.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no URL found in input")
	}

	// 获取最后一个匹配的 URL
	lastMatch := matches[len(matches)-1]
	if len(lastMatch) < 2 {
		return "", fmt.Errorf("invalid URL format")
	}

	// 清理 URL 中的换行符和空格
	url := strings.TrimSpace(lastMatch[1])
	if strings.Contains(url, "virtio-win") {
		return "", errors.New("URL contains virtio-win")
	}

	return url, nil
}

func GetSystemURL(ctx context.Context, quickget string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, quickget, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	input := strings.Replace(string(output), "\n", "", -1)
	return ParseLastURL(input)
}

func PveReverseScripts() ([]byte, error) {
	f, err := scriptFiles.Open("pve-reverse.sh")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
