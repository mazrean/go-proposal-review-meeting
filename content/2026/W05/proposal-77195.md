---
issue_number: 77195
title: "x/tools/go/astutil/inspector: add Cursor.IsChildOf"
previous_status: active
current_status: likely_accept
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "最終提案（コメント）"
    url: https://github.com/golang/go/issues/77195#issuecomment-3780616500
  - title: "proposal: x/tools/go/ast/inspector: add Cursor, to enable partial and multi-level traversals · Issue #70859 · golang/go"
    url: https://github.com/golang/go/issues/70859
  - title: "関連Issue: Cursor.Valid()メソッド #76361"
    url: https://github.com/golang/go/issues/76361
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77195
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
---
## 概要
`x/tools/go/ast/inspector`パッケージの`Cursor`型に、`ParentEdge()`メソッドが返す2つの値（edge.KindとIndex）を個別に取得できる便利メソッド`ParentEdgeKind()`と`ParentEdgeIndex()`を追加する提案です。これにより、親ノードとの関係性を判定する際のコードが簡潔になります。

## ステータス変更
**active** → **likely_accept**

Proposal Review Meetingでの議論により、当初提案されていた`IsChildOf`メソッドから設計が変更され、より柔軟な2つの独立したアクセサメソッドを提供する形で承認見込みとなりました。この変更により、単純な等値比較だけでなく、より複雑な条件判定にも対応できるようになりました。

## 技術的背景

### 現状の問題点
`inspector.Cursor.ParentEdge()`メソッドは、カーソルの親ノードに対する「エッジの種類（edge.Kind）」と「リスト内のインデックス」の2つの値をペアで返します。これは両方を同時に計算する方が効率的なためですが、実際にはエッジの種類だけが必要な場合が大半です。

現在は以下のように2つ目の戻り値を`_`で破棄する必要があり、特に長い条件式の中では冗長になります:

```go
if ek, _ := cur.ParentEdge(); ek == edge.CallExpr_Args { ... }
```

そのため、x/tools内部では`astutil.IsChildOf`というヘルパー関数が使われており、現在23箇所で利用されています:

```go
if astutil.IsChildOf(cur, edge.SelectorExpr_Sel) { ... }
```

しかし、この関数は`internal`パッケージにあるため公開APIとして使えず、またメソッドではないため記述が直感的ではありません。

### 提案された解決策
当初は`IsChildOf`メソッドの追加が提案されましたが、レビュー過程での議論を経て、より汎用的な2つのアクセサメソッドを提供する形に変更されました:

```go
package inspector // golang.org/x/tools/go/ast/inspector

type Cursor ...

// ParentEdge()の第1要素（edge.Kind）を返す
func (Cursor) ParentEdgeKind() edge.Kind

// ParentEdge()の第2要素（index）を返す
func (Cursor) ParentEdgeIndex() int
```

## これによって何ができるようになるか

AST（抽象構文木）を走査する際、現在のノードが親ノードのどのフィールドに含まれているかを簡潔に判定できるようになります。これはGo言語の静的解析ツールやリファクタリングツールの開発において非常に重要な機能です。

### コード例

```go
// Before: 従来の書き方（不要な値を破棄する必要がある）
if ek, _ := cur.ParentEdge(); ek == edge.CallExpr_Args {
    // このノードは関数呼び出しの引数
}

// Before: 内部ヘルパー関数（publicには使えない）
if astutil.IsChildOf(cur, edge.SelectorExpr_Sel) {
    // このノードはセレクタの選択部分（x.Selのsel部分）
}

// After: 新APIを使った書き方（簡潔で直感的）
if cur.ParentEdgeKind() == edge.CallExpr_Args {
    // このノードは関数呼び出しの引数
}

// After: インデックスも必要な場合
if cur.ParentEdgeKind() == edge.CallExpr_Args && cur.ParentEdgeIndex() == 0 {
    // このノードは関数呼び出しの第1引数
}

// After: より複雑な条件（IsChildOfでは不可能）
if k := cur.ParentEdgeKind(); k == edge.CallExpr_Args || k == edge.CompositeLit_Elts {
    // 複数のエッジタイプをまとめて判定
}
```

**実用例**: 関数呼び出しの引数として使われているidentifierだけを検出したい場合、`ParentEdgeKind() == edge.CallExpr_Args`で簡潔に判定できます。従来は`ParentEdge()`を呼んで戻り値の一方を破棄する必要がありました。

## 議論のハイライト

- **命名の議論**: 当初`IsChildOf`が提案されたが、@DanielMorsingから「"Child of"はノードタイプを示唆し、エッジに対しては違和感がある」との指摘があり、`HasParentEdgeKind`などの代替案が検討されました

- **設計の改善**: @aclementsから「`Has`は単なる`==`チェックなので、`ParentEdgeKind()`を提供して呼び出し側で柔軟に比較させる方が良い」との提案があり、2つの独立したアクセサメソッドを提供する最終形に至りました

- **柔軟性の向上**: メソッド形式に変更することで、単純な等値比較だけでなく、複数の値との比較（`||`）やスイッチ文での使用など、より多様な使い方が可能になりました

- **実装CL**: 既に実装CL（go.dev/cl/740280）が作成されており、Final Comment Period（最終コメント期間）に入っています

- **広範な利用実績**: x/tools内部で既に23箇所で使われている実績があり、実用性が実証されています

## 関連リンク

- [最終提案（コメント）](https://github.com/golang/go/issues/77195#issuecomment-3780616500)
- [proposal: x/tools/go/ast/inspector: add Cursor, to enable partial and multi-level traversals · Issue #70859 · golang/go](https://github.com/golang/go/issues/70859)
- [関連Issue: Cursor.Valid()メソッド #76361](https://github.com/golang/go/issues/76361)
- [Proposal Issue](https://github.com/golang/go/issues/77195)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
