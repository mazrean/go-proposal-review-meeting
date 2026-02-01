---
issue_number: 17747
title: "cmd/vet: check for missing Err calls for bufio.Scanner and sql.Rows"
previous_status: discussions
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "proposal: cmd/vet: check for missing Err calls for bufio.Scanner and sql.Rows · Issue #17747 · golang/go"
    url: https://github.com/golang/go/issues/17747
  - title: "Review Minutes (2026-01-28)"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
---

## 要約

## 概要

`bufio.Scanner`や`sql.Rows`を使用する際に、エラーチェック用の`Err()`メソッド呼び出しを忘れることは非常によくある誤りです。このproposalは、`go vet`がこのような見落としを検出し、警告を出すべきだという提案です。

## ステータス変更

**discussions** → **active**

2022年2月に一度「active」になったものの、実装の精度評価のため長期間「hold」状態でした。2025年12月にholdが解除され、2026年1月の実装テストと議論を経て、再び「active」として議論が進行中です。最新の2026年1月28日のProposal Review Meetingでは、チェック対象の絞り込みについてコメントが追加され、現在も設計の詳細を調整中です。

## 技術的背景

### 現状の問題点

`bufio.Scanner`や`sql.Rows`は、ループ処理中にエラーが発生した場合、そのエラーを内部に保持し、最後に`Err()`メソッドを呼び出すことで取得する設計になっています。しかし、多くの開発者がこの最終的なエラーチェックを忘れてしまいます。

**問題のあるコード例:**

```go
// bufio.Scannerの場合
scanner := bufio.NewScanner(file)
for scanner.Scan() {
    line := scanner.Text()
    fmt.Println(line)
}
// scanner.Err()のチェックを忘れている!
// I/Oエラーやトークンサイズ超過が発生しても気づかない

// sql.Rowsの場合
rows, err := db.Query("SELECT * FROM users")
if err != nil {
    return err
}
defer rows.Close()

for rows.Next() {
    var user User
    rows.Scan(&user.ID, &user.Name)
}
// rows.Err()のチェックを忘れている!
// クエリ途中のエラーを見逃す可能性
```

特に問題となるのは:
- **バッファサイズ超過**: `bufio.Scanner`のデフォルトバッファは64KBで、これを超える行があると`ErrTooLong`エラーが発生するが、`Err()`を呼ばないと検出できない
- **ネットワーク/I/Oエラー**: ファイル読み込み中やデータベースクエリ実行中のエラーが無視される
- **サブプロセスからの読み込み**: `os/exec`で起動したプロセスの出力を読む場合、長い行によるエラーはプロセス自体の失敗とは別に発生する

### 提案された解決策

`go vet`に新しいチェッカーを追加し、以下のパターンを検出します:

1. **対象型**: `bufio.Scanner`と`sql.Rows`（将来的に拡張可能）
2. **検出パターン**: `Scan()`や`Next()`のループ後に`Err()`メソッドが呼ばれていない場合
3. **偽陽性の削減**: 明らかにエラーが発生しない`strings.Reader`や`bytes.Reader`からの読み込みは除外

実装は`go/analysis/passes/scannererr`パッケージとして開発され、CL 730480で実装が進められています。

## これによって何ができるようになるか

開発者が無意識に見逃していたエラーハンドリングの抜け漏れを、`go vet`が自動的に検出してくれるようになります。これにより:

- **バグの早期発見**: テスト環境で問題を発見でき、本番環境での予期しないエラーを防げる
- **コードレビューの効率化**: レビュアーが毎回チェックする必要がなくなる
- **教育効果**: 初心者がこのパターンを学ぶ機会になる

### コード例

```go
// Before: エラーチェック漏れ（vetで検出される）
func readLines(filename string) []string {
    file, _ := os.Open(filename)
    defer file.Close()

    scanner := bufio.NewScanner(file)
    var lines []string
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    return lines
    // 問題: scanner.Err()をチェックしていない
    // → go vetが警告を出すようになる
}

// After: 正しいエラーハンドリング
func readLines(filename string) ([]string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    var lines []string
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }

    // 最終的なエラーチェックを追加
    if err := scanner.Err(); err != nil {
        return nil, err
    }

    return lines, nil
}
```

## 議論のハイライト

- **偽陽性問題の調査**: 2万モジュールでテスト実装を実行したところ、当初は2,337件の検出がありました。`strings.Reader`や`bytes.Reader`を除外した結果、1,258件に減少。その中から25件をサンプリングしたところ、真陽性率は72%（18/25）でした

- **「偽陽性」の再評価**: 当初偽陽性とされた7件は全て`os/exec`のパイプからの読み込みでしたが、Austin Clements氏は「サブプロセスが長い行を出力した場合、プロセス自体が失敗しなくてもScanner.Err()が発生する」と指摘。これらも実際にはチェックすべきという結論に

- **対象の絞り込み**: 全ての`Err() error`メソッドを持つ型を対象にするのは過剰との判断。`bufio.Scanner`と`sql.Rows`に限定する方向で議論が進行

- **vetの実行モード別対応の提案**: Robert Griesemer氏から、`*os.File`からの読み込みは常にチェックし、より攻撃的なチェック（全てのScanner/Rows）は明示的な`go vet`実行時のみ有効にするという提案が出されました（2026年1月28日）

- **実装の進捗**: Alan Donovan氏が実装を担当し、分析パッケージ`scannererr`を開発中。診断メッセージとドキュメントを明確化し、トークンサイズ超過のリスクについても説明する方針

## 関連リンク

- [Proposal Issue #17747](https://github.com/golang/go/issues/17747)
- [Review Minutes (2026-01-28)](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [実装CL 730480](https://go.dev/cl/730480)
- [bufio.Scanner公式ドキュメント](https://pkg.go.dev/bufio)
- [database/sql公式ドキュメント](https://pkg.go.dev/database/sql)

## Sources

以下の情報源を参照しました:

- [proposal: cmd/vet: check for missing Err calls for bufio.Scanner and sql.Rows · Issue #17747 · golang/go](https://github.com/golang/go/issues/17747)
- [bufio package - bufio - Go Packages](https://pkg.go.dev/bufio)
- [In-depth introduction to bufio.Scanner in Golang | by Michał Łowicki | golangspec | Medium](https://medium.com/golangspec/in-depth-introduction-to-bufio-scanner-in-golang-55483bb689b4)
- [Handling Errors (go-database-sql.org)](http://go-database-sql.org/errors.html)
- [sql package - database/sql - Go Packages](https://pkg.go.dev/database/sql)
- [Eliminate error handling by eliminating errors | Dave Cheney](https://dave.cheney.net/2019/01/27/eliminate-error-handling-by-eliminating-errors)

## 関連リンク

- [proposal: cmd/vet: check for missing Err calls for bufio.Scanner and sql.Rows · Issue #17747 · golang/go](https://github.com/golang/go/issues/17747)
- [Review Minutes (2026-01-28)](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
