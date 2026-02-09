---
issue_number: 77195
title: "x/tools/go/astutil/inspector: add Cursor.ParentEdge{Kind,Index} convenience accessors"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-02-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3872311559
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77195
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3872311559
  - title: "関連Issue: Cursor.Valid()の提案"
    url: https://github.com/golang/go/issues/76361
  - title: "関連Issue: Cursor導入時の提案"
    url: https://github.com/golang/go/issues/70859
---
## 概要
`golang.org/x/tools/go/ast/inspector` パッケージの `Cursor` 型に、既存の `ParentEdge` メソッドが返す情報（エッジ種別とインデックス）を個別に取得できる便利メソッド `ParentEdgeKind` と `ParentEdgeIndex` を追加する提案です。

## ステータス変更
**likely_accept** → **accepted**

2026年1月21日にProposal Review Meetingで議論され、likely acceptとなった後、2週間のfinal comment期間を経て異論が出なかったため、2026年2月9日に正式に受理されました。実装CL（https://go.dev/cl/740280）も既に提出されています。

## 技術的背景

### 現状の問題点
`inspector.Cursor.ParentEdge` メソッドは、親ノードに対する現在のノードの関係性を表す「エッジ種別（edge.Kind）」と「インデックス（int）」の2つの値をペアで返します。例えば、関数呼び出し `f(x, y)` において、引数 `x` は `edge.CallExpr_Args` というエッジ種別とインデックス `0` を持ちます。

しかし実際のコードでは、多くの場合エッジ種別だけが必要であり、以下のようにインデックスを捨てる書き方が頻出していました。

```go
if ek, _ := cur.ParentEdge(); ek == edge.CallExpr_Args { ... }
```

このパターンは条件式の中で使われることが多く、可読性を低下させていました。そのため、x/tools内部では `astutil.IsChildOf` という内部ヘルパー関数が作られ、23箇所で使用されていました。

### 提案された解決策
当初は `IsChildOf` を公開APIとして提供する案でしたが、レビュー議論の中で「Has」や「IsChildOf」という名前は判定関数を示唆するものの、実際には単純な `==` チェックであることから、より柔軟性の高いアクセサメソッドとして設計し直されました。

最終的に、`ParentEdge` が返す2つの値を個別に取得できる以下のメソッドが提案されました。

```go
package inspector // golang.org/x/tools/go/ast/inspector

// ParentEdgeKind は ParentEdge の第1返り値（エッジ種別）のみを返す
func (Cursor) ParentEdgeKind() edge.Kind

// ParentEdgeIndex は ParentEdge の第2返り値（インデックス）のみを返す
func (Cursor) ParentEdgeIndex() int
```

## これによって何ができるようになるか

AST（抽象構文木）を走査する際、特定のノードが親ノードのどのフィールドに属しているかを判定するコードが簡潔に書けるようになります。

### コード例

```go
// Before: 既存の書き方（不要な返り値を捨てる必要がある）
if ek, _ := cur.ParentEdge(); ek == edge.CallExpr_Args {
    // このノードは関数呼び出しの引数である
    handleArgument(cur.Node())
}

// After: 新APIを使った書き方（意図が明確）
if cur.ParentEdgeKind() == edge.CallExpr_Args {
    // 同じ判定がより読みやすく
    handleArgument(cur.Node())
}

// インデックスも必要な場合
if cur.ParentEdgeKind() == edge.CallExpr_Args && cur.ParentEdgeIndex() == 0 {
    // 第1引数の場合のみ処理
    handleFirstArgument(cur.Node())
}
```

この変更により、静的解析ツールやリファクタリングツールの実装者が、より読みやすく保守しやすいコードを書けるようになります。x/tools内部の23箇所のヘルパー関数呼び出しも、この新APIに置き換えられる予定です。

## 議論のハイライト

- **命名の議論**: 当初 `IsChildOf` という名前が提案されましたが、@DanielMorsingから「"Child of"はノード型を示唆するがパラメータはエッジ種別であり混乱を招く」との指摘があり、`HasParentEdgeKind` などが検討されました
- **設計の転換**: @aclementsから「"Has"は単なる `==` チェックであり、アクセサメソッドとして提供し呼び出し側で自由に比較させる方が柔軟」との提案があり、最終的に `ParentEdgeKind()` と `ParentEdgeIndex()` の2つのアクセサメソッドに決定
- **実用性の確認**: x/tools内部で既に23箇所の使用実績があることから、この機能の需要が実証されており、迅速に受理された
- **関連提案との一貫性**: issue #76361（Cursor.Valid()の追加）や issue #70859（Cursor自体の導入）など、inspector.Cursor APIを段階的に使いやすくする取り組みの一環として位置づけられています

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/77195)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3872311559)
- [関連Issue: Cursor.Valid()の提案](https://github.com/golang/go/issues/76361)
- [関連Issue: Cursor導入時の提案](https://github.com/golang/go/issues/70859)
