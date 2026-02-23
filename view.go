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

	// セッション連携インジケーター
	sessionWorkingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")). // 水色
				Bold(true)
	sessionActionNeededStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("214")). // オレンジ
					Bold(true)
)

func (m model) View() string {
	activeSection := m.renderActiveSection()

	var sections []string
	sections = append(sections, activeSection)
	sections = append(sections, m.renderListSection())
	if m.urlSelectMode {
		sections = append(sections, m.renderURLSelectSection())
	} else if m.showDetail {
		sections = append(sections, m.renderDetailSection())
	}
	if m.statusMessage != "" {
		sections = append(sections, m.renderStatusMessage())
	} else {
		sections = append(sections, m.renderHelpSection())
	}

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
			line := fmt.Sprintf("#%d %s", t.ID, t.Name)
			// セッション紐付けを確認し、状態に応じたインジケーターを表示
			if binding, ok := m.findSessionBinding(t.ID); ok {
				switch binding.Status {
				case StatusWorking:
					if binding.Message != "" {
						line += "  " + sessionWorkingStyle.Render(fmt.Sprintf(">> Claude稼働中（%s）", binding.Message))
					} else {
						line += "  " + sessionWorkingStyle.Render(">> Claude稼働中")
					}
				case StatusWaiting:
					line += "  " + sessionActionNeededStyle.Render("!! 確認してください")
				}
			}
			lines = append(lines, line)
		}
	}

	content := strings.Join(lines, "\n")
	w := m.contentWidth()
	box := sectionStyle.Width(w).Render(content)

	return title + "\n" + box
}

// findSessionBinding はタスクIDに対応するセッションバインディングを返す
// 同一タスクに複数セッションがある場合、最も緊急度の高いものを返す
// 優先度: waiting > working+message > working
func (m model) findSessionBinding(taskID int) (SessionBinding, bool) {
	var best SessionBinding
	found := false
	for _, b := range m.sessions.Bindings {
		if b.TaskID != taskID {
			continue
		}
		if !found {
			best = b
			found = true
			continue
		}
		if b.Status == StatusWaiting {
			best = b
		} else if b.Message != "" && best.Status != StatusWaiting {
			if best.Message == "" || best.Status != StatusWorking {
				best = b
			}
		}
	}
	return best, found
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
		fmt.Sprintf("状態:   %s", t.Status.Label()),
		fmt.Sprintf("優先度: %s", t.Priority.Label()),
		fmt.Sprintf("名前:   %s", t.Name),
		formatDescription(desc),
		fmt.Sprintf("作成:   %s", t.CreatedAt.Format("2006-01-02 15:04")),
		fmt.Sprintf("更新:   %s", t.UpdatedAt.Format("2006-01-02 15:04")),
	}

	content := strings.Join(lines, "\n")
	w := m.contentWidth()
	box := sectionStyle.Width(w).Render(content)
	return title + "\n" + box
}

// formatDescription は説明文を複数行対応でフォーマットする
func formatDescription(desc string) string {
	parts := strings.Split(desc, "\n")
	if len(parts) <= 1 {
		return fmt.Sprintf("説明:   %s", desc)
	}
	lines := make([]string, len(parts))
	lines[0] = fmt.Sprintf("説明:   %s", parts[0])
	for i := 1; i < len(parts); i++ {
		lines[i] = fmt.Sprintf("        %s", parts[i])
	}
	return strings.Join(lines, "\n")
}

// renderURLSelectSection はURL選択パネルを描画する
func (m model) renderURLSelectSection() string {
	title := titleStyle.Render(" URL選択 ")

	var lines []string
	for i, u := range m.urlCandidates {
		num := fmt.Sprintf("%d. ", i+1)
		display := truncateURL(u, m.contentWidth()-6)
		if i == m.urlCursor {
			lines = append(lines, cursorStyle.Render("> "+num+display))
		} else {
			lines = append(lines, normalStyle.Render("  "+num+display))
		}
	}
	lines = append(lines, "")
	lines = append(lines, helpStyle.Render("j/k:移動 Enter:開く 1-9:番号選択 Esc:戻る"))

	content := strings.Join(lines, "\n")
	w := m.contentWidth()
	box := sectionStyle.Width(w).Render(content)
	return title + "\n" + box
}

// renderStatusMessage はステータスメッセージをヘルプ領域の代わりに表示する
func (m model) renderStatusMessage() string {
	content := helpStyle.Render(m.statusMessage)
	w := m.contentWidth()
	return sectionStyle.Width(w).Render(content)
}

// renderHelpSection はショートカットキー一覧を描画する
func (m model) renderHelpSection() string {
	line1 := "a:追加 e:編集 t:作業中 w:確認待ち u:未着手 f:完了 o:URL"
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

	line := fmt.Sprintf("#%d [%s] [%s] %s", t.ID, statusLabel, priorityLabel, t.Name)

	if selected {
		return cursorStyle.Render("> ") + line
	}
	return "  " + line
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
