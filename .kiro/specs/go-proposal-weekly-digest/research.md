# Research & Design Decisions

---
**Purpose**: Go proposal weekly digestの技術設計に向けたディスカバリー調査結果を記録する。

**Usage**:
- 技術スタック選定の根拠を文書化
- アーキテクチャパターンの評価結果を保持
- 実装時の参照資料としてリンクを保存
---

## Summary
- **Feature**: `go-proposal-weekly-digest`
- **Discovery Scope**: New Feature（グリーンフィールド）
- **Key Findings**:
  - proposal review meeting minutesは構造化されたフォーマットで投稿されており、正規表現でパース可能
  - Claude Code ActionでGitHub Actions内からAI要約を生成可能、APIキー管理が簡素化
  - Cloudflare Workers Assetsは従来のPagesを置き換え、wrangler v4で統合デプロイが可能

## Research Log

### GitHub Issue #33502のフォーマット解析
- **Context**: proposalステータス変更を自動検出するため、コメントフォーマットの理解が必要
- **Sources Consulted**: [GitHub issue #33502](https://github.com/golang/go/issues/33502)
- **Findings**:
  - 日付ヘッダー: `**YYYY-MM-DD** / @username1, @username2, ...`
  - 各proposalエントリ: issue番号、タイトル（リンク付き）、アクション詳細
  - ステータスマーカー:
    - `discussion ongoing` - 議論継続中
    - `likely accept/decline` + `last call for comments` - 最終決定段階
    - `no final comments; accepted 🎉` - 承認
    - `no final comments; declined` - 却下
    - `put on hold` - 保留
- **Implications**: 正規表現またはパーサーでステータス変更を抽出可能。日付ベースで差分検出できる

### Claude API統合方式の比較
- **Context**: AI要約生成のための統合方式選択
- **Sources Consulted**: [anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go)、[claude-code-action](https://github.com/anthropics/claude-code-action)
- **Findings**:
  - **anthropic-sdk-go**: 公式Go SDK、Go 1.22以上必要、APIキー管理が必要
  - **claude-code-action**: GitHub Actions統合、GitHub App経由で認証、構造化出力対応
  - Claude Code Actionはワークフロー内で直接実行可能、外部依存が減少
- **Implications**: Claude Code Actionを採用。ワークフロー内で要約生成を実行し、APIキー管理を簡素化

### templ静的サイト生成
- **Context**: Goテンプレートエンジンでの静的HTML生成方式
- **Sources Consulted**: [templ.guide](https://templ.guide/static-rendering/generating-static-html-files-with-templ/)
- **Findings**:
  - templコンポーネントは`io.Writer`インターフェースに出力
  - `Render(context.Context, io.Writer)`メソッドでファイル出力可能
  - `templ generate`でGoコードを生成後、ビルド・実行で静的HTML生成
- **Implications**: ビルドパイプラインは`templ generate` → `go build` → `./generator`の流れ

### UnoCSS統合
- **Context**: templテンプレートへのCSSスタイル適用
- **Sources Consulted**: [UnoCSS公式](https://unocss.dev/)、Bridgetown統合例
- **Findings**:
  - UnoCSS CLIで.templファイルからユーティリティクラスを抽出可能
  - `unocss --config uno.config.ts --out-file dist/styles.css`形式で生成
  - templ出力HTML + UnoCSS抽出CSSを組み合わせ
- **Implications**: ビルドパイプラインにUnoCSS CLIを統合。extractorパターンで.templファイルを対象に

### Lit Web Components + esbuild
- **Context**: 動的UIコンポーネントのバンドル方式
- **Sources Consulted**: [Lit公式](https://lit.dev/)、[esbuild-plugin-lit](https://github.com/nicr92/esbuild-plugin-lit)
- **Findings**:
  - esbuildはRollupより高速、TypeScript組み込み対応
  - `esbuild-plugin-lit-css`でCSS importをタグ付きテンプレートリテラルに変換
  - `splitting: true`で共通チャンクを自動生成
  - Declarative Shadow DOMでSSR/静的レンダリングと組み合わせ可能
- **Implications**: esbuildでLitコンポーネントをバンドル。静的HTMLにscript type="module"で埋め込み

### Cloudflare Workers Assets
- **Context**: 静的サイトのデプロイ先選定
- **Sources Consulted**: [Cloudflare Workers docs](https://developers.cloudflare.com/workers/static-assets/)
- **Findings**:
  - Workers Sitesは非推奨、Workers Static Assetsを使用
  - wrangler.jsonc設定: `assets.directory`、`not_found_handling`、`html_handling`
  - `wrangler deploy`で静的ファイルとWorkerコードを一括デプロイ
  - PRプレビューはbranch aliasで対応可能
- **Implications**: wrangler v4 + wrangler.jsoncで設定。GitHub Actionsから`wrangler deploy`実行

### Claude Code GitHub Action
- **Context**: AI自動化のGitHub Actions統合
- **Sources Consulted**: [claude-code-action](https://github.com/anthropics/claude-code-action)、[Marketplace](https://github.com/marketplace/actions/claude-code-action-official)
- **Findings**:
  - `@claude`メンションでPR/issue内からAI呼び出し
  - スケジュール実行で定期タスク自動化可能
  - Anthropic API、AWS Bedrock、Google Vertex AIに対応
  - 構造化出力でJSONレスポンスをAction outputとして利用可能
- **Implications**: 週次更新ワークフローでclaude-code-actionを使用。proposalごとの要約生成を自動化

### RSSフィード生成ライブラリ
- **Context**: RSS 2.0フィード生成のGo実装
- **Sources Consulted**: [gorilla/feeds](https://github.com/gorilla/feeds)、[gopherlibs/feedhub](https://github.com/gopherlibs/feedhub)
- **Findings**:
  - gorilla/feedsは2022年にアーカイブ済み
  - gopherlibs/feedhubがforkとして継続メンテナンス
  - RSS 2.0、Atom、JSON Feed対応
  - シンプルなAPI: `Feed`構造体にアイテム追加 → `ToRss()`で出力
- **Implications**: gopherlibs/feedhubを採用。メンテナンス継続性を重視

## Architecture Pattern Evaluation

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|---------------------|-------|
| Monolithic Generator | 単一のGoバイナリで全処理 | シンプル、デプロイ容易 | スケーラビリティ制限 | 本プロジェクトの規模に適合 |
| Pipeline Architecture | 各フェーズを独立コンポーネント化 | 拡張性、テスト容易 | 複雑性増加 | 将来の拡張に備えたい場合 |
| Event-Driven | GitHub Webhookトリガー | リアルタイム性 | 設定複雑、コスト増 | 週次更新には過剰 |

**選択**: Pipeline Architecture（軽量版）
- パース、コンテンツ生成、サイト生成を論理的に分離
- 単一バイナリ内でパイプライン実行
- テスト容易性と拡張性のバランス

## Design Decisions

### Decision: Claude Code ActionによるAI要約生成
- **Context**: proposal変更の日本語要約を生成する統合方式の選択
- **Alternatives Considered**:
  1. anthropic-sdk-go - Go SDKで直接API呼び出し、Goコード内で完結
  2. Claude Code Action - GitHub Actionsワークフロー内で実行、構造化出力対応
  3. 外部サービス（Lambda等） - 別途インフラ管理が必要
- **Selected Approach**: Claude Code Action
- **Rationale**:
  - APIキー管理が不要（GitHub App経由で認証）
  - ワークフロー内で完結し、Goバイナリの責務を限定できる
  - 構造化出力でMarkdown形式の要約を安定して取得可能
  - GitHub Actions実行ログで処理内容を確認しやすい
- **Trade-offs**: Goコード単体でのテストが困難になる。ワークフロー全体のE2Eテストが必要
- **Follow-up**: ワークフローステップ間のデータ受け渡し（changes.json → summaries/）を検証

### Decision: コンテンツ形式としてmdxではなくJSONを採用
- **Context**: 要件ではmdxを指定しているが、Go環境での扱いやすさを考慮
- **Alternatives Considered**:
  1. mdx - フロントエンド統合に優れるが、Goでのパースが複雑
  2. YAML frontmatter + Markdown - 構造化データとコンテンツを分離可能
  3. JSON - Goでの型安全な処理が容易
- **Selected Approach**: YAML frontmatter + Markdown
- **Rationale**: Goでのパースが容易（go-yaml）、かつMarkdownコンテンツとメタデータを分離できる。mdxのJSX部分は静的生成には不要
- **Trade-offs**: JSXコンポーネント埋め込みは不可。代わりにLit Web Componentをスクリプト埋め込み
- **Follow-up**: templテンプレート内でMarkdownレンダリングを検証

### Decision: 差分検出のためのステート管理
- **Context**: 前回チェック以降の変更を検出する仕組みが必要
- **Alternatives Considered**:
  1. ファイルベース - 最終チェック日時をファイル保存
  2. Git履歴 - コミットハッシュで状態管理
  3. GitHub API - issue更新日時をAPI取得
- **Selected Approach**: Git履歴 + ファイルベースのハイブリッド
- **Rationale**: 各週のコンテンツはGitコミットとして保存。最終処理日時はstate.jsonで管理
- **Trade-offs**: Gitリポジトリへの書き込み権限が必要
- **Follow-up**: GitHub Actionsでのgit push権限設定を確認

### Decision: フロントエンドビルドツールチェーン
- **Context**: TypeScript/Lit/UnoCSS統合
- **Alternatives Considered**:
  1. esbuild単体 - 高速だがプラグインエコシステムが限定的
  2. Vite - 開発体験優れるがSSG用途には過剰
  3. tsgo + esbuild - 最新のTypeScriptツールチェーン
- **Selected Approach**: esbuild + tsgo
- **Rationale**: 高速ビルド、TypeScript標準ツールチェーン活用、プラグインでLit対応
- **Trade-offs**: Viteほどの開発サーバー機能はない（静的生成用途では問題なし）
- **Follow-up**: tsgoのtempl統合を検証

## Risks & Mitigations
- **GitHub APIレート制限** - issue #33502は更新頻度が低いため問題なし。念のためETAGキャッシュを実装
- **Claude APIコスト** - 要約生成は週1回、proposal数件程度。コスト予測可能
- **Cloudflareデプロイ失敗** - wrangler出力をログ保存、Slack/Discord通知で監視
- **proposal minutesフォーマット変更** - パーサーにフォールバック処理を実装

## References
- [GitHub issue #33502](https://github.com/golang/go/issues/33502) - proposal review meeting minutes
- [anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go) - 公式Go SDK
- [templ.guide](https://templ.guide/) - templドキュメント
- [Cloudflare Workers Static Assets](https://developers.cloudflare.com/workers/static-assets/) - デプロイガイド
- [claude-code-action](https://github.com/anthropics/claude-code-action) - GitHub Actions統合
- [gopherlibs/feedhub](https://github.com/gopherlibs/feedhub) - RSSフィード生成
- [Lit](https://lit.dev/) - Web Componentsライブラリ
- [esbuild](https://esbuild.github.io/) - JavaScriptバンドラー
