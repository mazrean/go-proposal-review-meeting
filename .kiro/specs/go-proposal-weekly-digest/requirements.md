# Requirements Document

## Introduction

本プロジェクトは、Go言語の[proposal: review meeting minutes](https://github.com/golang/go/issues/33502)を監視し、先週からのproposalステータス変更を自動的に検出・要約して静的Webサイトとして公開するシステムである。毎週の更新を追跡し、日本語で分かりやすく提供することで、Goコミュニティのproposal動向を把握しやすくする。

## Requirements

### Requirement 1: Issueコメント解析

**Objective:** システム運用者として、GitHub issue #33502のコメントを自動解析し、proposalのステータス変更を検出できるようにする

#### Acceptance Criteria
1. When 定期実行がトリガーされた時, the Issue Parser shall GitHub issue #33502の最新コメントを取得する
2. When コメントを取得した時, the Issue Parser shall 前回チェック以降の新規コメントを識別する
3. When 新規コメントを解析する時, the Issue Parser shall proposalのステータス（Discussions、Accepted、Declined、Likely Accept、Active、Hold）を抽出する
4. When ステータス変更を検出した時, the Issue Parser shall 変更前後のステータスと対象proposalを記録する
5. If コメント取得に失敗した場合, then the Issue Parser shall エラーをログに記録し、次回実行まで待機する

### Requirement 2: コンテンツ管理

**Objective:** コンテンツ管理者として、週次更新を構造化されたフォーマットで保存し、履歴の追跡と編集を容易にする

#### Acceptance Criteria
1. When ステータス変更が検出された時, the Content Manager shall 週ごとのディレクトリ（YYYY-WXX形式）を作成する
2. When 変更を保存する時, the Content Manager shall proposalごとに個別のmdxファイルを生成する
3. The Content Manager shall mdxファイルにproposal番号、タイトル、ステータス変更、関連リンクを含める
4. While 同一週内で複数回更新がある場合, the Content Manager shall 既存ファイルを上書きせず差分をマージする
5. The Content Manager shall 過去の週次データを変更せず保持する

### Requirement 3: AI要約生成

**Objective:** サイト閲覧者として、proposalのステータス変更理由を日本語で理解し、技術的な背景を素早く把握できるようにする

#### Acceptance Criteria
1. When ステータス変更が検出された時, the Summary Generator shall Claude APIを呼び出して要約を生成する
2. When 要約を生成する時, the Summary Generator shall ステータス変更の理由と背景を日本語で説明する
3. The Summary Generator shall 関連するGitHub issueや議論へのリンクを要約に含める
4. If Claude API呼び出しに失敗した場合, then the Summary Generator shall フォールバックとして基本情報のみを出力する
5. The Summary Generator shall 要約の長さを200〜500文字程度に制限する

### Requirement 4: 静的サイト生成

**Objective:** サイト閲覧者として、見やすく整理されたWebページで更新を確認し、週次の変更を効率的に把握できるようにする

#### Acceptance Criteria
1. When mdxコンテンツが更新された時, the Site Generator shall templテンプレートを使用してHTMLを生成する
2. The Site Generator shall proposalごとのステータス変化を視覚的に目立たせる
3. The Site Generator shall 週ごとのインデックスページと個別proposalページを生成する
4. When サイトを生成する時, the Site Generator shall unocssでスタイルを適用する
5. Where 動的UIが必要な箇所では, the Site Generator shall Lit Web Componentを埋め込む
6. The Site Generator shall 生成されたHTML、CSS、JSをesbuildでバンドルする

### Requirement 5: RSSフィード

**Objective:** フィード購読者として、RSSで更新通知を受け取り、サイトを直接訪問せずに新着を確認できるようにする

#### Acceptance Criteria
1. When サイトが生成された時, the Feed Generator shall RSS 2.0形式のフィードを生成する
2. The Feed Generator shall 各週の更新をフィードアイテムとして含める
3. The Feed Generator shall フィードにproposalのタイトル、ステータス変更、要約を含める
4. The Feed Generator shall フィードのautodiscoveryタグをHTMLに埋め込む
5. The Feed Generator shall 最新20件の更新をフィードに保持する

### Requirement 6: 定期実行とデプロイ

**Objective:** システム運用者として、自動的に更新とデプロイを行い、手動介入なしで最新情報を提供できるようにする

#### Acceptance Criteria
1. When 毎週木曜日12:00 JSTになった時, the GitHub Actions shall コンテンツ更新ワークフローを実行する
2. When 変更が検出された時, the GitHub Actions shall 静的サイトを再生成する
3. When 再生成が完了した時, the GitHub Actions shall mainブランチに直接コミットする
4. When mainブランチが更新された時, the Deployment Pipeline shall Cloudflare Workers Assetsに本番デプロイする
5. When PRが作成された時, the Deployment Pipeline shall プレビュー環境にデプロイする
6. If デプロイに失敗した場合, then the GitHub Actions shall エラー通知を送信する

### Requirement 7: PR差分確認

**Objective:** レビュアーとして、PRでHTMLの差分を確認し、変更内容を視覚的に検証できるようにする

#### Acceptance Criteria
1. When PRが作成または更新された時, the Diff Generator shall 生成されたHTMLの差分を計算する
2. When 差分が計算された時, the Diff Generator shall PRコメントに差分サマリーを投稿する
3. The Diff Generator shall 追加、削除、変更されたページの一覧を表示する
4. Where 差分が大きい場合, the Diff Generator shall 主要な変更点のみを要約して表示する
