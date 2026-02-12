---
issue_number: 73878
title: "x/tools/go/analysis: add GoMod, ... fields to Module"
previous_status: 
current_status: active
changed_at: 2026-02-11T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3886687081
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/73878
  - title: "関連Issue: Pass.Module追加の元提案"
    url: https://github.com/golang/go/issues/66315
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3886686378
---
## 概要

`x/tools/go/analysis`パッケージの`Module`型に、`go/packages`パッケージで既に提供されている`Dir`、`GoMod`等のフィールドを追加する提案です。これにより、解析ツール（リンター）の実装者が、解析対象パッケージの`go.mod`ファイルのパスやモジュールディレクトリを直接取得できるようになります。

## ステータス変更

**(空)** → **active**

本提案は2026年2月11日にproposal review groupによってactiveステータスに移行されました。提案者のAkihiroSuda氏が依存関係の評判チェックツール[gosocialcheck](https://github.com/AkihiroSuda/gosocialcheck)の実装で`go.mod`（および同一ディレクトリの`go.sum`）へのアクセスが必要となったことが発端です。当初は`GoMod`フィールドのみの追加提案でしたが、メンテナのadonovan氏がレビュー中に「`go/packages.Module`と共通するフィールドをすべて追加すべき」と提案し、より包括的なAPI追加へと発展しました。

## 技術的背景

### 現状の問題点

現在、`golang.org/x/tools/go/analysis.Module`型には以下の3つのフィールドしかありません（[提案#66315](https://github.com/golang/go/issues/66315)で追加）:

```go
type Module struct {
    Path      string // モジュールパス
    Version   string // モジュールバージョン
    GoVersion string // モジュールで使用されているGoバージョン
}
```

一方、`golang.org/x/tools/go/packages.Module`には10個のフィールドがあり、`Dir`（モジュールディレクトリ）や`GoMod`（`go.mod`ファイルのパス）など、リンター実装で有用な情報が含まれています。

回避策として、`Pass.Fset`のファイル名からディレクトリを推測することは可能ですが、パッケージパスとモジュールパスのサフィックス処理が複雑で、エラーが発生しやすい実装になります。また、`go build -modfile alternate.mod`のような代替`go.mod`を指定するケースでは、`filepath.Join(Dir, "go.mod")`という単純な結合では正確なパスを得られません。

### 提案された解決策

`analysis.Module`型を拡張し、`packages.Module`と`cmd/go/internal/modinfo.ModulePublic`に共通するすべてのフィールドを追加します:

```go
type Module struct {
    Path      string       // モジュールパス
    Version   string       // モジュールバージョン
    Replace   *Module      // このモジュールに置き換えられる
    Time      *time.Time   // バージョンが作成された時刻
    Main      bool         // これがメインモジュールか
    Indirect  bool         // メインモジュールの間接的な依存のみか
    Dir       string       // モジュールファイルを保持するディレクトリ
    GoMod     string       // モジュールロード時に使用されたgo.modファイルのパス
    GoVersion string       // モジュールで使用されているGoバージョン
    Error     *ModuleError // モジュールロード時のエラー
}

type ModuleError struct {
    Err string // エラー本体
}
```

## これによって何ができるようになるか

1. **依存関係の評判チェック**: [gosocialcheck](https://github.com/AkihiroSuda/gosocialcheck)のようなツールが、`go.mod`/`go.sum`にアクセスして依存モジュールがCNCF Graduatedプロジェクト等の信頼できる組織で採用されているかを検証できます。

2. **モジュール置換の検証**: `Replace`フィールドにより、`go.mod`の`replace`ディレクティブを検出し、ローカル依存やフォーク使用を警告するリンターが実装可能になります。

3. **Goバージョン互換性チェック**: 既存の`GoVersion`に加え、`Main`フィールドでメインモジュールを識別し、依存関係のGoバージョン要件と比較する解析が容易になります。

4. **ワークスペース対応解析**: メインモジュールと外部依存を区別し、プロジェクト固有のルールと外部ライブラリ向けルールを使い分けることができます。

### コード例

```go
// Before: go.modパスの推測（複雑でエラーが起きやすい）
func run(pass *analysis.Pass) (interface{}, error) {
    // Pass.Fsetから任意のファイルパスを取得
    // Module.PathとPackage.Pathの差分を計算
    // ディレクトリからサフィックスを削除してModule.Dirを推測
    // filepath.Join(moduleDir, "go.mod") を構築
    // ※-modfileオプション使用時は正しいパスにならない
}

// After: 直接アクセス可能
func run(pass *analysis.Pass) (interface{}, error) {
    if pass.Module == nil {
        return nil, nil // モジュール情報なし
    }

    if pass.Module.GoMod != "" {
        // go.modファイルを直接読み取り
        data, err := os.ReadFile(pass.Module.GoMod)
        // go.sumは同じディレクトリにある
        sumPath := strings.TrimSuffix(pass.Module.GoMod, ".mod") + ".sum"
    }

    if pass.Module.Main && pass.Module.Replace != nil {
        pass.Reportf(pass.Files[0].Pos(),
            "main module uses replace directive")
    }
}
```

## 議論のハイライト

- **最小セットvs最大セット**: 当初Timothy King氏が`{Path, Version, GoVersion}`の最小セットを提案しましたが、最終的にadonovan氏が「将来の追加を避けるため、`packages.Module`の全フィールドを一度に追加すべき」と主張し、この方針が採用されました。

- **代替ビルドシステムへの配慮**: Bazel、Pants、Buck等の`go.mod`を使用しないビルドシステムでもフィールドを埋められるかが確認され、BazelのrulegoメンテナやPlease開発者から実装可能との回答を得ました。`Time`や`Indirect`等の一部フィールドが埋められない可能性がありますが、部分的な情報でも`Pass.Module != nil`で提供される方針です。

- **GoModとDirの両方が必要な理由**: コメントで「`Dir`があれば`GoMod`は`filepath.Join(Dir, "go.mod")`で計算できるのでは」という疑問が出ましたが、sudo-bmitch氏が`go build -modfile alternate.mod`のように代替`go.mod`ファイルを指定できるケースを指摘し、`GoMod`フィールドの必要性が確認されました。

- **nil処理**: `Pass.Module`はドライバによっては`nil`になる可能性があるため、解析ツール側で必ず`nil`チェックが必要です。また、`Version`フィールドは`(devel)`や`""`（ワークスペースモジュール）になる場合があります。

- **実装PR**: [golang/tools#577](https://github.com/golang/tools/pull/577)で初期実装が提出されていますが、adonovan氏によるHold+1のため、フィールド追加の方針確定待ちの状態です。

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/73878)
- [関連Issue: Pass.Module追加の元提案](https://github.com/golang/go/issues/66315)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3886686378)
