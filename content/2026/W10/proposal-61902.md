---
issue_number: 61902
title: "regexp: add iterator forms of matching methods"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-03-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
related_issues:
  - title: "range over function proposal #61405"
    url: https://github.com/golang/go/issues/61405
  - title: "related proposals list #61897"
    url: https://github.com/golang/go/issues/61897
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/61902
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
---
## 概要

`regexp` パッケージの正規表現マッチメソッドにイテレータ形式を追加する提案です。既存の `FindAll*` 系メソッドがマッチ結果を全てスライスに詰めて返すのに対し、新たに `All*` 系メソッドを追加することで `iter.Seq` 形式のイテレータとして逐次処理できるようにします。また、`SplitSeq` メソッドも追加されます。

## ステータス変更

**likely_accept** → **accepted**

Proposal Review Meeting での議論の結果、v1 API の延長線上で既存の `FindAll*` に対応する `All*` 系イテレータメソッドを追加する当初の提案が最もクリーンな解決策と判断されました。カーソル型（`Cursor`/`Substring`）を用いた新 API はより表現力が高いものの、v1 に導入すると「2つの異なる設計思想が混在する」という問題が生じるため、v2 での対応に持ち越すこととし、v1 では当初提案を受け入れることで合意しました。

## 技術的背景

### 現状の問題点

`regexp.Regexp` の `FindAll*` メソッド群（`FindAllString`、`FindAllStringSubmatch` など）は、全マッチ結果をスライスとして一括返却します。大きなテキストを処理する際、実際には1件ずつ処理したい場合でも全マッチのスライスがメモリ上に確保されます。また、途中でイテレーションを中断したいケースにも対応できません。

```go
// 現在: 全マッチをスライスとして一括取得（大きなテキストではメモリ浪費）
matches := re.FindAllString(text, -1)
for _, m := range matches {
    process(m)
}
```

### 提案された解決策

`FindAll*` 8メソッドそれぞれに対し、`Find` プレフィックスを除いた `All*` 系イテレータメソッドを追加します。命名規則は既存の `Find(All)?(String)?(Submatch)?(Index)?` パターンを踏まえ、`(All|FindAll)?(String)?(Submatch)?(Index)?` に拡張されます。

追加される新メソッド一覧:

```go
func (re *Regexp) All(b []byte) iter.Seq[[]byte]
func (re *Regexp) AllIndex(b []byte) iter.Seq[[]int]
func (re *Regexp) AllString(s string) iter.Seq[string]
func (re *Regexp) AllStringIndex(s string) iter.Seq[[]int]
func (re *Regexp) AllStringSubmatch(s string) iter.Seq[[]string]
func (re *Regexp) AllStringSubmatchIndex(s string) iter.Seq[[]int]
func (re *Regexp) AllSubmatch(b []byte) iter.Seq[[][]byte]
func (re *Regexp) AllSubmatchIndex(b []byte) iter.Seq[[]int]

// strings.SplitSeq に対応する新メソッド
func (re *Regexp) SplitSeq(s string) iter.Seq[string]
```

## これによって何ができるようになるか

大量のテキストを正規表現でスキャンする際に、全結果をメモリに積み込まずに逐次処理できるようになります。また `range` ループと `break` を組み合わせることで、条件を満たした最初のマッチで処理を打ち切ることも簡単になります。

### コード例

```go
// Before: 全マッチを一括取得してからループ処理
re := regexp.MustCompile(`\b\w+@\w+\.\w+\b`)
matches := re.FindAllString(largeText, -1)
for _, m := range matches {
    if isSpam(m) {
        flagEmail(m)
    }
}

// After: イテレータで逐次処理（全スライスの事前確保不要）
re := regexp.MustCompile(`\b\w+@\w+\.\w+\b`)
for m := range re.AllString(largeText) {
    if isSpam(m) {
        flagEmail(m)
    }
}

// After: 最初のマッチだけ使いたい場合も break で効率的に
for m := range re.AllString(largeText) {
    process(m)
    break
}

// After: SplitSeq でセパレータ区切りの文字列を逐次処理
re := regexp.MustCompile(`\s*,\s*`)
for field := range re.SplitSeq(csvLine) {
    handleField(field)
}
```

## 議論のハイライト

- **命名: `All*` vs `FindIter*` vs `Each*`**: `Iter` や `Each` を名前に含める案があったが、`strings.SplitSeq` など標準ライブラリで確立した「`All` = イテレータ」の慣例に合わせ `All*` が採用された。`FindAll*` の `Find` を落とすことでイテレータであることを表現している。

- **新型 `Cursor`/`Substring` による統合 API**: @adonovan が `iter.Seq[Substring]` 型を返す抽象マッチ型を用いた新 API（CL #643896）を提案。これはより表現力が高く、文字列/バイトスライスを意識せず扱える。しかし、v1 に導入すると既存の `FindAll*` と「2つの設計思想が混在」し一貫性が失われることから v2 での対応が妥当と判断された。

- **`RegexpIter` 型を別途定義する案**: メソッド数増加を懸念する声があったが、@rsc が「別型を作っても全体のメソッド数は減らない」と指摘し棄却。現在 40 メソッドに今回の 8 メソッドを加えて 48 メソッドとなる。

- **スライス再利用とメモリ効率**: イテレーション間でスライスを再利用すべきか否かの議論があった。安全性を優先し現時点では再利用しない方針となっているが、コンパイラが将来的にアロケーションを最適化できる可能性がある。

- **実装 CL**: @rsc が CL #742801（`regexp: reimplement API using iterators, revise doc comments`）として実装を準備。新メソッドの追加と合わせて、既存メソッドのドキュメントも大幅に改善される。

## 関連リンク

- [range over function proposal #61405](https://github.com/golang/go/issues/61405)
- [related proposals list #61897](https://github.com/golang/go/issues/61897)
- [Proposal Issue](https://github.com/golang/go/issues/61902)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4000757564)
