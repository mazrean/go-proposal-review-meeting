---
issue_number: 17747
title: "cmd/vet: check for missing Err calls for bufio.Scanner and sql.Rows"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-02-18T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3923200976
related_issues:
  - title: "関連Issue #71485: printf checkerのバージョンゲート（先行事例）"
    url: https://github.com/golang/go/issues/71485
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/17747
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3923200976
  - title: "関連Issue #51299: Scanner finalizer によるランタイム警告の代替提案"
    url: https://github.com/golang/go/issues/51299
  - title: "関連Issue #75440: 新しいvetチェックをGoバージョンにゲートする仕組みについて"
    url: https://github.com/golang/go/issues/75440
---
## 概要

`go vet` に `bufio.Scanner` および `sql.Rows` の使用後に `.Err()` メソッドが呼ばれていない箇所を検出する静的解析チェックを追加する提案です。イテレーション終了後のエラー確認漏れというよくある実装ミスを自動検出できるようにします。

## ステータス変更

**likely_accept** → **accepted**

2026年2月11日のProposal Review Meetingで "likely accept" となり、翌週2月18日の会議（@aclements, @cherrymui, @griesemer, @ianlancetaylor, @neild, @rolandshoemaker 参加）で合意に変化がないとして正式に **accepted** となりました。実装CL（[go.dev/cl/730480](https://go.dev/cl/730480)）がすでに存在しており、約20,000モジュールを対象としたコーパス調査により偽陽性率が許容範囲内であることが確認されたことが最終承認の決め手となりました。

## 技術的背景

### 現状の問題点

`bufio.Scanner` は `for scanner.Scan() { ... }` というループ構文で使われますが、`Scan()` がループを抜けた理由は「正常なEOF」か「エラー発生」かの二通りあります。エラーを知るには必ず `scanner.Err()` を呼ぶ必要がありますが、現在の `go vet` はこれを検出しません。

重要な落とし穴として、`bufio.Scanner` は渡される `io.Reader` の種類に関係なく `ErrTooLong`（1トークンがバッファサイズを超えた場合）というエラーが発生しえます。つまり、`strings.Reader` や `bytes.Buffer` のような「失敗しない」ソースから読んでいる場合でも、長すぎる行を受け取ると黙ってエラーになります。この事実を多くの開発者が認識しておらず、実際にコーパス調査では約18/25件が真の陽性（本物のバグ）でした。

同様のパターンは `sql.Rows` にも存在します。`rows.Next()` のループ後に `rows.Err()` を確認しない場合、クエリ中に発生したエラーを見逃すことになります。

```go
// よくある間違いの例
scanner := bufio.NewScanner(file)
for scanner.Scan() {
    fmt.Println(scanner.Text())
}
// ここで scanner.Err() を確認していない！
// ErrTooLong などのエラーが黙って握りつぶされる
```

### 提案された解決策

`go/analysis/passes/scannererr` という新しい解析パスを `cmd/vet` に追加します。

- `bufio.Scanner` を `Scan()` ループで使用した後に `Err()` が呼ばれていない場合に警告を発する
- `io.Reader` が `bytes.Buffer` や `strings.Reader` など「失敗しないことが既知」のものである場合は警告を除外する
- `go test` 実行時のベット検査（自動実行分）には含めず、`go vet` を明示的に実行した場合にのみ検出する（既存コードへの影響を抑えるため）
- 新しい Go バージョン指定（go.mod の `go` ディレクティブ）に依存させることも検討されている（#75440）

## これによって何ができるようになるか

`go vet` を実行するだけで、エラーチェック漏れのイテレーションパターンを自動検出できます。コードレビューで繰り返し指摘されていたこの種のミスが、ツールによって機械的に検出されるようになります。

### コード例

```go
// Before: よくある誤った書き方（go vet が警告するようになる）
func readLines(r io.Reader) []string {
    var lines []string
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    // scanner.Err() を確認していない → ErrTooLong などを見逃す
    return lines
}

// After: 正しい書き方
func readLines(r io.Reader) ([]string, error) {
    var lines []string
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    if err := scanner.Err(); err != nil {
        return nil, err
    }
    return lines, nil
}

// エラーを意図的に無視する場合（明示的に示す）
scanner := bufio.NewScanner(r)
for scanner.Scan() { /* ... */ }
_ = scanner.Err() // 意図的に無視
```

## 議論のハイライト

- **長年の懸案事項**: この提案は2016年11月に最初に報告されたにもかかわらず、約9年間「likely accept」止まりでした。実装のCL（[go.dev/cl/730480](https://go.dev/cl/730480)）が投稿され、大規模コーパス調査で偽陽性率が定量化されたことで、2026年2月にようやく正式承認に至りました。

- **偽陽性の問題と調査**: 約20,000モジュールの調査では最初2,337件の指摘が出ましたが、`bytes.Buffer` と `strings.Reader` を除外することで1,258件に絞り込まれました。無作為サンプル25件の確認では18件が真の陽性（本物のバグ）、7件が偽陽性（子プロセスのpipeからの読み取りなど）でした。

- **ErrTooLong の重要性**: 「infallible reader（失敗しないReader）」からスキャンする場合でも、トークンが大きすぎる場合には `ErrTooLong` が発生します。これは `bufio.Scanner` 固有のエラーパターンであり、`Err()` チェックをいかなる場合も省略してよいわけではないという重要な事実が、議論を通じて広く認識されました。

- **go test との分離**: 既存コードへの影響を最小化するため、`go test` 時の自動vet実行には含めず、`go vet` の明示的な実行時のみ検出する方針が採用されました。また、go.mod のGoバージョンにゲートする仕組みの導入も検討されています。

- **スコープの絞り込み**: 「`Err() error` メソッドを持つ全ての型」に一般化する案もありましたが、偽陽性が増えすぎるとして却下され、`bufio.Scanner` のみを対象とする（`sql.Rows` は当初の提案タイトルにあったものの最終的にスコープ外）より絞り込まれた検査として承認されました。

## 関連リンク

- [関連Issue #71485: printf checkerのバージョンゲート（先行事例）](https://github.com/golang/go/issues/71485)
- [Proposal Issue](https://github.com/golang/go/issues/17747)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3923200976)
- [関連Issue #51299: Scanner finalizer によるランタイム警告の代替提案](https://github.com/golang/go/issues/51299)
- [関連Issue #75440: 新しいvetチェックをGoバージョンにゲートする仕組みについて](https://github.com/golang/go/issues/75440)
