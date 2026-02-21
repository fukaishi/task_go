package main

import (
	"sync"
	"testing"
)

func TestAddNotification(t *testing.T) {
	setupTestDir(t)

	err := AddNotification(1, NotifyTaskStarted, "タスク1が開始されました")
	if err != nil {
		t.Fatalf("AddNotification error: %v", err)
	}

	store, err := LoadNotificationStore()
	if err != nil {
		t.Fatalf("LoadNotificationStore error: %v", err)
	}
	if len(store.Notifications) != 1 {
		t.Fatalf("Notifications length = %d, want 1", len(store.Notifications))
	}
	n := store.Notifications[0]
	if n.ID != 1 {
		t.Errorf("ID = %d, want 1", n.ID)
	}
	if n.TaskID != 1 {
		t.Errorf("TaskID = %d, want 1", n.TaskID)
	}
	if n.Type != NotifyTaskStarted {
		t.Errorf("Type = %q, want %q", n.Type, NotifyTaskStarted)
	}
	if n.Read {
		t.Error("Read should be false")
	}
}

func TestUnreadNotifications(t *testing.T) {
	setupTestDir(t)

	AddNotification(1, NotifyTaskStarted, "通知1")
	AddNotification(2, NotifyTaskCompleted, "通知2")
	AddNotification(3, NotifyActionNeeded, "通知3")

	// 1つ既読にする
	SaveNotificationWithLock(func(store *NotificationStore) {
		store.Notifications[0].Read = true
	})

	unread, err := UnreadNotifications()
	if err != nil {
		t.Fatalf("UnreadNotifications error: %v", err)
	}
	if len(unread) != 2 {
		t.Fatalf("Unread count = %d, want 2", len(unread))
	}
	if unread[0].TaskID != 2 {
		t.Errorf("Unread[0].TaskID = %d, want 2", unread[0].TaskID)
	}
	if unread[1].TaskID != 3 {
		t.Errorf("Unread[1].TaskID = %d, want 3", unread[1].TaskID)
	}
}

func TestMarkAllRead(t *testing.T) {
	setupTestDir(t)

	AddNotification(1, NotifyTaskStarted, "通知1")
	AddNotification(2, NotifyTaskCompleted, "通知2")

	err := MarkAllRead()
	if err != nil {
		t.Fatalf("MarkAllRead error: %v", err)
	}

	unread, err := UnreadNotifications()
	if err != nil {
		t.Fatalf("UnreadNotifications error: %v", err)
	}
	if len(unread) != 0 {
		t.Errorf("Unread count = %d, want 0", len(unread))
	}
}

func TestNotificationAutoIncrement(t *testing.T) {
	setupTestDir(t)

	AddNotification(1, NotifyTaskStarted, "通知A")
	AddNotification(2, NotifyTaskCompleted, "通知B")

	store, err := LoadNotificationStore()
	if err != nil {
		t.Fatalf("LoadNotificationStore error: %v", err)
	}
	if store.NextID != 3 {
		t.Errorf("NextID = %d, want 3", store.NextID)
	}
	if store.Notifications[0].ID != 1 {
		t.Errorf("Notifications[0].ID = %d, want 1", store.Notifications[0].ID)
	}
	if store.Notifications[1].ID != 2 {
		t.Errorf("Notifications[1].ID = %d, want 2", store.Notifications[1].ID)
	}
}

func TestNotificationFileModTime(t *testing.T) {
	setupTestDir(t)

	// ファイルがない場合はゼロ値
	mt := NotificationFileModTime()
	if !mt.IsZero() {
		t.Error("NotificationFileModTime should be zero when file doesn't exist")
	}

	// 追加後はゼロ値ではない
	AddNotification(1, NotifyTaskStarted, "テスト")
	mt = NotificationFileModTime()
	if mt.IsZero() {
		t.Error("NotificationFileModTime should not be zero after add")
	}
}

func TestUnreadNotificationsEmpty(t *testing.T) {
	setupTestDir(t)

	unread, err := UnreadNotifications()
	if err != nil {
		t.Fatalf("UnreadNotifications error: %v", err)
	}
	if unread != nil {
		t.Errorf("Unread should be nil when no notifications, got %v", unread)
	}
}

func TestNotificationConcurrentWrite(t *testing.T) {
	setupTestDir(t)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			err := AddNotification(n, NotifyTaskStarted, "並行通知")
			if err != nil {
				t.Errorf("AddNotification concurrent error: %v", err)
			}
		}(i)
	}
	wg.Wait()

	store, err := LoadNotificationStore()
	if err != nil {
		t.Fatalf("LoadNotificationStore error: %v", err)
	}
	if len(store.Notifications) != 10 {
		t.Errorf("Notifications count = %d, want 10", len(store.Notifications))
	}

	// IDの重複がないか確認
	ids := make(map[int]bool)
	for _, n := range store.Notifications {
		if ids[n.ID] {
			t.Errorf("Duplicate notification ID: %d", n.ID)
		}
		ids[n.ID] = true
	}
}
