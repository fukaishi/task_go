package main

import (
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// urlPattern はHTTP/HTTPS URLにマッチする正規表現（ASCII範囲のみ）
var urlPattern = regexp.MustCompile(`https?://[A-Za-z0-9\-._~:/?#\[\]@!$&'()*+,;=%]+`)

// extractURLs はテキストからURLを抽出し、重複を除いた一覧を返す
func extractURLs(text string) []string {
	matches := urlPattern.FindAllString(text, -1)
	seen := make(map[string]bool)
	var result []string
	for _, u := range matches {
		u = trimTrailingPunct(u)
		if seen[u] {
			continue
		}
		seen[u] = true
		result = append(result, u)
	}
	return result
}

// trimTrailingPunct はURLの末尾にある句読点や閉じ括弧を除去する
func trimTrailingPunct(u string) string {
	for len(u) > 0 {
		last := u[len(u)-1]
		if last == '.' || last == ',' || last == ';' || last == ':' ||
			last == ')' || last == ']' || last == '>' ||
			last == '\'' || last == '"' ||
			// 日本語句読点（UTF-8マルチバイト）
			strings.HasSuffix(u, "。") || strings.HasSuffix(u, "、") {
			if strings.HasSuffix(u, "。") {
				u = strings.TrimSuffix(u, "。")
			} else if strings.HasSuffix(u, "、") {
				u = strings.TrimSuffix(u, "、")
			} else {
				u = u[:len(u)-1]
			}
			continue
		}
		break
	}
	return u
}

// openURL はデフォルトブラウザでURLを開く
func openURL(url string) error {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	default:
		cmd = "xdg-open"
	}
	return exec.Command(cmd, url).Start()
}

// truncateURL はURLを指定幅に収まるよう省略する
func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	if maxLen <= 3 {
		return url[:maxLen]
	}
	return url[:maxLen-3] + "..."
}
