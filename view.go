package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// スタイル定義
var (
	// セクションタイトル
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	// ステータスのスタイル
	statusWorkingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46")).  // 緑
				Bold(true)
	statusWaitingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")). // オレンジ
				Bold(true)
	statusNotStartedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")). // グレー
				Bold(true)

	// 優先度のスタイル
	priorityHighStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")). // 赤
				Bold(true)
	priorityMediumStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("226")). // 黄
				Bold(true)
	priorityLowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")). // グレー
				Bold(true)

	// 選択行
	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")) // 黄色

	// 通常行
	normalStyle = lipgloss.NewStyle()

	// ヘルプセクション
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	// ボーダー
	sectionStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)
)

func (m model) View() string {
	activeSection := m.renderActiveSection()
	listSection := m.renderListSection()
	helpSection := m.renderHelpSection()

	sections := []string{activeSection, listSection}
	if m.showDetail {
		sections = append(sections, m.renderDetailSection())
	}
	sections = append(sections, helpSection)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderActiveSection は作業中タスクセクションを描画する
func (m model) renderActiveSection() string {
	title := titleStyle.Render(" 作業中タスク ")

	var lines []string
	if len(m.working) == 0 {
		lines = append(lines, helpStyle.Render("(なし)"))
	} else {
		for _, t := range m.working {
			lines = append(lines, fmt.Sprintf("#%d %s", t.ID, t.Name))
		}
	}

	content := strings.Join(lines, "\n")
	w := m.contentWidth()
	box := sectionStyle.Width(w).Render(content)

	return title + "\n" + box
}

// renderListSection はタスク一覧セクションを描画する
func (m model) renderListSection() string {
	title := titleStyle.Render(" タスク一覧 ")

	if len(m.visible) == 0 {
		content := helpStyle.Render("タスクがありません。'a'キーで追加してください。")
		w := m.contentWidth()
		box := sectionStyle.Width(w).Render(content)
		return title + "\n" + box
	}

	maxVisible := m.listHeight()
	end := m.offset + maxVisible
	if end > len(m.visible) {
		end = len(m.visible)
	}

	var lines []string
	for i := m.offset; i < end; i++ {
		t := m.visible[i]
		line := formatTaskLine(t, i == m.cursor)
		lines = append(lines, line)
	}

	// スクロールインジケーター
	if m.offset > 0 {
		lines = append([]string{helpStyle.Render("  ↑ 上にスクロール")}, lines...)
	}
	if end < len(m.visible) {
		lines = append(lines, helpStyle.Render("  ↓ 下にスクロール"))
	}

	content := strings.Join(lines, "\n")
	w := m.contentWidth()
	box := sectionStyle.Width(w).Render(content)

	return title + "\n" + box
}

// renderDetailSection は選択中タスクの詳細パネルを描画する
func (m model) renderDetailSection() string {
	title := titleStyle.Render(" タスク詳細 ")

	if len(m.visible) == 0 || m.cursor >= len(m.visible) {
		content := helpStyle.Render("(タスクが選択されていません)")
		w := m.contentWidth()
		box := sectionStyle.Width(w).Render(content)
		return title + "\n" + box
	}

	t := m.visible[m.cursor]
	desc := t.Description
	if desc == "" {
		desc = "(なし)"
	}

	lines := []string{
		fmt.Sprintf("ID:     #%d", t.ID),
		fmt.Sprintf("名前:   %s", t.Name),
		fmt.Sprintf("説明:   %s", desc),
		fmt.Sprintf("状態:   %s", t.Status.Label()),
		fmt.Sprintf("優先度: %s", t.Priority.Label()),
		fmt.Sprintf("作成:   %s", t.CreatedAt.Format("2006-01-02 15:04")),
		fmt.Sprintf("更新:   %s", t.UpdatedAt.Format("2006-01-02 15:04")),
	}

	content := strings.Join(lines, "\n")
	w := m.contentWidth()
	box := sectionStyle.Width(w).Render(content)
	return title + "\n" + box
}

// renderHelpSection はショートカットキー一覧を描画する
func (m model) renderHelpSection() string {
	line1 := "a:追加 e:編集 t:作業中 w:確認待ち u:未着手 f:完了"
	line2 := "h:高 m:中 l:低 j/↓:下 k/↑:上 Space:詳細 q:終了"
	content := helpStyle.Render(line1) + "\n" + helpStyle.Render(line2)

	w := m.contentWidth()
	return sectionStyle.Width(w).Render(content)
}

// formatTaskLine はタスク1行分をフォーマットする
func formatTaskLine(t Task, selected bool) string {
	// ステータスラベル
	statusLabel := formatStatus(t.Status)
	// 優先度ラベル
	priorityLabel := formatPriority(t.Priority)

	line := fmt.Sprintf("[%s] [%s] #%d %s", statusLabel, priorityLabel, t.ID, t.Name)

	if selected {
		return cursorStyle.Render("> " + line)
	}
	return normalStyle.Render("  " + line)
}

// formatStatus はステータスに応じたスタイル付きラベルを返す
func formatStatus(s Status) string {
	label := s.Label()
	switch s {
	case StatusWorking:
		return statusWorkingStyle.Render(label)
	case StatusWaiting:
		return statusWaitingStyle.Render(label)
	case StatusNotStarted:
		return statusNotStartedStyle.Render(label)
	default:
		return label
	}
}

// formatPriority は優先度に応じたスタイル付きラベルを返す
func formatPriority(p Priority) string {
	label := p.Label()
	switch p {
	case PriorityHigh:
		return priorityHighStyle.Render(label)
	case PriorityMedium:
		return priorityMediumStyle.Render(label)
	case PriorityLow:
		return priorityLowStyle.Render(label)
	default:
		return label
	}
}

// contentWidth はコンテンツエリアの幅を返す
func (m model) contentWidth() int {
	w := m.width - 4 // ボーダーとパディング分
	if w < 40 {
		return 40
	}
	return w
}
