---
issue_number: 76361
title: "x/tools/go/ast/inspector: add (\\*Cursor).Valid() bool method"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "関連Issue #70859 - Cursor型の導入提案"
    url: https://github.com/golang/go/issues/70859
  - title: "関連Issue #77195 - ParentEdge系メソッドの追加提案"
    url: https://github.com/golang/go/issues/77195
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/76361
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/76361#issuecomment-3814234826
---

## 要約

## 概要

`inspector.Cursor`はGo言語のAST（抽象構文木）を効率的に走査するための型ですが、カーソルが有効かどうかを判定する方法が不明瞭でした。このproposalは、カーソルの有効性を直接確認できる`Valid()`メソッドを追加することで、コードをより明確で読みやすくすることを目的としています。

## ステータス変更
**likely_accept** → **accepted**

2026年1月21日に「likely accept」とされ、最終的に2026年1月28日に正式承認されました。議論の過程で反対意見がなく、提案の明確な有用性とシンプルさが評価されました。proposal review groupは「コンセンサスに変更なし」として承認を決定し、実装フェーズに移行しています（CL 738821で実装中）。

## 技術的背景

### 現状の問題点

`inspector.Cursor`を使用する際、カーソルが有効かどうかを確認する必要がある場面が頻繁にあります。現在は以下のような回りくどい方法を使う必要があります。

```go
// 方法1: ゼロ値との比較（冗長）
if cur != (Cursor{}) {
    use(cur)
}

// 方法2: Inspectorフィールドのnil チェック（不明瞭）
if cur.Inspector() != nil {
    use(cur)
}
```

特に2つ目の方法は、`Inspector()`が何を表すのか、なぜnilチェックが有効性判定になるのかが直感的に理解しにくく、コードの可読性を損ないます。

### 提案された解決策

`Cursor`型に`Valid()`メソッドを追加し、カーソルの有効性を明示的に報告できるようにします。

```go
package inspector // golang.org/x/tools/go/ast/inspector

type Cursor ...

// Valid reports whether the cursor is valid.
func (Cursor) Valid() bool
```

## これによって何ができるようになるか

カーソルの有効性チェックがより直感的で読みやすくなります。特に、`Cursor`の多くのナビゲーションメソッド（`FindByPos`, `LastChild`, `PrevSibling`, `NextSibling`など）は見つからない場合にゼロ値のカーソルを返すため、その後の処理で有効性確認が必須です。`Valid()`メソッドにより、このパターンがシンプルになります。

### コード例

```go
// Before: 従来の書き方（回りくどい or 不明瞭）
cursor := parent.FindByPos(start, end)
if cursor != (Cursor{}) {
    process(cursor)
}
// または
if cursor.Inspector() != nil {
    process(cursor)
}

// After: Valid()メソッドを使った書き方（明確で読みやすい）
cursor := parent.FindByPos(start, end)
if cursor.Valid() {
    process(cursor)
}
```

**実用例: 兄弟ノードの走査**

```go
// Before
for sibling, ok := node.NextSibling(); ok && sibling != (Cursor{}); sibling, ok = sibling.NextSibling() {
    process(sibling)
}

// After
for sibling, ok := node.NextSibling(); ok && sibling.Valid(); sibling, ok = sibling.NextSibling() {
    process(sibling)
}
```

## 議論のハイライト

- **シンプルさが評価された**: 提案は非常に明快で、既存の`Cursor`APIの使い勝手を改善する小さな追加として受け入れられました
- **関連提案との連携**: issue #77195で提案されている`ParentEdgeKind()`や`ParentEdgeIndex()`メソッドと同様に、`Cursor`の利便性を向上させる一連の改善の一部として位置づけられました
- **迅速な承認**: 提案が「active」ステータスになってからわずか1週間で「likely accept」、さらに1週間で正式承認という迅速なプロセスでした
- **実装の進行**: 承認からわずか数日後にCL 738821として実装が開始され、コミュニティの積極的な対応が見られます
- **Cursorの文脈**: `Cursor`型自体は比較的新しく（v0.34.0で追加）、issue #70859で導入されたAST部分走査・多段階走査を可能にする機能の一部です。`Valid()`はこの新しいAPIをより使いやすくするための自然な拡張と見なされました

## 関連リンク

- [関連Issue #70859 - Cursor型の導入提案](https://github.com/golang/go/issues/70859)
- [関連Issue #77195 - ParentEdge系メソッドの追加提案](https://github.com/golang/go/issues/77195)
- [Proposal Issue](https://github.com/golang/go/issues/76361)
- [Review Minutes](https://github.com/golang/go/issues/76361#issuecomment-3814234826)
