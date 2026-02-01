---
issue_number: 77195
title: "x/tools/go/astutil/inspector: add Cursor.IsChildOf"
previous_status: discussions
current_status: likely_accept
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77195
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "関連Issue: Cursor.Valid()の追加 #76361"
    url: https://github.com/golang/go/issues/76361
  - title: "proposal: x/tools/go/ast/inspector: add Cursor - Issue #70859"
    url: https://github.com/golang/go/issues/70859
---

## 要約

## 概要
`inspector.Cursor`型に、親ノードとのエッジ情報(種類とインデックス)を個別に取得できる2つの便利メソッド`ParentEdgeKind()`と`ParentEdgeIndex()`を追加する提案です。現在は`ParentEdge()`メソッドが両方をペアで返しますが、多くの場合エッジの種類(Kind)だけが必要とされるため、よりシンプルなAPIが求められています。

## ステータス変更
**active** → **likely_accept**

2026年1月28日のProposal Review Meetingで議論され、likely accept(承認見込み)のステータスに移行しました。これは最終承認前の「最後のコメント期間」を意味します。議論を経て、当初の`IsChildOf`メソッド案から、より柔軟性の高い2つのアクセサメソッドへと提案内容が改善されたことが評価されました。

## 技術的背景

### 現状の問題点
`inspector.Cursor`の`ParentEdge()`メソッドは、親ノードに対するエッジの種類(edge.Kind)とインデックス(int)の両方を返します。これは効率的ですが、実際には**エッジの種類だけ**が必要なケースが大半です。

```go
// 現在の書き方: 不要な第2戻り値を破棄する必要がある
if ek, _ := cur.ParentEdge(); ek == edge.CallExpr_Args { ... }
```

このパターンは特に長い条件式の中で頻出し、コードの可読性を損ねます。そのため、x/tools内部には既に`IsChildOf`というヘルパー関数が存在し、23箇所で使われています。

```go
// 内部ヘルパー関数(現在は internal/astutil にある)
if astutil.IsChildOf(cur, edge.SelectorExpr_Sel) { ... }
```

### 提案された解決策
議論の結果、単純な真偽値チェック(`IsChildOf`)よりも、**個別のアクセサメソッド**を提供する方がより柔軟だという結論に至りました。

```go
package inspector // golang.org/x/tools/go/ast/inspector

type Cursor struct { ... }

// ParentEdge() の第1・第2戻り値をそれぞれ返す便利メソッド
func (Cursor) ParentEdgeKind() edge.Kind
func (Cursor) ParentEdgeIndex() int
```

## これによって何ができるようになるか

AST(抽象構文木)を走査する際、ノードが親のどのフィールドに属しているかをより簡潔にチェックできるようになります。これはGo言語の静的解析ツールやリファクタリングツール(goplsなど)の開発において頻繁に必要となる操作です。

### コード例

```go
// Before: ParentEdge()を使う場合(冗長)
if ek, _ := cur.ParentEdge(); ek == edge.CallExpr_Args {
    // 関数呼び出しの引数部分かチェック
    ...
}

// After: ParentEdgeKind()を使う場合(簡潔)
if cur.ParentEdgeKind() == edge.CallExpr_Args {
    // 同じチェックがよりシンプルに
    ...
}

// ParentEdgeIndex()も必要な場合
if cur.ParentEdgeKind() == edge.CallExpr_Args && cur.ParentEdgeIndex() > 0 {
    // 第2引数以降かどうかをチェック
    ...
}
```

**実用例**: 関数呼び出し`f(x, y)`のAST走査において
- 識別子`f`は `ParentEdgeKind() == edge.CallExpr_Fun` (関数名部分)
- 識別子`x`は `ParentEdgeKind() == edge.CallExpr_Args && ParentEdgeIndex() == 0` (第1引数)
- 識別子`y`は `ParentEdgeKind() == edge.CallExpr_Args && ParentEdgeIndex() == 1` (第2引数)

## 議論のハイライト

- **命名の改善**: 当初の`IsChildOf`という名前について、「edge(エッジ)をテストしているのにchild of(〜の子)という名前は紛らわしい」という指摘があり、`ParentEdgeIs`や`HasParentEdgeKind`などが検討されました

- **柔軟性の重視**: @aclementsから「単純な`==`チェックよりも、`ParentEdgeKind()`を提供して呼び出し側で好きなように使えるようにする方が良いのでは」という提案があり、+2の支持を得ました

- **最終提案への移行**: 議論の結果、`ParentEdge()`の戻り値を個別に取得できる2つのアクセサメソッドを提供することで合意。これにより単純な比較から複雑な条件分岐まで幅広いユースケースに対応可能になりました

- **実装開始**: 2026年1月29日にCL 740280として実装が投稿されました(+64行、-70行の変更)。内部ヘルパー関数を公開メソッドに置き換える形での実装と見られます

## 関連リンク
- [Proposal Issue](https://github.com/golang/go/issues/77195)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連Issue: Cursor.Valid()の追加 #76361](https://github.com/golang/go/issues/76361)
- [関連Issue: Cursor型の導入 #70859](https://github.com/golang/go/issues/70859)
- [実装CL 740280](https://go-review.googlesource.com/c/tools/+/740280)

---

**Sources:**
- [inspector package - golang.org/x/tools/go/ast/inspector - Go Packages](https://pkg.go.dev/golang.org/x/tools/go/ast/inspector)
- [edge package - golang.org/x/tools/go/ast/edge - Go Packages](https://pkg.go.dev/golang.org/x/tools/go/ast/edge)
- [astutil package - golang.org/x/tools/internal/astutil - Go Packages](https://pkg.go.dev/golang.org/x/tools/internal/astutil)
- [proposal: x/tools/go/ast/inspector: add Cursor - Issue #70859](https://github.com/golang/go/issues/70859)

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/77195)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連Issue: Cursor.Valid()の追加 #76361](https://github.com/golang/go/issues/76361)
- [proposal: x/tools/go/ast/inspector: add Cursor - Issue #70859](https://github.com/golang/go/issues/70859)
