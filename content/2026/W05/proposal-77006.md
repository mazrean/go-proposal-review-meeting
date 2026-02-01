---
issue_number: 77006
title: "x/net/html: add NodeType.String() method"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814234502
  - title: "関連: go/constant.Kind.String提案"
    url: https://github.com/golang/go/issues/46211
  - title: "関連: regexp/syntax.Op.String提案"
    url: https://github.com/golang/go/issues/22684
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77006
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
---

## 要約

## 概要
`x/net/html`パッケージの`NodeType`型にデバッグを容易にするための`String()`メソッドを追加する提案です。`NodeType`は単なる整数型の列挙値であり、現在は値を出力しても数値しか表示されないため、`stringer`ツールを使用して人間が読める文字列表現を返すメソッドを自動生成します。

## ステータス変更
**likely_accept** → **accepted**

2026年1月28日の提案レビュー会議で、1週間前の「likely accept」判定から特に異論がなかったため正式に承認されました。既に実装CL（go.dev/cl/738100）が提出されており、レビュー待ちの状態です。

## 技術的背景

### 現状の問題点
`x/net/html`パッケージで定義されている`NodeType`は、HTML文書のノード種別を表す列挙型です。以下の定数が定義されています。

```go
const (
    ErrorNode NodeType = iota
    TextNode
    DocumentNode
    ElementNode
    CommentNode
    DoctypeNode
    RawNode
)
```

現在、`NodeType`の値をデバッグ出力すると単なる整数（0, 1, 2...）として表示されます。開発者はこれらの数値がどのノード種別に対応するのかを記憶するか、ドキュメントを確認する必要があり、デバッグ効率が低下しています。

同じパッケージ内の`TokenType`には既に手動で実装された`String()`メソッドがあり、この実装は16年前のオリジナルコミットから存在していますが、`NodeType`には同様のメソッドがありませんでした。

### 提案された解決策
`NodeType`に対して、[stringer](https://pkg.go.dev/golang.org/x/tools/cmd/stringer)ツールを使用して`String()`メソッドを自動生成します。`stringer`は、整数型の列挙値に対して`fmt.Stringer`インターフェースを満たす`String()`メソッドを自動生成するGo公式ツールです。

生成されるメソッドは、各`NodeType`の値を対応する定数名（"ErrorNode", "TextNode", "ElementNode"など）の文字列として返します。

## これによって何ができるようになるか

HTMLパーサーのデバッグ時に、`NodeType`の値が人間が読める形式で出力されるようになります。これにより、ログやデバッガ上で即座にノードの種別を理解でき、開発効率が大幅に向上します。

### コード例

```go
// Before: 従来の出力（数値のみ）
node := &html.Node{Type: html.ElementNode}
fmt.Printf("Node type: %v\n", node.Type)
// 出力: Node type: 3  （どのノード種別か不明瞭）

// After: String()メソッド追加後
node := &html.Node{Type: html.ElementNode}
fmt.Printf("Node type: %v\n", node.Type)
// 出力: Node type: ElementNode  （明確で読みやすい）

// デバッグログの例
for n := doc.FirstChild; n != nil; n = n.NextSibling {
    fmt.Printf("Processing %s node\n", n.Type)
    // 出力: Processing DocumentNode node
    // 出力: Processing ElementNode node
    // など、一目でノード種別が分かる
}
```

## 議論のハイライト

- **既存パターンとの一貫性**: 同じパッケージ内の`TokenType`が16年前から`String()`メソッドを持っていることが指摘され、`NodeType`にも同様のメソッドを追加することに説得力がありました
- **類似提案の存在**: 関連issueとして`go/constant.Kind.String`（#46211）や`regexp/syntax.Op.String`（#22684）など、標準ライブラリの他の列挙型にも同様の機能追加が提案されており、デバッグ時の可読性向上は共通のニーズであることが示されています
- **実装の簡潔性**: `stringer`ツールを使用した自動生成により、実装とメンテナンスが容易であることが評価されました
- **迅速な承認**: 提案から2週間弱で「likely accept」となり、さらに1週間で正式承認という速いペースで進行しました。これは提案内容が明確でリスクが低く、開発者にとって明らかに有益であると判断されたためです

## 関連リンク
- [Proposal Issue](https://github.com/golang/go/issues/77006)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [実装CL](https://go.dev/cl/738100)
- [関連: go/constant.Kind.String提案](https://github.com/golang/go/issues/46211)
- [関連: regexp/syntax.Op.String提案](https://github.com/golang/go/issues/22684)

## Sources
- [stringer command - golang.org/x/tools/cmd/stringer - Go Packages](https://pkg.go.dev/golang.org/x/tools/cmd/stringer)
- [html package - golang.org/x/net/html - Go Packages](https://pkg.go.dev/golang.org/x/net/html)
- [net/html/node.go at master · golang/net](https://github.com/golang/net/blob/master/html/node.go)
- [net/html/token.go at master · golang/net](https://github.com/golang/net/blob/master/html/token.go)

## 関連リンク

- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814234502)
- [関連: go/constant.Kind.String提案](https://github.com/golang/go/issues/46211)
- [関連: regexp/syntax.Op.String提案](https://github.com/golang/go/issues/22684)
- [Proposal Issue](https://github.com/golang/go/issues/77006)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
