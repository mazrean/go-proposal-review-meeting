---
issue_number: 76285
title: "go/token: add (\\*File).String method"
previous_status: active
current_status: likely_accept
changed_at: 2026-02-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3872311559
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/76285
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3872311559
  - title: "関連提案: File.End() メソッド (#75849)"
    url: https://github.com/golang/go/issues/75849
---
## 概要
`go/token.File` 型に `String()` メソッドを追加し、デバッグ時の出力を改善します。現在、この型を `fmt.Println` で出力すると、内部構造の生ダンプが表示され、巨大な行オフセットテーブルが含まれるため非常に読みにくい状態です。この提案により、ファイル名・開始位置・終了位置を含む簡潔で人間が読みやすい文字列表現が得られます。

## ステータス変更
**active** → **likely_accept**

2026年2月9日の提案レビュー会議で "likely accept"（承認見込み）に移行しました。提案は当初「ファイル名のみを返す」というシンプルなものでしたが、レビュアーの@mvdanからの提案で「サイズや範囲情報も含めるべき」という意見が出され、最終的に `Name(Base-End)` 形式の出力が採用されました。コアチームメンバー（@aclements、@griesemer）は提案に賛成しており、実装の詳細（出力フォーマット）について柔軟性を持たせることで合意しています。

## 技術的背景

### 現状の問題点
`go/token.File` 型はフィールドが非公開であるため、`fmt.Println` でデバッグ出力すると、内部構造がそのまま表示されます。

```go
f := token.NewFileSet().AddFile("foo.go", -1, 0)
fmt.Println(f)
// 出力: "&{foo.go 1 0 {{} {0 0}} [0] []}"
// 巨大なファイルでは行オフセットテーブルが延々と表示される
```

この出力は開発者にとって以下の問題があります:
- 必要な情報（ファイル名）が構造体ダンプの中に埋もれる
- 大規模なソースファイルでは、行オフセット配列が数千行にわたって出力される
- デバッグ用の `log.Println(fset.File(pos))` を急いで追加したときに、ログが読めなくなる

### 提案された解決策
`fmt.Stringer` インターフェースを実装する `String()` メソッドを追加します。

```go
package token // "go/token"

// String returns a brief description of the file.
func (f *File) String() string
```

実装例（議論を経て改良されたバージョン）:
```go
func (f *File) String() string {
    return fmt.Sprintf("%s(%d-%d)", f.Name(), f.Base(), f.End())
}
```

このフォーマットは以下を含みます:
- **ファイル名**: `Name()` メソッドの戻り値
- **Base位置**: ファイルが `FileSet` 内で占める開始位置
- **End位置**: ファイルの終了位置（関連提案 #75849 で追加される `End()` メソッドを使用）

## これによって何ができるようになるか

デバッグ時のログ出力が劇的に改善されます。AST解析ツールや静的解析ツールの開発時に、位置情報を含む簡潔なファイル表現を即座に得られるようになります。

### コード例

```go
// Before: 従来の書き方（問題のある出力）
fset := token.NewFileSet()
file := fset.AddFile("main.go", -1, 1024)
fmt.Println(file)
// 出力: "&{main.go 1 1024 {{} {0 0}} [0 128 256 ...] []}"
// → 内部フィールドが露出し、大規模ファイルでは行配列が膨大に

// After: String()メソッド実装後
fmt.Println(file)
// 出力: "main.go(1-1025)"
// → ファイル名とPos範囲が一目瞭然
```

**実用例**（gopls等のツールでの典型的な使用）:
```go
// デバッグ時に位置情報を確認
pos := someNode.Pos()
file := fset.File(pos)
log.Printf("Processing node in %s at offset %d", file, file.Offset(pos))
// 出力: "Processing node in example.go(100-5000) at offset 234"
```

## 議論のハイライト

- **出力フォーマットの進化**: 当初は「ファイル名のみ」を返す提案でしたが、@mvdanが「`Name()` メソッドがあるので、`String()` ではより包括的な情報を提供すべき」と指摘。最終的に `Name(Base-End)` 形式が採用されました。

- **GoStringerの検討**: @mateusz834が `fmt.GoStringer`（`%#v` 用）の実装も提案しましたが、現時点ではシンプルな `String()` のみで進行しています。

- **実装の柔軟性**: @adonovanは「出力フォーマットの詳細で提案レビュー委員会の時間を取りたくない」として、`String()` メソッドの仕様を「brief description（簡潔な説明）」とし、実装者に裁量を持たせることを提案しました。

- **関連提案との連携**: #75849（`File.End()` メソッドの追加）が先に承認されており、この提案の実装では `End()` メソッドを活用できます。

- **x/toolsでの需要**: 提案者の調査により、golang.org/x/tools リポジトリ内だけで16箇所以上が `Base() + Size()` のパターンを使用しており、類似の改善需要が高いことが示されています。

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/76285)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3872311559)
- [関連提案: File.End() メソッド (#75849)](https://github.com/golang/go/issues/75849)
