package main

import (
	"sync"
	"testing"
)

func TestBindAndGetSession(t *testing.T) {
	setupTestDir(t)

	err := BindSession("session-1", 42, StatusWorking)
	if err != nil {
		t.Fatalf("BindSession error: %v", err)
	}

	taskID, ok := GetTaskBySession("session-1")
	if !ok {
		t.Fatal("GetTaskBySession returned false")
	}
	if taskID != 42 {
		t.Errorf("GetTaskBySession = %d, want 42", taskID)
	}
}

func TestBindSessionOverwrite(t *testing.T) {
	setupTestDir(t)

	BindSession("session-1", 10, StatusWorking)
	BindSession("session-1", 20, StatusWorking)

	taskID, ok := GetTaskBySession("session-1")
	if !ok {
		t.Fatal("GetTaskBySession returned false")
	}
	if taskID != 20 {
		t.Errorf("GetTaskBySession = %d, want 20 (overwritten)", taskID)
	}

	// 紐付けが1件だけであることを確認
	store, _ := LoadSessionStore()
	count := 0
	for _, b := range store.Bindings {
		if b.SessionID == "session-1" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Binding count = %d, want 1", count)
	}
}

func TestUnbindSession(t *testing.T) {
	setupTestDir(t)

	BindSession("session-1", 42, StatusWorking)

	err := UnbindSession("session-1")
	if err != nil {
		t.Fatalf("UnbindSession error: %v", err)
	}

	_, ok := GetTaskBySession("session-1")
	if ok {
		t.Error("GetTaskBySession should return false after unbind")
	}
}

func TestUnbindSessionNotFound(t *testing.T) {
	setupTestDir(t)

	// 存在しないセッションのUnbindはエラーにならない
	err := UnbindSession("nonexistent")
	if err != nil {
		t.Fatalf("UnbindSession error: %v", err)
	}
}

func TestGetSessionByTask(t *testing.T) {
	setupTestDir(t)

	BindSession("session-abc", 7, StatusWorking)

	sessionID, ok := GetSessionByTask(7)
	if !ok {
		t.Fatal("GetSessionByTask returned false")
	}
	if sessionID != "session-abc" {
		t.Errorf("GetSessionByTask = %q, want %q", sessionID, "session-abc")
	}

	// 存在しないタスクID
	_, ok = GetSessionByTask(999)
	if ok {
		t.Error("GetSessionByTask should return false for unknown task")
	}
}

func TestGetTaskBySessionNotFound(t *testing.T) {
	setupTestDir(t)

	_, ok := GetTaskBySession("nonexistent")
	if ok {
		t.Error("GetTaskBySession should return false for unknown session")
	}
}

func TestSessionConcurrentWrite(t *testing.T) {
	setupTestDir(t)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			err := BindSession("session-concurrent", n, StatusWorking)
			if err != nil {
				t.Errorf("BindSession concurrent error: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// 最終的に1件の紐付けがある
	store, err := LoadSessionStore()
	if err != nil {
		t.Fatalf("LoadSessionStore error: %v", err)
	}
	count := 0
	for _, b := range store.Bindings {
		if b.SessionID == "session-concurrent" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Binding count = %d, want 1", count)
	}
}
