---
issue_number: 73878
title: "x/tools/go/analysis: add GoMod, ... fields to Module"
previous_status: active
current_status: likely_accept
changed_at: 2026-02-25T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3962620065
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/73878
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3962620065
  - title: "関連Issue #66315: x/tools/go/analysis: add Pass.Module field"
    url: https://github.com/golang/go/issues/66315
---
## 概要

`golang.org/x/tools/go/analysis` パッケージの `Module` 構造体に、`golang.org/x/tools/go/packages` パッケージの `Module` 構造体と同等のフィールド群（`GoMod`、`Dir`、`Replace`、`Main`、`Indirect`、`Time`、`Error` 等）を追加するproposalです。これにより、静的解析ツール（linter）の実装者がモジュール情報へより詳細にアクセスできるようになります。

## ステータス変更
**active** → **likely_accept**

`aclements`（提案レビューグループ）が2026年2月25日のweekly proposal review meetingにて、`adonovan`が提示した拡張仕様（`packages.Module`の全フィールドを追加する案）に基づいてlikely acceptと判断しました。当初の提案は`GoMod`フィールドのみの追加でしたが、`adonovan`が`go/packages.Module`と`cmd/go/internal/modinfo.ModulePublic`に共通する全フィールドを揃えることを提案し、その方向性がそのまま採用されました。

## 技術的背景

### 現状の問題点

現在の `go/analysis.Module` 構造体は以下の3フィールドのみを持ちます。

```go
// 現在の go/analysis.Module
type Module struct {
    Path      string // モジュールパス
    Version   string // モジュールバージョン
    GoVersion string // モジュールで使用するGoバージョン
}
```

これは2024年にissue #66315で追加された際、レビューで最小限のフィールドに絞られたものです。その結果、linterの実装者は以下の情報を直接取得できません。

- `go.mod` ファイルの場所（`GoMod`）
- モジュールのソースディレクトリ（`Dir`）
- モジュールの置き換え情報（`replace` ディレクティブ対応の `Replace`）
- メインモジュールかどうかの判定（`Main`）

`Module.Dir`を迂回的に推定する方法は存在します。たとえばファイルのパスからパッケージパスのサフィックスを除去する手順を踏めば可能ですが、`go build -modfile alternate.mod` のような代替 mod ファイル指定オプションが存在する場合、`filepath.Join(Dir, "go.mod")` では正確な `GoMod` の値が得られません。

### 提案された解決策

`go/analysis.Module` に `go/packages.Module` と同等のフィールドを追加します。

```diff
package analysis

type Module struct {
    Path      string       // module path
    Version   string       // module version
+   Replace   *Module      // replaced by this module
+   Time      *time.Time   // time version was created
+   Main      bool         // is this the main module?
+   Indirect  bool         // is this module only an indirect dependency of main module?
+   Dir       string       // directory holding files for this module, if any
+   GoMod     string       // path to go.mod file used when loading this module, if any
    GoVersion string       // go version used in module
+   Error     *ModuleError // error loading module
}

+// ModuleError holds errors loading a module.
+type ModuleError struct {
+    Err string // the error itself
+}
```

## これによって何ができるようになるか

依存関係のレピュテーションチェックや、`go.mod` の内容を解析するlinterが、迂回的な手順なしにモジュール情報へ直接アクセスできるようになります。

### コード例

```go
// Before: Go.modのパスを迂回的に推定する（エラーが発生しやすい）
func run(pass *analysis.Pass) (interface{}, error) {
    if pass.Module == nil {
        return nil, nil
    }
    // Fsetからファイルパスを取得してDir/GoModを推測する必要がある
    var goModPath string
    pass.Fset.Iterate(func(f *token.File) bool {
        dir := filepath.Dir(f.Name())
        pkgSuffix := strings.TrimPrefix(pass.Pkg.Path(), pass.Module.Path)
        pkgSuffix = strings.TrimPrefix(pkgSuffix, "/")
        if strings.HasSuffix(dir, pkgSuffix) {
            moduleDir := strings.TrimSuffix(dir, pkgSuffix)
            goModPath = filepath.Join(moduleDir, "go.mod")
            return false
        }
        return true
    })
    // goModPath を使った処理...
    return nil, nil
}

// After: Module.GoModに直接アクセスできる
func run(pass *analysis.Pass) (interface{}, error) {
    if pass.Module == nil || pass.Module.GoMod == "" {
        return nil, nil
    }
    goModPath := pass.Module.GoMod // 直接アクセス可能
    // goModPath を使った処理...
    return nil, nil
}
```

## 議論のハイライト

- **当初の絞り込みと拡張の方針**: issue #66315 で `Module` フィールドが初めて導入された際、設計レビューで最小の3フィールドに絞られた経緯があります。今回の変更はその際に「後で追加可能」と留保されていた拡張を実施するものです。
- **`Dir`だけでよいか**: 提案者は当初 `GoMod` のみ要求しましたが、`go build -modfile` による代替modファイル指定の存在が指摘（`sudo-bmitch`）され、`Dir` と `GoMod` は別フィールドとして持つ必要があることが明確になりました。
- **フィールドの範囲**: `adonovan` が `packages.Module` と `cmd/go/internal/modinfo.ModulePublic` の両方に共通するフィールド、すなわち `packages.Module` の全フィールドを揃える方向を提案し、提案者がPR（golang/tools#577）を更新する形で合意しました。
- **`ModuleError` の追加**: モジュールロード失敗時の情報を提供するために `ModuleError` 型も新たに追加されます。既存の analyzer はモジュール情報が `nil` でも動作できる設計になっているため、後方互換性は保たれます。
- **迂回手段の不完全性の確認**: `adonovan` は当初「ファイルパスからDirを推定できる」と示しましたが、`Pass.Pkg.Path()` はパッケージパス（インポートパス）であり、ファイルシステムパスではないという指摘があり、推定の手順が非自明であることが再確認されました。

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/73878)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3962620065)
- [関連Issue #66315: x/tools/go/analysis: add Pass.Module field](https://github.com/golang/go/issues/66315)
