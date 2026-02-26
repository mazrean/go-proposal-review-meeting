---
issue_number: 61902
title: "regexp: add iterator forms of matching methods"
previous_status: active
current_status: likely_accept
changed_at: 2026-02-25T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3962620065
related_issues:
  - title: "Proposal Issue #61902"
    url: https://github.com/golang/go/issues/61902
  - title: "Review Minutes (issuecomment-3962620065)"
    url: https://github.com/golang/go/issues/33502#issuecomment-3962620065
  - title: "Range over function proposal #61405"
    url: https://github.com/golang/go/issues/61405
  - title: "iter package proposal #61897"
    url: https://github.com/golang/go/issues/61897
---
## 概要

`regexp` パッケージの `FindAll*` 系メソッドに、スライスを返す代わりにイテレータを返す対応メソッド群を追加するproposalです。Go 1.22で導入された「range over function」機能（`iter.Seq`型）を活用し、大量のマッチ結果を一括でスライスに格納することなく、1件ずつ処理できるようにします。

## ステータス変更
**active** → **likely_accept**

2026年2月の週次Proposalレビューミーティングにて、`All*` イテレータAPIを採用する方向で強く傾いたと判断されました。議論の中でより高度な「Cursorベースの抽象型」APIも検討されましたが、v1に複数の設計思想を混在させると一貫性が失われるとして、v2以降への課題として先送りが決定されました。既存APIの自然な拡張として整合性が高い`All*`案が現実的とみなされ、`likely accept`となりました。

## 技術的背景

### 現状の問題点

`regexp`パッケージは現在16個の `FindAll*` 系メソッドを持ち、これらはすべてマッチ結果をスライスに格納して返します。大量のテキストを処理する場合、全マッチをスライスに蓄積してから処理することになり、メモリ消費が無駄に増えます（最悪ケースでは入力テキスト自体よりも多くのメモリが必要になる場合もあります）。

```go
// 現在: 全マッチを一度にスライスとして受け取る
matches := re.FindAllString(text, -1)
for _, m := range matches {
    process(m)
}
// textが数GBの場合、matchesスライス自体が大量のメモリを占有する
```

### 提案された解決策

`FindAll*` 系8メソッド各々に対応するイテレータ版メソッドを追加します。命名規則は「`Find`プレフィックスを取り除き、`All`を先頭に置く」です。また、`regexp.Regexp.Split`に対応する`SplitSeq`も追加されます。

追加されるメソッド一覧:

```go
func (re *Regexp) All(b []byte) iter.Seq[[]byte]
func (re *Regexp) AllIndex(b []byte) iter.Seq[[]int]
func (re *Regexp) AllString(s string) iter.Seq[string]
func (re *Regexp) AllStringIndex(s string) iter.Seq[[]int]
func (re *Regexp) AllStringSubmatch(s string) iter.Seq[[]string]
func (re *Regexp) AllStringSubmatchIndex(s string) iter.Seq[[]int]
func (re *Regexp) AllSubmatch(b []byte) iter.Seq[[][]byte]
func (re *Regexp) AllSubmatchIndex(b []byte) iter.Seq[[]int]
func (re *Regexp) SplitSeq(s string) iter.Seq[string]
```

既存の `FindAll*` メソッドは維持されます（後方互換性あり）。

## これによって何ができるようになるか

Go 1.22以降の `range over function` 構文と組み合わせて、メモリ効率の良いマッチ処理が可能になります。

### コード例

```go
// Before: 全マッチをスライスに格納してからループ処理
re := regexp.MustCompile(`\b\w+@\w+\.\w+\b`)
matches := re.FindAllString(emailLog, -1)
for _, email := range matches {
    sendNotification(email)
}

// After: イテレータで1件ずつ処理（スライス割り当て不要）
re := regexp.MustCompile(`\b\w+@\w+\.\w+\b`)
for email := range re.AllString(emailLog) {
    sendNotification(email)
}
```

```go
// SplitSeq: 区切り文字で分割した各部分を順番に処理
re := regexp.MustCompile(`\s+`)
for word := range re.SplitSeq(text) {
    index(word)
}

// Before (Split): 全部分を一度にスライスとして取得
for _, word := range re.Split(text, -1) {
    index(word)
}
```

```go
// 早期終了も自然に書ける（最初のマッチだけ取得）
re := regexp.MustCompile(`error: (.+)`)
for m := range re.AllStringSubmatch(log) {
    fmt.Println("First error:", m[1])
    break  // 以降のスキャンはスキップされる
}
```

## 議論のハイライト

- **命名論争**: `Iter`を使う案（`FindIterString`等）も提案されたが、Go標準ライブラリでは`strings.SplitSeq`等で`All`がイテレータを意味する規約として確立されているとして却下。「`Find`プレフィックスを除く」命名が採用される方向。
- **新型`Cursor`/`Substring`の提案**: `adonovan`氏が`iter.Seq[Substring]`を返す抽象型ベースAPIを提案し、長期にわたり議論された。インデックス・文字列・バイト列を統一的に返せる利点があるが、v1に2つの設計思想が混在する問題がある。最終的に「v2の課題」として先送り決定。
- **`RegexpIter`型の新設案**: メソッド数が膨大になることを懸念する意見もあったが、型を新設しても合計メソッド数は変わらず、むしろ学習コストが増えるとして却下。
- **メモリ効率**: スライスの再利用（イテレーション間で`[]byte`等を使い回す）の可能性も議論されたが、安全性とシンプルさを優先し、各イテレーションで新規割り当てとする方向。
- **前提条件の充足**: このproposalは「range over function」（#61405）の承認を前提としていたが、同提案はGo 1.22で受理・実装済みであり、ブロッカーが解消された。

## 関連リンク

- [Proposal Issue #61902](https://github.com/golang/go/issues/61902)
- [Review Minutes (issuecomment-3962620065)](https://github.com/golang/go/issues/33502#issuecomment-3962620065)
- [Range over function proposal #61405](https://github.com/golang/go/issues/61405)
- [iter package proposal #61897](https://github.com/golang/go/issues/61897)
