package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// NotificationType は通知の種類を表す
type NotificationType string

const (
	NotifyActionNeeded  NotificationType = "action_needed"
	NotifyTaskStarted   NotificationType = "task_started"
	NotifyTaskCompleted NotificationType = "task_completed"
)

// Notification は通知を表す
type Notification struct {
	ID        int              `json:"id"`
	TaskID    int              `json:"task_id"`
	Type      NotificationType `json:"type"`
	Message   string           `json:"message"`
	Timestamp time.Time        `json:"timestamp"`
	Read      bool             `json:"read"`
}

// NotificationStore は通知一覧と自動採番用のNextIDを保持する
type NotificationStore struct {
	NextID        int            `json:"next_id"`
	Notifications []Notification `json:"notifications"`
}

func notificationPath() string {
	return filepath.Join(storeDir(), "notifications.json")
}

func notificationLockPath() string {
	return filepath.Join(storeDir(), "notifications.lock")
}

// LoadNotificationStore はJSONファイルからNotificationStoreを読み込む
func LoadNotificationStore() (NotificationStore, error) {
	var store NotificationStore
	store.NextID = 1

	data, err := os.ReadFile(notificationPath())
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

// SaveNotificationStore はNotificationStoreをJSONファイルにアトミックに保存する
func SaveNotificationStore(store NotificationStore) error {
	dir := storeDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := notificationPath() + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, notificationPath())
}

// SaveNotificationWithLock は排他制御付きで通知ストアを変更・保存する
func SaveNotificationWithLock(modify func(*NotificationStore)) error {
	dir := storeDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	lockFile, err := os.OpenFile(notificationLockPath(), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	store, err := LoadNotificationStore()
	if err != nil {
		return err
	}

	modify(&store)

	return SaveNotificationStore(store)
}

// AddNotification は通知を追加する
func AddNotification(taskID int, ntype NotificationType, message string) error {
	return SaveNotificationWithLock(func(store *NotificationStore) {
		store.Notifications = append(store.Notifications, Notification{
			ID:        store.NextID,
			TaskID:    taskID,
			Type:      ntype,
			Message:   message,
			Timestamp: time.Now(),
			Read:      false,
		})
		store.NextID++
	})
}

// UnreadNotifications は未読の通知を返す
func UnreadNotifications() ([]Notification, error) {
	store, err := LoadNotificationStore()
	if err != nil {
		return nil, err
	}

	var unread []Notification
	for _, n := range store.Notifications {
		if !n.Read {
			unread = append(unread, n)
		}
	}
	return unread, nil
}

// MarkAllRead は全ての通知を既読にする
func MarkAllRead() error {
	return SaveNotificationWithLock(func(store *NotificationStore) {
		for i := range store.Notifications {
			store.Notifications[i].Read = true
		}
	})
}

// NotificationFileModTime は通知ファイルの更新日時を返す
func NotificationFileModTime() time.Time {
	info, err := os.Stat(notificationPath())
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
