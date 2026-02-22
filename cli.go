package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// runCLI はCLIサブコマンドを処理する
func runCLI(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("サブコマンドを指定してください: list, add, update, get, current, mcp")
	}

	switch args[0] {
	case "list":
		return cmdList(args[1:])
	case "add":
		return cmdAdd(args[1:])
	case "update":
		return cmdUpdate(args[1:])
	case "get":
		return cmdGet(args[1:])
	case "current":
		return cmdCurrent(args[1:])
	case "notify":
		return cmdNotify(args[1:])
	case "mcp":
		return cmdMCP(args[1:])
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("不明なサブコマンド: %s", args[0])
	}
}

func printUsage() {
	fmt.Println(`task_go - TUIタスク管理ツール

使い方:
  task_go              TUIモードで起動
  task_go <command>    CLIコマンドを実行

コマンド:
  list                 タスク一覧を表示
  add <name>           タスクを追加
  update <id>          タスクを更新
  get <id>             タスク詳細を表示
  current              現在のセッションのタスクを表示
  notify               セッションを確認待ち状態にする
  mcp                  MCPサーバーを起動
  help                 ヘルプを表示`)
}

// cmdList はタスク一覧を表示する
func cmdList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	statusFlag := fs.String("status", "", "ステータスでフィルタ (working, waiting, not_started, completed)")
	formatFlag := fs.String("format", "text", "出力形式 (text, json)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := LoadStore()
	if err != nil {
		return fmt.Errorf("ストアの読み込みに失敗: %w", err)
	}

	tasks := store.Tasks
	if *statusFlag != "" {
		tasks = filterByStatus(tasks, Status(*statusFlag))
	}
	SortTasks(tasks)

	if *formatFlag == "json" {
		return outputJSON(tasks)
	}
	return outputText(tasks)
}

// cmdAdd はタスクを追加する
func cmdAdd(args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	priorityFlag := fs.String("priority", "medium", "優先度 (high, medium, low)")
	descFlag := fs.String("description", "", "説明")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("タスク名を指定してください")
	}
	name := strings.Join(fs.Args(), " ")

	priority, err := parsePriority(*priorityFlag)
	if err != nil {
		return err
	}

	now := time.Now()
	var newID int

	err = SaveWithLock(func(store *TaskStore) {
		newID = store.NextID
		store.Tasks = append(store.Tasks, Task{
			ID:          store.NextID,
			Name:        name,
			Description: *descFlag,
			Status:      StatusNotStarted,
			Priority:    priority,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
		store.NextID++
	})
	if err != nil {
		return fmt.Errorf("タスクの追加に失敗: %w", err)
	}

	fmt.Fprintf(os.Stdout, "タスク #%d を追加しました: %s\n", newID, name)
	return nil
}

// cmdUpdate はタスクを更新する
func cmdUpdate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("タスクIDを指定してください")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("無効なタスクID: %s", args[0])
	}

	fs := flag.NewFlagSet("update", flag.ExitOnError)
	statusFlag := fs.String("status", "", "ステータス (working, waiting, not_started, completed)")
	priorityFlag := fs.String("priority", "", "優先度 (high, medium, low)")
	nameFlag := fs.String("name", "", "タスク名")
	descFlag := fs.String("description", "", "説明")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	found := false
	err = SaveWithLock(func(store *TaskStore) {
		for i := range store.Tasks {
			if store.Tasks[i].ID == id {
				found = true
				if *statusFlag != "" {
					store.Tasks[i].Status = Status(*statusFlag)
				}
				if *priorityFlag != "" {
					p, err := parsePriority(*priorityFlag)
					if err == nil {
						store.Tasks[i].Priority = p
					}
				}
				if *nameFlag != "" {
					store.Tasks[i].Name = *nameFlag
				}
				if *descFlag != "" {
					store.Tasks[i].Description = *descFlag
				}
				store.Tasks[i].UpdatedAt = time.Now()
				return
			}
		}
	})
	if err != nil {
		return fmt.Errorf("タスクの更新に失敗: %w", err)
	}
	if !found {
		return fmt.Errorf("タスク #%d が見つかりません", id)
	}

	fmt.Fprintf(os.Stdout, "タスク #%d を更新しました\n", id)
	return nil
}

