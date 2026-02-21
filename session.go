package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// SessionBinding はセッションとタスクの紐付けを表す
type SessionBinding struct {
	SessionID string    `json:"session_id"`
	TaskID    int       `json:"task_id"`
	Status    Status    `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

// SessionStore はセッション紐付け一覧を保持する
type SessionStore struct {
	Bindings []SessionBinding `json:"bindings"`
}

func sessionPath() string {
	return filepath.Join(storeDir(), "sessions.json")
}

func sessionLockPath() string {
	return filepath.Join(storeDir(), "sessions.lock")
}

// LoadSessionStore はJSONファイルからSessionStoreを読み込む
func LoadSessionStore() (SessionStore, error) {
	var store SessionStore

	data, err := os.ReadFile(sessionPath())
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return store, err
	}

	if err := json.Unmarshal(data, &store); err != nil {
		return store, err
	}
	return store, nil
}

// SaveSessionStore はSessionStoreをJSONファイルにアトミックに保存する
func SaveSessionStore(store SessionStore) error {
	dir := storeDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := sessionPath() + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, sessionPath())
}

// SaveSessionWithLock は排他制御付きでセッションストアを変更・保存する
func SaveSessionWithLock(modify func(*SessionStore)) error {
	dir := storeDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	lockFile, err := os.OpenFile(sessionLockPath(), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	store, err := LoadSessionStore()
	if err != nil {
		return err
	}

	modify(&store)

	return SaveSessionStore(store)
}

// BindSession はセッションをタスクに紐付ける（既存の同セッション紐付けは上書き）
func BindSession(sessionID string, taskID int, status Status) error {
	return SaveSessionWithLock(func(store *SessionStore) {
		// 既存の紐付けを探して上書き
		for i := range store.Bindings {
			if store.Bindings[i].SessionID == sessionID {
				store.Bindings[i].TaskID = taskID
				store.Bindings[i].Status = status
				store.Bindings[i].StartedAt = time.Now()
				return
			}
		}
		// 新規追加
		store.Bindings = append(store.Bindings, SessionBinding{
			SessionID: sessionID,
			TaskID:    taskID,
			Status:    status,
			StartedAt: time.Now(),
		})
	})
}

// UnbindSession はセッションの紐付けを削除する
func UnbindSession(sessionID string) error {
	return SaveSessionWithLock(func(store *SessionStore) {
		for i := range store.Bindings {
			if store.Bindings[i].SessionID == sessionID {
				store.Bindings = append(store.Bindings[:i], store.Bindings[i+1:]...)
				return
			}
		}
	})
}

// GetTaskBySession はセッションIDからタスクIDを返す
func GetTaskBySession(sessionID string) (int, bool) {
	store, err := LoadSessionStore()
	if err != nil {
		return 0, false
	}
	for _, b := range store.Bindings {
		if b.SessionID == sessionID {
			return b.TaskID, true
		}
	}
	return 0, false
}

// GetSessionByTask はタスクIDからセッションIDを返す
func GetSessionByTask(taskID int) (string, bool) {
	store, err := LoadSessionStore()
	if err != nil {
		return "", false
	}
	for _, b := range store.Bindings {
		if b.TaskID == taskID {
			return b.SessionID, true
		}
	}
	return "", false
}
