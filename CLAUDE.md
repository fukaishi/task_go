# task_go プロジェクト

## ビルド・インストール

- `make install` でビルドと `~/bin/` へのインストールを行う
- **macOS では `cp` でGoバイナリをコピーすると adhoc 署名が無効化され、SIGKILL で即座に殺される。** `make install` には `codesign --force --sign -` による再署名が含まれているので、手動で `cp` せず必ず `make install` を使うこと

## データ保存先

- `~/.task_go/tasks.json` — タスクデータ
- `~/.task_go/sessions.json` — Claude Code セッション紐付け
- `~/.task_go/notifications.json` — TUI通知
- 環境変数 `TASK_GO_HOME` でデータディレクトリを変更可能（テスト用）

## MCP サーバー

- `task_go mcp` で stdio トランスポートの MCP サーバーを起動
- MCP サーバーモードでは stdout は JSON-RPC 専用。ログは必ず stderr に出力すること（`log.SetOutput(os.Stderr)`）
