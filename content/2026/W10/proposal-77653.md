---
issue_number: 77653
title: "cmd/go: change \\`go mod init\\` default go directive back to 1.N"
previous_status: active
current_status: accepted
changed_at: 2026-03-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77653
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
  - title: "元の変更を導入した Issue #74748"
    url: https://github.com/golang/go/issues/74748
  - title: "1.26 バックポート Issue #77860"
    url: https://github.com/golang/go/issues/77860
  - title: "1.N.0 案の分離 Issue #77923"
    url: https://github.com/golang/go/issues/77923
---
## 概要

`go mod init` コマンドが go.mod ファイルに書き込む `go` ディレクティブのデフォルト値を、Go 1.26.0 で変更された挙動から元の挙動に戻すことを提案するものです。具体的には、実行中のツールチェーンバージョン `1.N.M` をそのままデフォルト値として使用する従来の動作に復元します。

## ステータス変更
**active** → **accepted**

Go Command Working Group とプロポーザルレビューグループの両方が、1.26.0 の変更を元に戻すことで合意しました。変更が初心者にとって特に混乱を招くものであること、また変更が本来はプロポーザルプロセスを経るべきだったという認識が共有されたことで、1.26.1 パッチリリースへのバックポートを目標として承認されました。

## 技術的背景

### 現状の問題点

Go 1.26.0 では、`go mod init` が go.mod ファイルに書き込む `go` ディレクティブのデフォルト値が変更されました（issue #74748 による変更）。変更後の動作は以下の通りです。

- 安定版 Go 1.N.M を使用している場合: デフォルトは `go 1.(N-1).0`
- RC版または開発版 Go 1.N を使用している場合: デフォルトは `go 1.(N-2).0`

この結果、Go 1.26 のツールチェーンで `go mod init` を実行すると go.mod には `go 1.25.0` と書き込まれ、1.26 の言語機能（例: `new(42)` に型引数を渡す新機能）を使おうとするとビルドエラーになるという混乱が生じました。

```go
// Go 1.26 ツールチェーンで go mod init した直後
// go.mod に "go 1.25.0" と書かれた状態でビルドすると...

// t.go
package main
import "fmt"
func main() {
    fmt.Println(new(42))  // 1.26 の新機能
}
```

```
$ go build
# t
./t.go:6:14: new(42) requires go1.26 or later (-lang was set to go1.25; check go.mod)
```

つまり、1.26 ツールチェーンをインストールして新しいモジュールを作ったにもかかわらず、1.26 の機能が使えないという直感に反する状況が発生していました。

### 提案された解決策

Go 1.26.0 以前の動作に戻し、`go mod init` は実行中のツールチェーンのバージョン（`1.N.M`）をそのまま `go` ディレクティブに使用するよう変更します。Go 1.26.1 でのバックポートを目標として実装が進められています。

## これによって何ができるようになるか

この変更により、`go mod init` を実行した後すぐに現在のツールチェーンの全機能が利用可能になります。開発者はモジュール作成直後にバージョンを手動で修正する必要がなくなります。

### コード例

```go
// Before（Go 1.26.0 での挙動）:
// $ go version → go1.26.0
// $ go mod init mymodule
// → go.mod に "go 1.25.0" と書き込まれる
// → 1.26 の言語機能が使えずビルドエラー

// After（提案後の挙動 / 1.26.1 以降）:
// $ go version → go1.26.0
// $ go mod init mymodule
// → go.mod に "go 1.26.0" と書き込まれる
// → 1.26 の全機能がすぐに利用可能
```

実践的なメリット:

- **新規プロジェクト**: インストールしたばかりのツールチェーンの機能をすぐに使える
- **学習者・初心者**: バージョン管理の仕組みを深く知らなくてもビルドエラーに遭遇しない
- **ライブラリ作者**: バージョンポリシーを意図的に選択できる（デフォルトが邪魔をしない）

## 議論のハイライト

- **1.26.0 の変更はプロポーザルプロセスを経ていなかった**: `@ianlancetaylor` の「一度決定したことは新情報がなければ覆さない」という原則に対し、反対派は「そもそもプロポーザルプロセスを経た決定ではない」と反論し、これがレビューの正当性を支持した。
- **エコシステム効果 vs 個人開発者体験のトレードオフ**: 1.26.0 の変更はライブラリ作者が無意識に 1.N を必要とするモジュールを公開することを防ぐエコシステム配慮が意図でしたが、`@aclements` は「1.(N-1) の利点は微妙で上級者向けの考慮であり、1.N の利点はわかりやすい」と整理し、初心者に不利なトレードオフと結論付けた。
- **混乱の非対称性**: `gopls` や `go vet` は go.mod の宣言バージョンより新しい API 使用を警告するが、標準ライブラリの振る舞い変化（シグネチャ変更なし）は検出できない。また `go build` は警告しないため、初心者が問題に気づきにくい。
- **Go Command Working Group の参加**: プロポーザルレビューグループと Go Command Working Group の両組織が合意してリバートを支持し、もし将来的に変更を再検討する場合は正式なプロポーザルプロセスを経るべきとの方針を確認した。
- **1.N.M か 1.N.0 かの議論**: 一部のコミュニティメンバーから「パッチバージョン（M）を固定して 1.N.0 にすべき」との声も上がったが、`@aclements` は「完全に元に戻す保守的なアプローチを取り、M の扱いは別途議論する」と判断し、issue #77923 として分離して提出された。

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/77653)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4000757564)
- [元の変更を導入した Issue #74748](https://github.com/golang/go/issues/74748)
- [1.26 バックポート Issue #77860](https://github.com/golang/go/issues/77860)
- [1.N.0 案の分離 Issue #77923](https://github.com/golang/go/issues/77923)
