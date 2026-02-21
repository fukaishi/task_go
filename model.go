package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// tickInterval は定期リロードの間隔
const tickInterval = 1 * time.Second

// tickMsg は定期リロード用のメッセージ
type tickMsg time.Time

// editorFinishedMsg はエディタ終了後のメッセージ
type editorFinishedMsg struct {
	name        string
	description string
	editID      int // 0なら新規、0以外なら編集対象のID
}

// model はBubble Teaモデル
type model struct {
	store        TaskStore
	visible      []Task // 完了以外のタスク(ソート済み)
	working      []Task // 作業中タスク(ソート済み)
	cursor       int    // タスク一覧のカーソル位置
	offset       int    // タスク一覧のスクロールオフセット
	width        int    // ターミナル幅
	height       int    // ターミナル高さ
	showDetail   bool   // タスク詳細パネルの表示フラグ
	lastLoadTime time.Time
	err          error
}

func newModel() model {
	store, _ := LoadStore()
	visible := FilterVisible(store.Tasks)
	SortTasks(visible)
	working := FilterWorking(store.Tasks)
	SortTasks(working)

	return model{
		store:        store,
		visible:      visible,
		working:      working,
		lastLoadTime: FileModTime(),
		width:        80,
		height:       24,
	}
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		// ファイルの更新日時をチェックし、変更があればリロード
		modTime := FileModTime()
		if modTime.After(m.lastLoadTime) {
			m.reload()
			m.lastLoadTime = modTime
		}
		return m, tickCmd()

	case editorFinishedMsg:
		if msg.name == "" {
			// Name空欄ならキャンセル
			return m, tickCmd()
		}
		now := time.Now()
		err := SaveWithLock(func(store *TaskStore) {
			if msg.editID == 0 {
				// 新規追加
				task := Task{
					ID:          store.NextID,
					Name:        msg.name,
					Description: msg.description,
					Status:      StatusNotStarted,
					Priority:    PriorityMedium,
					CreatedAt:   now,
					UpdatedAt:   now,
				}
				store.NextID++
				store.Tasks = append(store.Tasks, task)
			} else {
				// 編集
				for i := range store.Tasks {
					if store.Tasks[i].ID == msg.editID {
						store.Tasks[i].Name = msg.name
						store.Tasks[i].Description = msg.description
						store.Tasks[i].UpdatedAt = now
						break
					}
				}
			}
		})
		if err != nil {
			m.err = err
		}
		m.reload()
		return m, tickCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}
			return m, nil

		case "down", "j":
			if m.cursor < len(m.visible)-1 {
				m.cursor++
				maxVisible := m.listHeight()
				if m.cursor >= m.offset+maxVisible {
					m.offset = m.cursor - maxVisible + 1
				}
			}
			return m, nil

		case "a":
			// タスク追加: エディタ起動
			return m, openEditorCmd(0, "", "")

		case "e":
			// タスク編集: エディタ起動
			if len(m.visible) > 0 && m.cursor < len(m.visible) {
				t := m.visible[m.cursor]
				return m, openEditorCmd(t.ID, t.Name, t.Description)
			}
			return m, nil

		case "t":
			return m, m.setStatus(StatusWorking)
		case "w":
			return m, m.setStatus(StatusWaiting)
		case "u":
			return m, m.setStatus(StatusNotStarted)
		case "f":
			return m, m.setStatus(StatusCompleted)

		case " ":
			m.showDetail = !m.showDetail
			return m, nil

		case "h":
			return m, m.setPriority(PriorityHigh)
		case "m":
			return m, m.setPriority(PriorityMedium)
		case "l":
			return m, m.setPriority(PriorityLow)
		}
	}

	return m, nil
}

// setStatus は選択中のタスクのステータスを変更する
func (m *model) setStatus(status Status) tea.Cmd {
	if len(m.visible) == 0 || m.cursor >= len(m.visible) {
		return nil
	}
	taskID := m.visible[m.cursor].ID
	err := SaveWithLock(func(store *TaskStore) {
		for i := range store.Tasks {
			if store.Tasks[i].ID == taskID {
				store.Tasks[i].Status = status
				store.Tasks[i].UpdatedAt = time.Now()
				break
			}
		}
	})
	if err != nil {
		m.err = err
	}
	m.reload()
	// 完了にした場合、カーソルが範囲外にならないよう調整
	if m.cursor >= len(m.visible) && len(m.visible) > 0 {
		m.cursor = len(m.visible) - 1
	}
	return nil
}

// setPriority は選択中のタスクの優先度を変更する
func (m *model) setPriority(priority Priority) tea.Cmd {
	if len(m.visible) == 0 || m.cursor >= len(m.visible) {
		return nil
	}
	taskID := m.visible[m.cursor].ID
	err := SaveWithLock(func(store *TaskStore) {
		for i := range store.Tasks {
			if store.Tasks[i].ID == taskID {
				store.Tasks[i].Priority = priority
				store.Tasks[i].UpdatedAt = time.Now()
				break
			}
		}
	})
	if err != nil {
		m.err = err
	}
	m.reload()
	// ソート後にカーソルが元のタスクを追うよう調整
	for i, t := range m.visible {
		if t.ID == taskID {
			m.cursor = i
			break
		}
	}
	return nil
}

// reload はストアを再読み込みし、visible/workingを更新する
func (m *model) reload() {
	store, err := LoadStore()
	if err != nil {
		m.err = err
		return
	}
	m.store = store
	m.visible = FilterVisible(store.Tasks)
	SortTasks(m.visible)
	m.working = FilterWorking(store.Tasks)
	SortTasks(m.working)
	m.lastLoadTime = FileModTime()

	// カーソル位置の調整
	if m.cursor >= len(m.visible) {
		if len(m.visible) > 0 {
			m.cursor = len(m.visible) - 1
		} else {
			m.cursor = 0
		}
	}
}

// listHeight はタスク一覧に使える行数を返す
func (m model) listHeight() int {
	// 作業中セクション: 枠線1 + タスク数(最低1) + 空行1
	workingLines := 1 + max(len(m.working), 1) + 1
	// ヘルプセクション: 枠線1 + 2行 + 枠線1
	helpLines := 4
	// タスク一覧セクション: 枠線1
	listHeaderLines := 1
	// 詳細パネル: タイトル1 + 枠線上1 + 7行 + 枠線下1
	detailLines := 0
	if m.showDetail {
		detailLines = 10
	}
	overhead := workingLines + helpLines + listHeaderLines + detailLines
	available := m.height - overhead
	if available < 3 {
		return 3
	}
	return available
}

// openEditorCmd はエディタを起動するコマンドを返す
func openEditorCmd(editID int, name, description string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "task_go_edit.txt")

	// エディタ用ファイルを作成
	content := fmt.Sprintf("Name: %s\nDescription: %s\n---\n上記のName, Descriptionを編集してください。\nこの行(---)以降は無視されます。\n", name, description)
	os.WriteFile(tmpFile, []byte(content), 0644)

	c := exec.Command(editor, tmpFile)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return editorFinishedMsg{}
		}
		data, err := os.ReadFile(tmpFile)
		if err != nil {
			return editorFinishedMsg{}
		}
		os.Remove(tmpFile)
		n, d := parseEditorFile(string(data))
		return editorFinishedMsg{name: n, description: d, editID: editID}
	})
}

// parseEditorFile はエディタファイルの内容をパースする
func parseEditorFile(content string) (name, description string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "---") {
			break
		}
		if strings.HasPrefix(line, "Name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		} else if strings.HasPrefix(line, "Description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		}
	}
	return
}
