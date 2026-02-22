# task_go

ターミナルで動くタスク管理ツール。[Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) で構築したTUIアプリケーション。

Claude Code と連携し、AIが作業しているタスクの状態をリアルタイムにTUI上で確認できる。

## 機能

- タスクの追加・編集（エディタ連携）
- ステータス管理（未着手 / 作業中 / 確認待ち / 完了）
- 優先度管理（高 / 中 / 低）
- 優先度・IDによる自動ソート
- ファイル変更の自動検知・リロード
- 排他ロックによる安全なデータ保存
- Claude Code 連携（MCP + Hooks）

## インストール

### 前提条件

- Go 1.21 以上
- macOS または Linux
- Claude Code（連携機能を使う場合）

### 1. task_go 本体のビルド・インストール

```bash
git clone <repository-url>
cd task_go
make install    # ビルドして ~/bin にインストール
```

> **注意（macOS）:** `make install` は adhoc 署名の再付与を含んでいる。手動で `cp` するとバイナリが SIGKILL されるため、必ず `make install` を使うこと。

`~/bin` にパスが通っていない場合は `.zshrc` 等に追加する：

```bash
export PATH="$HOME/bin:$PATH"
```

### 2. MCP サーバーの設定

Claude Code の設定ファイル（`~/.claude/claude_desktop_config.json` または プロジェクトの `.mcp.json`）に以下を追加する：

```json
{
  "mcpServers": {
    "task_go": {
      "command": "/Users/<ユーザー名>/bin/task_go",
      "args": ["mcp"]
    }
  }
}
```

これにより Claude Code が MCP 経由で task_go のタスクを読み書きできるようになる。

### 3. Hooks の設定

Claude Code の設定ファイル（`~/.claude/settings.json` または プロジェクトの `.claude/settings.json`）に以下を追加する：

```json
{
  "hooks": {
    "Notification": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/Users/<ユーザー名>/bin/task_go notify"
          }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/Users/<ユーザー名>/bin/task_go notify"
          }
        ]
      }
    ]
  }
}
```

Hooks により以下のイベントが task_go に通知される：

| イベント | 効果 |
|---------|------|
| `Notification (idle_prompt)` | TUI に「入力待ち」と表示 |
| `Notification (permission_prompt)` | TUI に「許可が必要」と表示 |
| `Stop` | セッション紐付けを解除（Claude Code 終了時） |

## 使い方

### TUI モード

```bash
task_go
```

TUI を起動すると、画面上部に「作業中タスク」セクション、その下に「タスク一覧」が表示される。

#### キーバインド

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

### CLI コマンド

```bash
task_go list                          # タスク一覧
task_go list --status working         # ステータスでフィルタ
task_go list --format json            # JSON形式で出力
task_go add "タスク名"                 # タスク追加
task_go add "タスク名" --priority high # 優先度付きで追加
task_go update <id> --status working  # ステータス変更
task_go get <id>                      # タスク詳細
task_go get <id> --format json        # JSON形式で詳細
task_go version                       # バージョン表示
task_go help                          # ヘルプ
```

## Claude Code 連携

MCP と Hooks を設定すると、Claude Code と task_go が連携して以下のワークフローが実現する。

### 何ができるか

1. **タスクの参照・作成・更新** — Claude Code が MCP 経由でタスクを読み書きできる。「タスク一覧を見せて」「新しいタスクを作って」といった指示に対応。

2. **作業状態のリアルタイム表示** — Claude Code がタスクに着手すると、TUI の「作業中タスク」セクションに「>> Claude稼働中」と表示される。別ターミナルで TUI を開いておけば、Claude Code の稼働状況を常に把握できる。

3. **Claude Code の状態通知** — Hooks により Claude Code の状態変化が TUI に反映される：
   - 入力待ち状態 → 「>> Claude稼働中（入力待ち）」
   - 許可が必要な状態 → 「>> Claude稼働中（許可が必要）」
   - セッション終了 → インジケーター消去

4. **確認待ち通知** — Claude Code が作業を終えた時、TUI に「!! 確認してください」と表示して注意を促す。

### 運用フロー

```
1. TUI でタスクを作成（または Claude Code に依頼して作成）
2. Claude Code に「タスク N に着手して」と指示
3. Claude Code が start_task を呼び、TUI に「>> Claude稼働中」と表示
4. 作業中、Hooks が Claude Code の状態をリアルタイムに TUI へ反映
5. Claude Code が作業を終えたら notify_action_needed を呼び、
   TUI に「!! 確認してください」と表示
6. ユーザーが TUI 上でタスクを確認し、完了操作（f キー）を行う
```

> **設計方針:** タスクの完了はユーザーが TUI から行う。Claude Code がタスクを自動完了することはない。

### MCP ツール一覧

| ツール名 | 説明 |
|---------|------|
| `list_tasks` | タスク一覧を取得（ステータスフィルタ可） |
| `get_task` | 指定IDのタスク詳細を取得 |
| `create_task` | 新しいタスクを作成 |
| `update_task` | 既存タスクを部分更新 |
| `start_task` | タスクを作業中にしセッションに紐付け |
| `current_task` | セッションに紐付いた現在のタスクを取得 |
| `notify_action_needed` | セッションを「確認待ち」に変更 |

## データ保存先

| ファイル | 内容 |
|---------|------|
| `~/.task_go/tasks.json` | タスクデータ |
| `~/.task_go/sessions.json` | Claude Code セッション紐付け |

環境変数 `TASK_GO_HOME` でデータディレクトリを変更可能。

## 開発

```bash
make build      # ビルド
make test       # テスト実行
make clean      # ビルド成果物の削除
```
