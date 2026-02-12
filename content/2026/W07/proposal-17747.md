---
issue_number: 17747
title: "cmd/vet: check for missing Err calls for bufio.Scanner and sql.Rows"
previous_status: active
current_status: likely_accept
changed_at: 2026-02-11T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3886687081
related_issues:
  - title: "Proposal Issue #17747"
    url: https://github.com/golang/go/issues/17747
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3886687081
---
## 概要

`go vet`に`bufio.Scanner`や`sql.Rows`の`Err()`メソッドチェック漏れを検出する機能を追加する提案です。これらの型を使用した際に`Err()`が呼ばれていない場合に警告を出すことで、エラー処理の見落としによる潜在的なバグを防ぎます。

## ステータス変更

**active** → **likely_accept**

2026年2月11日のProposal Review Meetingで**likely accept**（承認見込み）となり、最終コメント期間（Final Comment Period）に入りました。約8年間の議論を経て、実装が作成され（CL 730480）、22,000モジュールでの大規模テストで有効性が実証されたことが決定の背景にあります。

## 技術的背景

### 現状の問題点

`bufio.Scanner`と`sql.Rows`は、エラーチェックに特殊なパターンを採用しています。繰り返し処理の終了後に`Err()`メソッドを呼び出さないと、I/Oエラーやトークンバッファオーバーフロー（デフォルト64KB制限）などの重要なエラーを見落とす可能性があります。

```go
// 問題のあるコード例: Err()チェックなし
scanner := bufio.NewScanner(file)
for scanner.Scan() {
    line := scanner.Text()
    // lineを処理
}
// scanner.Err()を呼び出していない！エラーが発生していても気づかない
```

`sql.Rows`も同様の問題があります。`rows.Next()`は「次の行がない」場合と「エラーが発生した」場合の両方で`false`を返すため、ループ後に`rows.Err()`を呼ばないとエラーを検出できません。

### 提案された解決策

新しい`go vet`チェッカーは以下を検出します：

- `bufio.Scanner`を使用するループの後で`Err()`が呼び出されていない
- 偽陽性（false positive）を減らすため、`bytes.Buffer`や`strings.Reader`など「失敗しない」ことが保証されている入力源は除外
- ただし`*os.File`からの読み取りは、I/Oエラーは稀でもトークンバッファオーバーフローの可能性があるためチェック対象

このチェックは`go test`実行時には含まれず、明示的な`go vet`実行時のみ有効になる予定です（false positiveが約28%あったため）。また、新しいGoバージョンで`go.mod`の`go`ディレクティブに基づいてゲート制御される方針です。

## これによって何ができるようになるか

開発者は`bufio.Scanner`や`sql.Rows`のエラー処理漏れをコードレビュー前に発見できます。実際の調査では、あるコードベースで使用箇所の約10%でこのエラーが発生していたことが報告されています。

### コード例

```go
// Before: エラーチェック漏れ
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
    // ファイルI/Oエラーや長い行によるバッファオーバーフローを見逃す
}

// After: go vetが警告を出すため修正を促される
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
    if err := scanner.Err(); err != nil {
        return nil, err  // I/Oエラーやバッファオーバーフローを適切に検出
    }
    return lines, nil
}
```

## 議論のハイライト

- **false positive問題**: 当初から最大の懸念事項。子プロセスのパイプから読む場合など、`Err()`チェックが不要に見えるケースが約28%存在したが、トークンバッファオーバーフローは入力源に関係なく起こりうるため、結局全てのケースでチェックすべきとの結論に
- **8年越しの決定**: 2016年の提起から2026年まで、慎重な証拠収集と議論が続けられた。2024年に実装が作成され、実際に22,000モジュールでテストして1,162モジュールで2,337件の問題を発見
- **`Close`との違い**: `Err()`メソッドと`Close()`メソッドの使い分けについて議論。`Close`は「呼ばなければならない」、`Err`は「呼べる」という異なる意味を持つべきとの意見があったが、実際には`Err()`も必須であると結論
- **`go test`から除外**: false positiveの割合（28%）がgovetの通常基準（5%以下）を大きく超えるため、`go test`実行時には含めず、明示的な`go vet`実行時のみ有効にする方針
- **バージョンゲート**: 新しい`go vet`チェックは`go.mod`の`go`ディレクティブでバージョン制御され、既存コードベースへの影響を最小化

## 関連リンク

- [Proposal Issue #17747](https://github.com/golang/go/issues/17747)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3886687081)
