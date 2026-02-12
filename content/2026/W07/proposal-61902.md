---
issue_number: 61902
title: "regexp: add iterator forms of matching methods"
previous_status: hold
current_status: active
changed_at: 2026-02-11T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3886687081
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/61902
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3886687081
  - title: "関連Issue #61405: range over function"
    url: https://github.com/golang/go/issues/61405
  - title: "関連Issue #61897: iter パッケージ"
    url: https://github.com/golang/go/issues/61897
---
## 概要

`regexp`パッケージに、既存の`FindAll*`メソッド群（すべてのマッチを一度にスライスで返すメソッド）のイテレータ版を追加する提案です。イテレータ形式により、大きなテキスト検索時にすべてのマッチを保持する必要がなくなり、メモリ効率が大幅に向上します。

## ステータス変更
**hold** → **active**

この提案は2023年8月に提出され、Go 1.23で追加された「range over function」機能（#61405）の実装完了を待って保留されていました。2026年2月にRuss Coxが実装CLを提出したことで、再びアクティブになりました。議論では新しいCursor型を使った全く新しいAPIも検討されましたが、既存のv1 APIとの一貫性を保つため、当初の提案に沿った形で進められています。

## 技術的背景

### 現状の問題点

`regexp`パッケージには「すべてのマッチを見つける」ための`FindAll*`メソッド群が8つ存在します（`FindAllString`、`FindAllSubmatchIndex`など）。これらは結果をすべて`[]string`や`[][]byte`などのスライスとして返すため、以下の問題があります。

```go
// 大きなテキストから正規表現マッチを全て取得
text := loadLargeDocument() // 数MB〜数十MBのテキスト
re := regexp.MustCompile(`pattern`)
matches := re.FindAllString(text, -1) // すべてのマッチをスライスに格納

// 問題: マッチ数が多い場合、matchesスライスの
// メモリ使用量が入力テキストよりも大きくなる可能性がある
```

特に大きなテキストで多数のマッチを検索する場合、マッチ結果のスライスがテキストサイズを超えることもあり、無駄なメモリ割り当てが発生します。実際には、多くのケースでマッチを一つずつ処理するだけで十分です。

### 提案された解決策

既存の`FindAll*`メソッドから`Find`プレフィックスを除去した新しいイテレータメソッド群を追加します。これらは`iter.Seq`または`iter.Seq2`型を返し、range文で直接ループできます。

**追加される8つのメソッド:**

```go
func (re *Regexp) All(b []byte) iter.Seq[[]byte]
func (re *Regexp) AllIndex(b []byte) iter.Seq[[]int]
func (re *Regexp) AllString(s string) iter.Seq[string]
func (re *Regexp) AllStringIndex(s string) iter.Seq[[]int]
func (re *Regexp) AllStringSubmatch(s string) iter.Seq[[]string]
func (re *Regexp) AllStringSubmatchIndex(s string) iter.Seq[[]int]
func (re *Regexp) AllSubmatch(b []byte) iter.Seq[[][]byte]
func (re *Regexp) AllSubmatchIndex(b []byte) iter.Seq[[]int]
```

さらに、`strings.SplitSeq`との類推で、`SplitSeq`メソッドも追加されます。

```go
func (re *Regexp) SplitSeq(s string) iter.Seq[string]
```

## これによって何ができるようになるか

大量のテキストを正規表現で処理する際、必要な分だけマッチを取得し、途中で処理を中断できるようになります。ログ解析、大規模テキストマイニング、ストリーム処理などで特に有用です。

### コード例

```go
// Before: 従来の書き方（すべてのマッチをスライスに格納）
re := regexp.MustCompile(`\d+`)
text := loadLargeLog() // 巨大なログファイル
allMatches := re.FindAllString(text, -1) // メモリに全マッチを格納
for _, match := range allMatches {
    if processMatch(match) {
        break // すでにスライスは全割り当て済み
    }
}

// After: 新しいイテレータAPI（必要な分だけ処理）
re := regexp.MustCompile(`\d+`)
text := loadLargeLog()
for match := range re.AllString(text) {
    if processMatch(match) {
        break // ここで正規表現エンジンも停止、メモリ節約
    }
}
```

**メリット:**
- メモリ割り当てが大幅に削減される
- 途中で処理を中断した場合、その後のマッチング処理も中断される
- 既存の`FindAll*`メソッドと並行して使用可能（後方互換性あり）

## 議論のハイライト

- **命名規則について**: 「Iter」や「Each」を使うべきという提案もありましたが、標準ライブラリ全体で「All」をイテレータの規約として確立するため、当初の提案通り「All」を採用
- **新しいCursor/Match型の導入**: Alan Donovanを中心に、`regexp.Substring`や`regexp.Match`といった抽象型を返す全く新しいAPIの議論がありました。この方式では、マッチ結果から`.String()`、`.Bytes()`、`.Submatch(i)`などのメソッドで情報を取得できるため、16〜24個ものメソッドを覚える必要がなくなります。この案は技術的に優れていますが、v2での採用が検討されており、v1では既存APIとの一貫性を優先
- **実用性の疑問**: 一部から「regexpの結果数はそこまで多くないのでは」という指摘もありましたが、Russ Coxは「大きなテキストでは結果スライスが入力より大きくなり得る」と反論
- **regexp/v2の可能性**: `math/rand/v2`のように、regexpもv2パッケージとしてシンプルなAPIで再設計すべきという意見も出ましたが、既存のv1で段階的に改善する方針

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/61902)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3886687081)
- [関連Issue #61405: range over function](https://github.com/golang/go/issues/61405)
- [関連Issue #61897: iter パッケージ](https://github.com/golang/go/issues/61897)
