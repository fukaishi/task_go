package main

import (
	"os"
	"sync"
	"testing"
	"time"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TASK_GO_HOME", dir)
	return dir
}

func TestLoadStoreEmpty(t *testing.T) {
	setupTestDir(t)

	store, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore error: %v", err)
	}
	if store.NextID != 1 {
		t.Errorf("NextID = %d, want 1", store.NextID)
	}
	if len(store.Tasks) != 0 {
		t.Errorf("Tasks length = %d, want 0", len(store.Tasks))
	}
}

func TestSaveAndLoadStore(t *testing.T) {
	setupTestDir(t)

	now := time.Now().Truncate(time.Millisecond)
	store := TaskStore{
		NextID: 3,
		Tasks: []Task{
			{ID: 1, Name: "タスクA", Status: StatusWorking, Priority: PriorityHigh, CreatedAt: now, UpdatedAt: now},
			{ID: 2, Name: "タスクB", Status: StatusNotStarted, Priority: PriorityMedium, CreatedAt: now, UpdatedAt: now},
		},
	}

	err := SaveStore(store)
	if err != nil {
		t.Fatalf("SaveStore error: %v", err)
	}

	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore error: %v", err)
	}
	if loaded.NextID != 3 {
		t.Errorf("NextID = %d, want 3", loaded.NextID)
	}
	if len(loaded.Tasks) != 2 {
		t.Fatalf("Tasks length = %d, want 2", len(loaded.Tasks))
	}
	if loaded.Tasks[0].Name != "タスクA" {
		t.Errorf("Tasks[0].Name = %q, want %q", loaded.Tasks[0].Name, "タスクA")
	}
	if loaded.Tasks[1].Status != StatusNotStarted {
		t.Errorf("Tasks[1].Status = %q, want %q", loaded.Tasks[1].Status, StatusNotStarted)
	}
}

func TestSaveWithLock(t *testing.T) {
	setupTestDir(t)

	// 初期タスクを保存
	err := SaveWithLock(func(store *TaskStore) {
		store.Tasks = append(store.Tasks, Task{
			ID:     store.NextID,
			Name:   "タスク1",
			Status: StatusNotStarted,
		})
		store.NextID++
	})
	if err != nil {
		t.Fatalf("SaveWithLock error: %v", err)
	}

	// 2つ目のタスクを追加
	err = SaveWithLock(func(store *TaskStore) {
		store.Tasks = append(store.Tasks, Task{
			ID:     store.NextID,
			Name:   "タスク2",
			Status: StatusWorking,
		})
		store.NextID++
	})
	if err != nil {
		t.Fatalf("SaveWithLock error: %v", err)
	}

	// 確認
	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore error: %v", err)
	}
	if len(loaded.Tasks) != 2 {
		t.Fatalf("Tasks length = %d, want 2", len(loaded.Tasks))
	}
	if loaded.NextID != 3 {
		t.Errorf("NextID = %d, want 3", loaded.NextID)
	}
}

func TestSaveWithLockConcurrent(t *testing.T) {
	setupTestDir(t)

	// 同時に10個のタスクを追加
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			err := SaveWithLock(func(store *TaskStore) {
				store.Tasks = append(store.Tasks, Task{
					ID:     store.NextID,
					Name:   "並行タスク",
					Status: StatusNotStarted,
				})
				store.NextID++
			})
			if err != nil {
				t.Errorf("SaveWithLock concurrent error: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// 全タスクが正しく保存されていることを確認
	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore error: %v", err)
	}
	if len(loaded.Tasks) != 10 {
		t.Errorf("Tasks length = %d, want 10", len(loaded.Tasks))
	}
	if loaded.NextID != 11 {
		t.Errorf("NextID = %d, want 11", loaded.NextID)
	}

	// IDの重複がないか確認
	ids := make(map[int]bool)
	for _, task := range loaded.Tasks {
		if ids[task.ID] {
			t.Errorf("Duplicate task ID: %d", task.ID)
		}
		ids[task.ID] = true
	}
}

func TestAtomicWrite(t *testing.T) {
	setupTestDir(t)

	// 保存
	store := TaskStore{NextID: 2, Tasks: []Task{{ID: 1, Name: "テスト"}}}
	err := SaveStore(store)
	if err != nil {
		t.Fatalf("SaveStore error: %v", err)
	}

	// tmpファイルが残っていないことを確認
	tmpPath := storePath() + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("Temporary file should not exist after save: %s", tmpPath)
	}
}

func TestFileModTime(t *testing.T) {
	setupTestDir(t)

	// ファイルがない場合はゼロ値
	mt := FileModTime()
	if !mt.IsZero() {
		t.Errorf("FileModTime should be zero when file doesn't exist")
	}

	// 保存後はゼロ値ではない
	store := TaskStore{NextID: 1}
	SaveStore(store)
	mt = FileModTime()
	if mt.IsZero() {
		t.Errorf("FileModTime should not be zero after save")
	}
}
