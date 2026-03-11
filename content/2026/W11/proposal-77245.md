---
issue_number: 77245
title: "spec: function type inference should work in all assignment contexts"
previous_status: active
current_status: likely_accept
changed_at: 2026-03-11T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4042167102
related_issues:
  - title: "Proposal Issue #77245"
    url: https://github.com/golang/go/issues/77245
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4042167102
  - title: "関連Issue #12854: spec: type inferred composite literals"
    url: https://github.com/golang/go/issues/12854
  - title: "関連Issue #59338: spec: infer type arguments from assignments of generic functions"
    url: https://github.com/golang/go/issues/59338
---
## 概要

Go言語の型推論機能において、ジェネリック関数値の代入が一部のコンテキストで機能しない不整合を修正するため、仕様（spec）における「代入可能性（assignability）」の定義を拡張するproposalです。すべての代入コンテキストで関数型推論が一貫して動作するようにすることを目的としています。

## ステータス変更
**active** → **likely_accept**

2026年3月11日の週次proposalレビュー会議において、`aclements`がlikely acceptと判定しました。griesemerが提示した「代入可能性の定義を拡張する」というアプローチが、あらゆる代入コンテキストを自動的にカバーする最もエレガントな解決策であると評価されたためです。具体的な仕様変更案はCL 751312に記述されています。

## 技術的背景

### 現状の問題点

Go 1.21でジェネリック関数の型推論が大幅に改善され、変数への代入や関数の戻り値としてジェネリック関数を使用する際に型引数を省略できるようになりました。しかし、この推論はすべての代入コンテキストに対して一貫して適用されているわけではありません。

具体的には、構造体フィールドへの直接代入は動作する一方で、複合リテラル（composite literal）内での同等の代入は失敗します。

```go
type S struct{ f func(int) }

func g[T any](T) {}

func _(s S) {
    s.f = g          // ok: 代入文では型推論が機能する
    s = S{f: g}      // error: 複合リテラルでは機能しない
    s = S{f: g[int]} // ok: 明示的なインスタンス化は必要
}
```

同様の問題は配列・スライス・マップの複合リテラルやチャネルへの送信でも発生します。

```go
type F func(int)
type S []F

func g[T any](T) {}

func _() {
    var s S
    s[0] = g   // ok: インデックス代入では機能する
    s = S{g}   // error: スライスリテラルでは機能しない

    var c chan F
    c <- g      // error: チャネル送信でも機能しない
}
```

LHS（左辺）の型が完全に確定しているためジェネリック型引数を明確に推論できるにもかかわらず、コンテキストによって動作が異なるという不整合が存在します。

### 提案された解決策

当初、「代入コンテキスト（assignment context）」という概念を新たに仕様に導入することが検討されました。しかし、griesemerの調査により、仕様中で `x` が代入コンテキストに置かれる箇所は、既に「代入可能性（assignability）」として定義されていることが判明しました。

そのため、**代入可能性の定義そのものに、部分的またはインスタンス化されていないジェネリック関数の代入に関するルールを追加する**というアプローチが採用されました。この変更により、代入可能性が要求されるすべての箇所（複合リテラル、チャネル送信、型変換など）が自動的にカバーされます。

## これによって何ができるようになるか

この変更により、型が明確に推論できる場面では、ジェネリック関数の明示的なインスタンス化（`g[int]`のような記述）を省略できるようになります。

```go
// Before: 明示的なインスタンス化が必要
type F func(int)
func g[T any](T) {}

s := []F{g[int]}               // 型引数を明示
m := map[string]F{"key": g[int]} // 型引数を明示
var c chan F
c <- g[int]                    // 型引数を明示

// After: 型推論が自動的に機能する
s := []F{g}                    // 型推論で int が推論される
m := map[string]F{"key": g}    // 型推論で int が推論される
var c chan F
c <- g                         // 型推論で int が推論される
```

**主なユースケース:**

1. **関数テーブルの構築**: 関数ポインタのスライスやマップを使ったディスパッチテーブルを作成する際に冗長な型引数を省略できる
2. **コールバックの登録**: イベントハンドラなど関数型が明確なフィールドへのジェネリック関数の設定がより簡潔になる
3. **チャネルを使った並行処理**: ワーカーパターンなどでジェネリック関数を型付きチャネルに渡す際にコードが読みやすくなる

## 議論のハイライト

- **当初はバグとして報告**: proposalの提出者（griesemer自身）も最初はこれをバグと考えていました。代入文（`s[0] = g`）では動作するにもかかわらず複合リテラル（`S{g}`）では動作しないことが非常に不整合に見えるためです。しかし、現行仕様はtype inferenceが機能する箇所を明示的に列挙しており、現在の動作は仕様に準拠していることが判明し、言語変更のproposalとして再定義されました。
- **アプローチの転換**: 「代入コンテキスト」を新概念として導入する案から、既存の「代入可能性」定義を拡張する案へと変更されました。後者のほうが仕様全体との整合性が高く、変更箇所を最小化できます。
- **go/types APIへの影響**: `aclements`は、代入コンテキストの概念が確立された場合、`go/types` APIにも統一的に公開することが有益である可能性を指摘しました。
- **関連する#12854との連携**: 代入可能性の拡張は、型推論の関連issue #12854（仕様: 型推論済み複合リテラル）にも適用できる可能性があり、より広範なスペック簡略化につながるかもしれません。
- **実装の早期提案**: コミュニティメンバー（@next-n）がissueオープンから12時間以内にPR #77247を提出しましたが、最終的にはより根本的な仕様レベルでの解決策（CL 739560、CL 751312）が選択されました。

## 関連リンク

- [Proposal Issue #77245](https://github.com/golang/go/issues/77245)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4042167102)
- [関連Issue #12854: spec: type inferred composite literals](https://github.com/golang/go/issues/12854)
- [関連Issue #59338: spec: infer type arguments from assignments of generic functions](https://github.com/golang/go/issues/59338)
