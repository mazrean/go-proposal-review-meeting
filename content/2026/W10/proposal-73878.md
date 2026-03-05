---
issue_number: 73878
title: "x/tools/go/analysis: add GoMod, ... fields to Module"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-03-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
related_issues:
  - title: "関連Issue #66315: x/tools/go/analysis: add Pass.Module field"
    url: https://github.com/golang/go/issues/66315
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/73878
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
---
## 概要

`golang.org/x/tools/go/analysis` パッケージの `Module` 型に、`GoMod`、`Dir`、`Replace`、`Time`、`Main`、`Indirect`、`Error` などの複数フィールドを追加するプロポーザルです。`go/packages` パッケージの `Module` 型と同等のフィールドセットを analysis フレームワークでも利用できるようにすることで、アナライザー（静的解析ツール）の実装者がモジュール情報に容易にアクセスできるようにします。

## ステータス変更

**likely_accept** → **accepted**

2026年3月4日、コアチームメンバーの @aclements が「議論における合意に変化はない」としてプロポーザルを正式に受理しました。2025年5月に提案が開始され、@adonovan がフィールド拡張の方向性を示し、その後 PR が作成されました。2026年2月の週次プロポーザルレビューミーティングで `likely_accept` となり、約1週間後に `accepted` へと移行しました。

## 技術的背景

### 現状の問題点

`golang.org/x/tools/go/analysis` パッケージの `Pass.Module` フィールドで参照できる `Module` 型は、現在以下の3フィールドのみを持ちます（Issue #66315 でのレビュー時に大幅に削減された経緯があります）。

```go
// 現在の analysis.Module（フィールドが少ない）
type Module struct {
    Path      string // モジュールパス
    Version   string // モジュールバージョン
    GoVersion string // go.mod に記載された Go バージョン
}
```

一方、`golang.org/x/tools/go/packages` パッケージの `Module` 型はより多くのフィールドを持ち、`GoMod`（go.mod ファイルのパス）や `Dir`（モジュールのディレクトリ）などが含まれます。この不一致により、アナライザーの実装者がモジュールの `go.mod` ファイルへアクセスするには、ファイルセット内のファイルパスからディレクトリを逆算するといった複雑なワークアラウンドが必要でした。

`go build -modfile alternate.mod` のように通常と異なるモジュールファイルを指定する場合、`filepath.Join(Dir, "go.mod")` では正確なパスを得られないため、`GoMod` フィールドを独立して持つことにも意義があります。

### 提案された解決策

`analysis.Module` 型を `go/packages.Module` と同等のフィールドセットに拡充します。@adonovan が提案した最終的な diff は以下のとおりです。

```go
package analysis

type Module struct {
    Path      string       // モジュールパス
    Version   string       // モジュールバージョン
    Replace   *Module      // 置き換え先モジュール（replace ディレクティブ）
    Time      *time.Time   // バージョンが作成された時刻
    Main      bool         // メインモジュールか否か
    Indirect  bool         // メインモジュールの間接依存か否か
    Dir       string       // モジュールファイルを保持するディレクトリ
    GoMod     string       // 使用する go.mod ファイルのパス
    GoVersion string       // go.mod に記載された Go バージョン
    Error     *ModuleError // モジュール読み込みエラー
}

// ModuleError はモジュール読み込みエラーを保持します。
type ModuleError struct {
    Err string // エラーメッセージ
}
```

## これによって何ができるようになるか

アナライザーの実装者がモジュール情報に直接アクセスできるようになり、以下のようなユースケースが実現しやすくなります。

- **依存関係の評判チェック**: `GoMod` フィールドから `go.mod` / `go.sum` ファイルを直接読み込み、依存パッケージのセキュリティや品質を確認するアナライザーを実装できます（提案者 @AkihiroSuda の [gosocialcheck](https://github.com/AkihiroSuda/gosocialcheck) がこのユースケースです）。
- **メインモジュールのみへの制限**: `Main` フィールドを使い、解析対象をメインモジュールに限定するアナライザーを簡潔に実装できます。
- **モジュール置き換えの検出**: `Replace` フィールドを使い、`replace` ディレクティブが使われているモジュールを検出するアナライザーが書けます。

### コード例

```go
// Before: go.mod のパスを取得するための複雑なワークアラウンド
func getGoModPath(pass *analysis.Pass) string {
    // ファイルセットから任意のファイルパスを取得し、
    // パッケージパスとモジュールパスから相対パスを計算してディレクトリを逆算する
    // （バージョンサフィックスの処理も必要で、エラーが起きやすい）
    var anyFile string
    pass.Fset.Iterate(func(f *token.File) bool {
        anyFile = f.Name()
        return false
    })
    dir := filepath.Dir(anyFile)
    relPkg := strings.TrimPrefix(pass.Pkg.Path(), pass.Module.Path)
    moduleDir := strings.TrimSuffix(dir, relPkg)
    return filepath.Join(moduleDir, "go.mod")
}

// After: フィールドに直接アクセス
func getGoModPath(pass *analysis.Pass) string {
    if pass.Module == nil {
        return ""
    }
    return pass.Module.GoMod // 直接取得できる
}
```

## 議論のハイライト

- **GoMod と Dir の両方が必要な理由**: `go build -modfile alternate.mod` のようにデフォルトとは異なる go.mod ファイルを指定できるため、`filepath.Join(Dir, "go.mod")` では常に正確なパスを得られません。`Dir` と `GoMod` を別々のフィールドとして持つことに合理性があります（@sudo-bmitch が指摘）。
- **既存フィールド削減の経緯**: Issue #66315 で `Pass.Module` が導入された際、当初は多くのフィールドが含まれていましたが、レビュー段階で現在の3フィールドに削減されました。今回の提案は「後から追加できるよう予約していた」方針に基づく自然な拡充です。
- **拡充の範囲**: 当初は `GoMod` のみの追加提案でしたが、@adonovan が `go/packages.Module` と `cmd/go/internal/modinfo.ModulePublic` の共通フィールドを全て追加することを提案し、最終的にその方向で承認されました。
- **ワークアラウンドの困難さ**: @adonovan は当初「既存情報から `Dir` を計算できる」と述べましたが、`Pass.Pkg.Path()` がファイルシステムパスではなくパッケージパス（インポートパス）を返すため、実際には複雑な逆算が必要であることが明確になりました。
- **実装PR**: [golang/tools#577](https://github.com/golang/tools/pull/577) および [CL 676455](https://go-review.googlesource.com/c/tools/+/676455) で実装が進められています。

## 関連リンク

- [関連Issue #66315: x/tools/go/analysis: add Pass.Module field](https://github.com/golang/go/issues/66315)
- [Proposal Issue](https://github.com/golang/go/issues/73878)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4000757564)
