---
issue_number: 77653
title: "cmd/go: change \\`go mod init\\` default go directive back to 1.N"
previous_status: 
current_status: active
changed_at: 2026-02-25T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3962620065
related_issues:
  - title: "関連Issue #76857"
    url: https://github.com/golang/go/issues/76857
  - title: "関連Issue #67574"
    url: https://github.com/golang/go/issues/67574
  - title: "Proposal Issue #77653"
    url: https://github.com/golang/go/issues/77653
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3962620065
  - title: "元の変更 Issue #74748"
    url: https://github.com/golang/go/issues/74748
---
## 概要

`go mod init` コマンドが新しいモジュールを作成する際にgo.modファイルの `go` ディレクティブに設定するデフォルトバージョンを、Go 1.26で変更された動作（`1.(N-1)`）から元の動作（現在のツールチェーンバージョン `1.N`）に戻すよう求めるプロポーザルです。

## ステータス変更
**(新規)** → **active**

このプロポーザルは2026年2月17日に提出され、同年2月25日に `@aclements` によってアクティブ（提案レビュー会議での審議対象）に移行しました。aclements は「元の変更（#74748）はプロポーザルプロセスを経ておらず、その全影響を誰も十分に把握していなかった」と認め、変更のリバートを「検討方向」として示しています。Go Command Working Group が翌日（2月26日）に審議を行う予定です。

## 技術的背景

### 現状の問題点

Go 1.26では、`go mod init` が生成するgo.modの `go` ディレクティブが現在のツールチェーンバージョンではなく、1つ前のバージョン（`1.(N-1).0`）に設定されるよう変更されました（issue #74748）。これにより、1.26ツールチェーンを使用してモジュールを作成しても、go.modには `go 1.25.0` が設定され、Go 1.26の言語機能を使おうとするとビルドエラーが発生します。

```go
// 1.26ツールチェーンでgo mod initした後
// go.mod: go 1.25.0  ← 自動設定された値

// t.go
package main

import "fmt"

func main() {
    fmt.Println(new(42))  // Go 1.26の機能
}
```

```
$ go build
# t
./t.go:6:14: new(42) requires go1.26 or later (-lang was set to go1.25; check go.mod)
```

1.26ツールチェーンをインストールして `go mod init` を実行したばかりのユーザーが、1.26の機能を使えないという混乱が生じます。

### 提案された解決策

`go mod init` のデフォルト `go` ディレクティブを、現在のツールチェーンバージョン `1.N` に戻すことを提案しています。これはGo 1.21以前から続いてきた元の動作です。

また、プロポーザルレビュー会議では `go mod init -go=latest` フラグの追加（`@mvdan`が提案）など、フラグによるオプトインも議論されています。

## これによって何ができるようになるか

このプロポーザルが承認されリバートが実施されると、以下が実現します。

1. **初心者にとっての混乱解消**: `go mod init` 後すぐに最新言語機能が使えるため、「なぜビルドエラーが出るのか」という余分な学習コストがなくなります。
2. **Goプレイグラウンドとの一貫性**: go.dev/play/ は `1.N` を使用するため、プレイグラウンドで動くコードがローカルでもそのまま動くようになります。
3. **「最小驚き原則」の回復**: 最新ツールチェーンをダウンロードしたということ自体が、そのバージョンの機能を使う意図の表明であるという直感的な認識と一致します。

### コード例

```go
// Before（Go 1.26での動作）: go mod init実行後のgo.mod
module myapp

go 1.25.0  // ← 1.26ツールチェーン使用時でも1つ前のバージョンが設定される

// この後、1.26の新機能を使うにはgo mod edit -go=1.26が必要

// After（リバート後）: go mod init実行後のgo.mod
module myapp

go 1.26.0  // ← 現在のツールチェーンバージョンが設定される
// 追加操作なしに1.26の全機能が使える
```

## 議論のハイライト

- **元変更（#74748）のプロセス上の問題**: `@ianlancetaylor` は当初「新情報がなければ決定を覆さない」と述べたが、`@aclements` は後に「元の変更はプロポーザルレビューを経ておらず、全員がその完全な影響を理解していなかった」と認めた。これが重要な論点となり、アクティブステータスへの移行に繋がった。
- **1.(N-1)デフォルトの利点に関する議論**: Go Teamは「大企業がGoバージョン更新を厳しく制限している場合、新規モジュールが`1.N`を要求することで間接的な互換性問題が生じる」ことを理由に挙げたが、コミュニティからはその利点が限定的・理論的にすぎるという反論が多数寄せられた。
- **初心者への悪影響**: `@aclements`は「`1.(N-1)`の理由は上級者向けの考慮事項であり、最も影響を受けやすい初心者には全く理解されない」として、現状のデフォルト設定のトレードオフが誤っていると指摘した。
- **govetとgoplsによる部分的な補完**: `@mvdan`は「新APIを古いgo directiveのモジュールで使った場合、goplsやgo vetが警告を出す」と述べたが、それでも完全ではなく（標準ライブラリの振る舞い変更などは検出困難）、ビルドエラーにはならない点が問題として残る。
- **エコシステムへの影響のトレードオフ**: `@apparentlymart`は「新バージョンの早期採用者を減らすことで、新バージョン使用の利点そのものが失われる悪循環が起きる可能性がある」という鋭い指摘を行った。

## 関連リンク

- [関連Issue #76857](https://github.com/golang/go/issues/76857)
- [関連Issue #67574](https://github.com/golang/go/issues/67574)
- [Proposal Issue #77653](https://github.com/golang/go/issues/77653)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3962620065)
- [元の変更 Issue #74748](https://github.com/golang/go/issues/74748)
