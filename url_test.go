package main

import (
	"testing"
)

func TestExtractURLs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "単一URL",
			input: "詳細は https://example.com を参照",
			want:  []string{"https://example.com"},
		},
		{
			name:  "複数URL",
			input: "http://foo.com と https://bar.com/path",
			want:  []string{"http://foo.com", "https://bar.com/path"},
		},
		{
			name:  "URLなし",
			input: "URLは含まれていません",
			want:  nil,
		},
		{
			name:  "重複URL",
			input: "https://example.com と https://example.com",
			want:  []string{"https://example.com"},
		},
		{
			name:  "末尾ピリオド",
			input: "参照: https://example.com/page.",
			want:  []string{"https://example.com/page"},
		},
		{
			name:  "末尾カンマ",
			input: "https://example.com/a, https://example.com/b,",
			want:  []string{"https://example.com/a", "https://example.com/b"},
		},
		{
			name:  "括弧で囲まれたURL",
			input: "(https://example.com/path)",
			want:  []string{"https://example.com/path"},
		},
		{
			name:  "クエリパラメータ付きURL",
			input: "https://example.com/search?q=test&page=1",
			want:  []string{"https://example.com/search?q=test&page=1"},
		},
		{
			name:  "フラグメント付きURL",
			input: "https://example.com/docs#section-1",
			want:  []string{"https://example.com/docs#section-1"},
		},
		{
			name:  "日本語句読点の後",
			input: "URLはhttps://example.com。こちらも参照",
			want:  []string{"https://example.com"},
		},
		{
			name:  "Redmine風URL",
			input: "https://redmine.example.com/issues/12345",
			want:  []string{"https://redmine.example.com/issues/12345"},
		},
		{
			name:  "空文字列",
			input: "",
			want:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractURLs(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("extractURLs(%q) = %v (len=%d), want %v (len=%d)", tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractURLs(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestTruncateURL(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		maxLen int
		want   string
	}{
		{
			name:   "短いURL",
			url:    "https://example.com",
			maxLen: 50,
			want:   "https://example.com",
		},
		{
			name:   "省略が必要",
			url:    "https://example.com/very/long/path/to/resource",
			maxLen: 30,
			want:   "https://example.com/very/lo...",
		},
		{
			name:   "ちょうど一致",
			url:    "https://example.com",
			maxLen: 19,
			want:   "https://example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateURL(tt.url, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateURL(%q, %d) = %q, want %q", tt.url, tt.maxLen, got, tt.want)
			}
		})
	}
}
