# Go proposal review meeting update

Go 言語の [proposal: review meeting minutes](https://github.com/golang/go/issues/33502) を基に、先週からの更新をまとめる、静的 Web サイトです。

## 要件

- 毎週日本時間木曜12:00に指定 issue のコメントをチェックし、反映する
- 変更があれば静的サイトを再生成し、デプロイする
- Go の proposal では、以下のステータスがあります
  - Discussions (not yet proposals)
  - Accepted
  - Declined
  - Likely Accept
  - Active
  - Hold
- 変更は コメントのメッセージを解析したうえで、内容の差分をとってください
- そのうえで、Claude Code Action により proposal ごとのステータスの変化とその理由を調査したうえで、要約してください
- サイトにはプロポーザルごとのステータスの変化とその理由が目立つようにしてください
- プロポーザルごとの関連 issue などのリンクも掲載してください
- 変更があれば、GitHub Actions で自動的にデプロイしてください
- RSS フィードも提供してください

## 技術スタック

- テンプレートエンジン: templ
- Web Component を用いてクライアント上で動作する動的 UI 部分を実装
    - コンポーネントは Lit で作成
    - esbuild でバンドル
    - TypeScript の処理には tsgo を使用
    - 最小限の使用にとどめ、極力静的サイトとして動作させる
- CSS は unocss を使用する
- GitHub Actions の Cron により定期的にコンテンツ更新
    - 直接 main ブランチに push する
- Claude Code Action を用いて、issue コメントの解析と要約を実装
    - ただし、issue コメントの解析は極力プログラム側で行い、Claude には要約のみに注力させる
    - GitHub MCP 経由で GitHub API を使用し、issue コメントを操作できるようにする
        - PAT は GitHub Actions に設定されたものを使用
- デプロイ先: Cloudflare Workers Assets
    - PR ごとにプレビューを生成する
    - main ブランチへのマージ時に本番デプロイ
- UI とコンテンツは分離する
    - 毎週の更新内容は mdx で階層的に管理
      - 週ごとのディレクトリが切られ、その中で管理
      - プロポーザルごとに mdx ファイルを分割
    - 静的サイト生成時に mdx をテンプレートエンジンで処理し、HTML に変換
- PR 時に html の差分を確認できるようにする
    - GitHub Actions で差分生成を行い、PR コメントに貼り付ける
