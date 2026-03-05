---
issue_number: 76485
title: "cmd/go: support dependency cooldown in Go tooling"
previous_status: 
current_status: active
changed_at: 2026-03-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
related_issues:
  - title: "Proposal Issue #76485"
    url: https://github.com/golang/go/issues/76485
  - title: "Review Minutes (aclements comment)"
    url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
---
## 概要

`go get` や `go mod tidy` などのGoツールチェーンに「依存関係クールダウン」機能を追加するプロポーザルです。これは、公開から一定期間（例: 15日）が経過していない依存関係バージョンを自動的に除外することで、サプライチェーン攻撃のリスクを軽減することを目的としています。

## ステータス変更

**(新規)** → **active**

2026年3月4日、`@aclements`（Goプロポーザルレビューグループ）によってactiveカラムに移動されました。その直前のコメント（2026年3月4日）で、このプロポーザルがレビューミーティングの対象に含まれていないことを指摘するユーザーがいたことがきっかけとなり、正式にweekly proposal review meetingの議題として取り上げられることになりました。

## 技術的背景

### 現状の問題点

オープンソースのサプライチェーン攻撃は年々増加しており、2025年にはGitLabがGoモジュールを対象としたタイポスクワッティング攻撃を検出した事例もあります。ブログ記事「We should all be using dependency cooldowns」の実証的調査によれば、最近の10件のサプライチェーン攻撃のうち8件は攻撃ウィンドウが1週間未満であり、多くは数時間から数日で検出・除去されています。

つまり、攻撃者が悪意あるコードを公開してから発覚されるまでの時間は非常に短い傾向にあり、「新しすぎる」バージョンを自動的に避けるだけで攻撃の大半を回避できる可能性があります。

現在のGoはMVS（Minimum Version Selection）の設計上、深いところの依存関係が最新バージョンになりにくいという特性はあるものの、`go get` や `go mod tidy` を実行した際に攻撃が進行中のバージョンを取得してしまうリスクは残ります。

### 提案された解決策

環境変数 `GOCOOLDOWN` を使って、考慮対象とする依存バージョンの最低経過時間を設定できるようにします。

```none
GOCOOLDOWN=15d go mod tidy
```

この設定により、公開から15日未満のバージョンは `go mod tidy` や `go get` による解決候補から除外されます。

**タイムスタンプの取得方法**については議論が進んでいます。`FiloSottile`（Filippo Valsorda、Goセキュリティチーム）は、sumdb（チェックサムデータベース）にバージョンの「初回観測タイムスタンプ（first-observed timestamp）」を追加し、これをクライアント側で参照する方式を提案しています。これにより、プロキシが嘘のタイムスタンプを返してもsumdbによって検証が担保されるという、信頼モデルに整合した実装が可能になります。なお、`index.golang.org` フィードにはすでに `proxy.golang.org` が初めてキャッシュした時刻（Timestamp）が含まれており、実装の基盤となり得ます。

## これによって何ができるようになるか

### コード例

```bash
# Before: 従来の書き方（クールダウンなし）
# go get や go mod tidy は公開直後のバージョンも取得する
go mod tidy

# After: クールダウンを設定した書き方
# 15日以内に公開されたバージョンはスキップされる
GOCOOLDOWN=15d go mod tidy

# または週単位・月単位での設定も想定される
GOCOOLDOWN=2w go get example.com/somepackage@latest
```

### 想定されるユースケース

- **企業・チームでのCI/CD環境**: 本番デプロイに使うGoツールチェーンにクールダウンを設定し、攻撃が発覚する前に悪意あるバージョンを取り込むリスクを低減
- **セキュリティ重視のプロジェクト**: 金融・医療・インフラ系のシステムで、外部依存関係の更新に一定の待機期間を設けることを標準化
- **個人開発者のリスク軽減**: 外部ツール（DependabotやRenovate）に頼らず、Goツールチェーン単体でサプライチェーン攻撃への基本的な防御を実現

なお、`@imjasonh` によるプロトタイプ実装 `go-cooldown`（軽量プロキシ方式）が既に公開されており、GOPROXYとして設定するだけで利用可能です。この実装はデフォルト7日のクールダウンを適用し、`/list` レスポンスから新しすぎるバージョンを除外します。

## 議論のハイライト

- **「外部ツールに任せるべき」という反論**: `@seankhliao` はDependabotなどの外部ツールで十分であり、Goには依存関係が公開された正確な時刻を把握するレジストリが存在しないため、ツール組み込みは難しいと指摘。ただし、その後 `index.golang.org` のタイムスタンプという解決策が示された。

- **sumdbへの組み込みが鍵**: `FiloSottile` はsumdbにfirst-observed timestampを追加することで、「プロキシを信頼しない」というGoのセキュリティモデルを維持しながらクールダウンを実装できると提案。プロキシが古い日付を偽ってもsumdbで検証できる設計が重要とされた。

- **タイムスタンプの偽装リスク**: `@Jorropo` はgitタグのタイムスタンプは攻撃者にバックデートされる可能性を指摘。これがsumdb/プロキシタイムスタンプ活用の根拠となった。

- **プロキシ方式のプロトタイプ**: 軽量プロキシとしてGOPROXYに挟む方式を `@seankhliao` が提案し、`FiloSottile` も「Geomysとしてプロトタイプを提供するかも」と言及。実際に `@imjasonh` の `go-cooldown` として実装された。

- **`go.mod` 更新との整合性**: すでにチームメンバーが新しいバージョン（クールダウン未満）を `go.mod` に記録していた場合の扱いが未解決の課題として残っている。

## 関連リンク

- [Proposal Issue #76485](https://github.com/golang/go/issues/76485)
- [Review Minutes (aclements comment)](https://github.com/golang/go/issues/33502#issuecomment-4000757564)
