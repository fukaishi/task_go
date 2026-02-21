# task_go

ターミナルで動くタスク管理ツール。[Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) で構築したTUIアプリケーション。

## 機能

- タスクの追加・編集（エディタ連携）
- ステータス管理（未着手 / 作業中 / 確認待ち / 完了）
- 優先度管理（高 / 中 / 低）
- 優先度・IDによる自動ソート
- ファイル変更の自動検知・リロード
- 排他ロックによる安全なデータ保存

## インストール

```bash
make build      # ビルド
make install    # ~/bin にインストール
```

## 使い方

```bash
./task_go
```

### キーバインド

| キー | 操作 |
|------|------|
| `a` | タスク追加 |
| `e` | タスク編集 |
| `t` | ステータス → 作業中 |
| `w` | ステータス → 確認待ち |
| `u` | ステータス → 未着手 |
| `f` | ステータス → 完了 |
| `h` | 優先度 → 高 |
| `m` | 優先度 → 中 |
| `l` | 優先度 → 低 |
| `j` / `↓` | カーソル下移動 |
| `k` / `↑` | カーソル上移動 |
| `Space` | タスク詳細の表示/非表示 |
| `q` / `Ctrl+C` | 終了 |

## データ保存先

`~/.task_go/tasks.json`

環境変数 `TASK_GO_HOME` でディレクトリを変更可能。

## 開発

```bash
make test       # テスト実行
make clean      # ビルド成果物の削除
```
