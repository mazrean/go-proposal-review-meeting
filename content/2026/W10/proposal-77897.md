---
issue_number: 77897
title: "cmd/go: only set vcs.modified=true if changes are relevant to build"
previous_status: 
current_status: active
changed_at: 2026-03-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77897
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
  - title: "関連Issue #50603: go buildでバージョンスタンプを行う提案"
    url: https://github.com/golang/go/issues/50603
  - title: "関連Issue #50603 コメント: vcs.modified変更の議論"
    url: https://github.com/golang/go/issues/50603#issuecomment-2315888855
---
## 概要

`go build` や `go install` 実行時にバイナリへ埋め込まれる `vcs.modified` フラグを、ビルドに実際に関連するファイルへの変更がある場合のみ `true` に設定するよう改善する提案です。現在はビルドに無関係なファイルの変更でも `+dirty` が付いてしまう問題を解決します。

## ステータス変更
**(新規)** → **active**

Go提案レビューグループのaclements氏により、毎週行われる提案レビューミーティングのアクティブキューに追加されました。提案内容が明確であり、長年にわたり複数の開発者から同様の問題が報告されてきたことが、迅速にactiveステータスへ移行した背景にあります。

## 技術的背景

### 現状の問題点

`cmd/go` はバイナリにVCS（バージョン管理システム）情報を埋め込む際、変更されたファイルのリスト (`git status` 相当) を取得し、リストが空でない場合に `vcs.modified=true` を設定します。この判定はビルドに関係するファイルかどうかを区別しません。

代表的な問題シナリオを以下に示します。

```
# 完全にクリーンなリポジトリを想定
go build   # ./foo バイナリが生成される → リポジトリが「変更あり」状態に
go install # +dirty サフィックス付きで $GOBIN/foo にインストール、./foo は削除される
           # → リポジトリは「変更なし」状態に戻る
go install # 今度は +dirty なしでインストールされる
```

`go build` が生成したバイナリ自体が「未追跡ファイル」として検出され、直後の `go install` で `+dirty` が付くという矛盾した動作が発生します。また、CPUプロファイルファイル (`x.prof`) など開発作業で生まれる一時ファイルも同様に `+dirty` を引き起こします。

この問題は Rob Pike の `ivy` プロジェクト (robpike/ivy#267) でも実際に発生し、開発者が `go clean` を毎回実行するという不便な回避策を強いられていました。

### 提案された解決策

変更済みファイルのリストを取得した後、そのファイルがビルドで実際に使用されるファイルかどうかを照合する処理を追加します。提案者の @rsc は以下の定義を示しています。

**modified = ビルドで使用されたいずれかのファイルがVCSのコピーと異なる場合、またはVCSに存在しない場合**

この定義はVCS無視設定 (`.gitignore`) の有無ではなく、VCSリポジトリへの登録有無を基準とします。`.gitignore` に記載されていても `git add -f` でリポジトリに追加済みのファイルは変更チェックの対象となります。

## これによって何ができるようになるか

### コード例

```go
// Before: go build 後に +dirty が付いてしまう問題

// リポジトリ内容:
//   main.go (tracked)
//   go.mod  (tracked)
//   foo     (untracked: go build で生成されたバイナリ)

// go install を実行すると:
// mod  example.com/myapp  v1.0.0+dirty  ← 不正確
```

```go
// After: ビルドに無関係なファイルは無視される

// リポジトリ内容:
//   main.go (tracked, 未変更)
//   go.mod  (tracked, 未変更)
//   foo     (untracked: go build で生成されたバイナリ ← ビルドに不要)

// go install を実行すると:
// mod  example.com/myapp  v1.0.0  ← 正確
```

実践的なユースケースとして以下が挙げられます。

- **ローカル開発中のリリース**: `go build` でテストバイナリを生成しながら、同じリポジトリで `go install` を実行しても正確なバージョンが埋め込まれる
- **プロファイリング作業**: `cpu.prof` などのプロファイルファイルを残したままビルドしても `+dirty` が付かない
- **リリーススクリプト**: @mvdan 氏のようにバイナリを「親ディレクトリに書き出す」という回避策が不要になる

## 議論のハイライト

- **長年の既知問題**: @mvdan 氏が数年前の issue #50603 で同様の提案をしており、週次ベースで自身が悩まされていたと述べています。また、9つ以上のアップボートを集めたコメントが存在し、コミュニティの支持が広いことが確認されています。
- **`.gitignore` との関係**: @mvdan 氏が「VCS無視ファイルが含まれる場合も dirty と判定すべきでは」と提案しましたが、@rsc 氏は「VCSリポジトリへの登録有無」を基準とすることで整合性のある定義を提示しました。
- **issue #50603 での議論経緯**: Go 1.24 向けの `go build` バージョンスタンプ機能実装時、`vcs.modified` の精密化は複雑すぎるとして先送りされ、「別途proposalを立てること」と明示されていました。本提案はその経緯を引き継いだものです。
- **実装の見通し**: 提案者 @rsc 氏がほぼ実装済みであると述べており、技術的障壁は低いとみられます。`cmd/go` はすでに変更ファイルリストを取得しているため、ビルド使用ファイルとのクロスチェックを追加する変更で済む見込みです。
- **後方互換性**: `+dirty` が付かなくなるケースが増えるため、版情報が変わる可能性があります。ただし意味的にはより正確な情報になるため、実質的なデグレードではないと考えられます。

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/77897)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4000757564)
- [関連Issue #50603: go buildでバージョンスタンプを行う提案](https://github.com/golang/go/issues/50603)
- [関連Issue #50603 コメント: vcs.modified変更の議論](https://github.com/golang/go/issues/50603#issuecomment-2315888855)
