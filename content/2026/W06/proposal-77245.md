---
issue_number: 77245
title: "spec: function type inference should work in all assignment contexts"
previous_status: 
current_status: active
changed_at: 2026-02-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3872311559
related_issues:
  - title: "関連Issue: #12854 - 代入コンテキストの定義"
    url: https://github.com/golang/go/issues/12854
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77245
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3872311559
  - title: "関連Issue: #59338 - reverse type inference提案"
    url: https://github.com/golang/go/issues/59338
---
## 概要

ジェネリック関数の型推論を、すべての代入コンテキスト（assignment context）で機能するように拡張する提案です。現在、Go 1.21で導入された型推論は一部の代入では機能しますが、複合リテラル内の暗黙的な代入やチャネル送信では機能しない不整合があります。

## ステータス変更

**(new)** → **active**

この提案は2026年2月9日のProposal Review Meetingで議論され、activeステータスとなりました。これは、仕様の明確化が必要な言語変更であるため、正式な提案プロセスを経る必要があると判断されたためです。

## 技術的背景

### 現状の問題点

Go 1.21では、ジェネリック関数の型推論が大幅に改善され、変数への代入時に型引数を省略できるようになりました。しかし、この型推論は**すべての代入で統一的に機能していません**。

```go
type S struct{ f func(int) }
func g[T any](T) {}

func _(s S) {
    s.f = g          // OK: 通常の代入では型推論が機能
    s = S{f: g}      // エラー: 複合リテラル内では型推論が機能しない
    s = S{f: g[int]} // OK: 明示的な型引数指定が必要
}
```

同様の不整合は配列、スライス、マップの複合リテラルでも発生します。

```go
type F func(int)
type A [10]F
func g[T any](T) {}

var a A
a[0] = g      // OK
a = A{g}      // エラー: 複合リテラルでは型推論が機能しない
a = A{g[int]} // OK
```

### 提案された解決策

提案では、仕様における「代入コンテキスト（assignment context）」を定義し、そのすべてのケースで型推論を機能させることを目指しています。代入コンテキストには以下が含まれます。

- 通常の代入文（`x = y`）
- 複合リテラル内の要素（`S{f: g}`）
- 関数の戻り値
- 関数引数の受け渡し
- チャネル送信（`ch <- g`）
- 型変換（`T(x)`で`x`が`T`に代入可能な場合）

## これによって何ができるようになるか

ジェネリック関数を使用する際のコードがより簡潔になり、一貫性が向上します。特に、複合リテラルを多用するコード（protocol buffersの構築など）で、冗長な型引数の明示が不要になります。

### コード例

```go
type F func(int)
type M map[string]F
func g[T any](T) {}

// Before: 複合リテラルでは明示的な型引数が必要
m := M{"foo": g[int]}

// After: 型推論により型引数を省略可能
m := M{"foo": g}

// チャネル送信の例
type C chan F
var c C

// Before: 明示的な型引数が必要
c <- g[int]

// After: 型推論により型引数を省略可能
c <- g
```

## 議論のハイライト

- **仕様の明示性**: 当初はバグと考えられていましたが、Go仕様の[Instantiations](https://golang.org/ref/spec#Instantiations)セクションが型推論の適用範囲を明確に定義しているため、正式な仕様変更が必要な言語提案となりました。
- **実装の進行**: CL 737800（後にクローズ）とCL 739560で実装が進められており、Robert Griesemer氏が積極的に関与しています。
- **代入コンテキストの定義**: #12854での議論を参照し、「代入コンテキスト」の正確な定義が検討されています。
- **型変換の追加**: 議論中に、型変換`T(x)`も代入コンテキストに含めるべきという指摘がありました（`x`が`T`に代入可能な場合）。
- **一貫性の向上**: コミュニティからは、現在の挙動（`s.f = g`はOKだが`s = S{f: g}`がエラー）は直感的でなく、一貫性を持たせるべきという声が多く上がっています。

## 関連リンク

- [関連Issue: #12854 - 代入コンテキストの定義](https://github.com/golang/go/issues/12854)
- [Proposal Issue](https://github.com/golang/go/issues/77245)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3872311559)
- [関連Issue: #59338 - reverse type inference提案](https://github.com/golang/go/issues/59338)
