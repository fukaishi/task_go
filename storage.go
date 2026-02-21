package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// storeDir はデータ保存先ディレクトリを返す
// 環境変数 TASK_GO_HOME でオーバーライド可能(テスト用)
func storeDir() string {
	if dir := os.Getenv("TASK_GO_HOME"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".task_go")
}

func storePath() string {
	return filepath.Join(storeDir(), "tasks.json")
}

func lockPath() string {
	return filepath.Join(storeDir(), "tasks.lock")
}

// LoadStore はJSONファイルからTaskStoreを読み込む
// ファイルが存在しない場合は空のストアを返す
func LoadStore() (TaskStore, error) {
	var store TaskStore
	store.NextID = 1

	data, err := os.ReadFile(storePath())
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

// SaveStore はTaskStoreをJSONファイルにアトミックに保存する
func SaveStore(store TaskStore) error {
	dir := storeDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	// アトミック書き込み: tmpファイルに書き込んでからリネーム
	tmpPath := storePath() + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, storePath())
}

// SaveWithLock は排他制御付きでストアを変更・保存する
// 1. ロックファイルで排他ロック取得
// 2. 最新のJSONを再読み込み
// 3. modify関数でストアを変更
// 4. アトミック書き込み
// 5. ロック解放
func SaveWithLock(modify func(*TaskStore)) error {
	dir := storeDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// ロックファイルを開く
	lockFile, err := os.OpenFile(lockPath(), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer lockFile.Close()

	// 排他ロック取得
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	// 最新のストアを再読み込み
	store, err := LoadStore()
	if err != nil {
		return err
	}

	// ストアを変更
	modify(&store)

	// アトミック書き込み
	return SaveStore(store)
}

// FileModTime はストアファイルの更新日時を返す
// ファイルが存在しない場合はゼロ値を返す
func FileModTime() time.Time {
	info, err := os.Stat(storePath())
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
