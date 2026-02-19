---
issue_number: 76285
title: "go/token: add (\\*File).String method"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-02-18T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3923200976
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/76285
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3923200976
  - title: "関連Issue: go/token: add File.End method #75849"
    url: https://github.com/golang/go/issues/75849
---
## 概要

`go/token` パッケージの `*File` 型に `String()` メソッドを追加するプロポーザルです。これにより、`*File` をデバッグ目的で `fmt.Println` などに渡した際に、内部構造のダンプではなく人間が読みやすい簡潔な説明が表示されるようになります。

## ステータス変更

**likely_accept** → **accepted**

2026年2月9日に "likely accept" とされた後、1週間以上反対意見がなかったため、2026年2月18日に @aclements（提案レビューグループ代表）によって正式に accepted となりました。実装CL（go.dev/cl/743280）はすでに "likely accept" の段階で投稿されており、承認と同時に実装が進められる状態にあります。

## 技術的背景

### 現状の問題点

`go/token` パッケージの `*File` 型は `fmt.Stringer` インターフェースを実装していません。そのため、デバッグ時に `fmt.Println` や `log.Println` でファイル情報を出力しようとすると、Go のデフォルトのフォーマット（構造体の全フィールドダンプ）が適用され、意図しない出力が得られます。

```go
// 現状の問題：巨大な行オフセットテーブルを含む内部構造が丸ごとダンプされる
f := token.NewFileSet().AddFile("foo.go", -1, 0)
fmt.Println(f) // "&{foo.go 1 0 {{} {0 0}} [0] []}"
```

実際のコードベース（golang.org/x/tools など）では、ファイルの終端位置を求めるために `Base() + Size()` という計算式が頻繁に用いられており、デバッグ時に `*File` をそのまま出力できないことが不便さを生んでいました。

### 提案された解決策

`*File` 型に以下の `String()` メソッドを追加します。

```go
package token // "go/token"

// String returns a brief description of the file.
func (f *File) String() string
```

内部実装は以下の形式で、ファイル名とその位置範囲（Base から End）を返します。

```go
func (f *File) String() string {
    return fmt.Sprintf("%s(%d-%d)", f.Name(), f.Base(), f.End())
}
```

なお、`f.End()` は同じく最近追加されたプロポーザル（Issue #75849）で追加された `(*File).End()` メソッドを利用しています。

## これによって何ができるようになるか

Goの構文解析やAST（抽象構文木）操作を行うツール開発者が、`token.File` をデバッグ出力する際に有用な情報を得られるようになります。

### コード例

```go
// Before: 内部構造がそのまま出力され、行オフセットテーブルなどが露出する
f := token.NewFileSet().AddFile("foo.go", -1, 100)
fmt.Println(f)
// 出力例: "&{foo.go 1 100 {{} {0 0}} [0] []}"

// After: 簡潔で人間が読みやすい説明が出力される
f := token.NewFileSet().AddFile("foo.go", -1, 100)
fmt.Println(f)
// 出力例: "foo.go(1-101)"

// デバッグ用ログ出力も自然に記述できる
fset := token.NewFileSet()
file := fset.File(pos)
log.Println(file) // "foo.go(1-101)" のような出力

// ファイルセット内のファイル一覧表示
fset.Iterate(func(f *token.File) bool {
    fmt.Println(f) // 各ファイルの簡潔な説明が表示される
    return true
})
```

## 議論のハイライト

- **String メソッドの内容を何にすべきか**: 当初の提案ではファイル名のみ（`Name()` の返り値と同じ）を返すことが提示されたが、@mvdan が「ファイル名だけなら `Name()` を直接呼べば良い」と指摘し、より情報量の多いフォーマットを提案。これを受けて @adonovan が位置範囲（Base から End）を含む `"%s(%d-%d)"` 形式に修正した。
- **デバッグ文字列のさらなる詳細化**: @griesemer が「デバッグ文字列をもっと自己説明的にすべき」とコメント。@adonovan は `"token.File{Name: %s, Pos: %d, End: %d}"` 形式も提案したが、String メソッドの出力形式は「人間が読める簡潔な説明」として実装者に裁量を持たせる方向で議論が収束した。
- **GoString との使い分け**: `fmt.GoStringer` インターフェース（`%#v` フォーマットに対応）を別途実装することでデバッグ用の詳細表示とシンプルな String メソッドを分ける案も出たが、採用はされなかった。
- **実装はすでに準備済み**: @adonovan はプロポーザル段階でコードを既に記述しており、承認を待つだけの状態であった。CL（go.dev/cl/743280）も "likely accept" と同日に投稿された。
- **関連プロポーザルとの連携**: `(*File).End()` を追加するプロポーザル（Issue #75849）が先行して閉じられており、本プロポーザルの実装がその成果を自然に活用できる形になっている。

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/76285)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3923200976)
- [関連Issue: go/token: add File.End method #75849](https://github.com/golang/go/issues/75849)