// cmdGet はタスク詳細を表示する
func cmdGet(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("タスクIDを指定してください")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("無効なタスクID: %s", args[0])
	}

	fs := flag.NewFlagSet("get", flag.ExitOnError)
	formatFlag := fs.String("format", "text", "出力形式 (text, json)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	store, err := LoadStore()
	if err != nil {
		return fmt.Errorf("ストアの読み込みに失敗: %w", err)
	}

	for _, t := range store.Tasks {
		if t.ID == id {
			if *formatFlag == "json" {
				data, err := json.MarshalIndent(t, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}
			outputTaskText(t)
			return nil
		}
	}

	return fmt.Errorf("タスク #%d が見つかりません", id)
}

// cmdCurrent は現在のセッションに紐付いたタスクを表示する
func cmdCurrent(args []string) error {
	fs := flag.NewFlagSet("current", flag.ExitOnError)
	sessionFlag := fs.String("session", "", "セッションID")
	formatFlag := fs.String("format", "text", "出力形式 (text, json)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	sessionID := *sessionFlag
	if sessionID == "" {
		sessionID = os.Getenv("CLAUDE_SESSION_ID")
	}
	if sessionID == "" {
		return fmt.Errorf("セッションIDを --session フラグまたは CLAUDE_SESSION_ID 環境変数で指定してください")
	}

	taskID, ok := GetTaskBySession(sessionID)
	if !ok {
		fmt.Println("セッションに紐付いたタスクはありません")
		return nil
	}

	store, err := LoadStore()
	if err != nil {
		return fmt.Errorf("ストアの読み込みに失敗: %w", err)
	}

	for _, t := range store.Tasks {
		if t.ID == taskID {
			if *formatFlag == "json" {
				data, err := json.MarshalIndent(t, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}
			outputTaskText(t)
			return nil
		}
	}

	return fmt.Errorf("タスク #%d が見つかりません", taskID)
}

// cmdNotify はセッションを確認待ち状態にする（Stopフックから呼ばれる想定）
func cmdNotify(args []string) error {
	fs := flag.NewFlagSet("notify", flag.ExitOnError)
	sessionFlag := fs.String("session", "", "セッションID")
	if err := fs.Parse(args); err != nil {
		return err
	}

	sessionID := *sessionFlag
	if sessionID == "" {
		sessionID = os.Getenv("CLAUDE_SESSION_ID")
	}
	if sessionID == "" {
		return fmt.Errorf("セッションIDを --session フラグまたは CLAUDE_SESSION_ID 環境変数で指定してください")
	}

	// セッションからタスクIDを取得
	taskID, ok := GetTaskBySession(sessionID)
	if !ok {
		// 紐付けがなければ何もしない（エラーにはしない）
		return nil
	}

	// セッションのステータスを「確認待ち」に変更
	if err := UpdateSessionStatus(sessionID, StatusWaiting); err != nil {
		return fmt.Errorf("セッション更新に失敗: %w", err)
	}

	// 通知を追加
	store, err := LoadStore()
	if err == nil {
		for _, t := range store.Tasks {
			if t.ID == taskID {
				AddNotification(taskID, NotifyActionNeeded,
					fmt.Sprintf("タスク #%d「%s」: 確認してください", taskID, t.Name))
				break
			}
		}
	}

	fmt.Fprintf(os.Stdout, "タスク #%d を確認待ちにしました\n", taskID)
	return nil
}

// filterByStatus は指定ステータスのタスクだけを返す
func filterByStatus(tasks []Task, status Status) []Task {
	var result []Task
	for _, t := range tasks {
		if t.Status == status {
			result = append(result, t)
		}
	}
	return result
}

// parsePriority は文字列から優先度を返す
func parsePriority(s string) (Priority, error) {
	switch strings.ToLower(s) {
	case "high", "h":
		return PriorityHigh, nil
	case "medium", "med", "m":
		return PriorityMedium, nil
	case "low", "l":
		return PriorityLow, nil
	default:
		return PriorityMedium, fmt.Errorf("無効な優先度: %s (high, medium, low のいずれかを指定)", s)
	}
}

// outputJSON はタスク一覧をJSON形式で出力する
func outputJSON(tasks []Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// outputText はタスク一覧をテキスト形式で出力する
func outputText(tasks []Task) error {
	if len(tasks) == 0 {
		fmt.Println("タスクはありません")
		return nil
	}
	for _, t := range tasks {
		fmt.Printf("#%-4d [%s] [%s] %s\n", t.ID, t.Status.Label(), t.Priority.Label(), t.Name)
	}
	return nil
}

// outputTaskText はタスク1件をテキスト形式で詳細表示する
func outputTaskText(t Task) {
	fmt.Printf("ID:       #%d\n", t.ID)
	fmt.Printf("名前:     %s\n", t.Name)
	desc := t.Description
	if desc == "" {
		desc = "(なし)"
	}
	fmt.Printf("説明:     %s\n", desc)
	fmt.Printf("状態:     %s\n", t.Status.Label())
	fmt.Printf("優先度:   %s\n", t.Priority.Label())
	fmt.Printf("作成日時: %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("更新日時: %s\n", t.UpdatedAt.Format("2006-01-02 15:04:05"))
}
