---
issue_number: 77006
title: "x/net/html: add NodeType.String() method"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "参考: go/constant.Kind String()提案（却下）"
    url: https://github.com/golang/go/issues/46211
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77006
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "関連: regexp/syntax.Op String()メソッド提案"
    url: https://github.com/golang/go/issues/22684
---

## 要約

## 概要
x/net/htmlパッケージのNodeType型にString()メソッドを追加し、デバッグ時にノードタイプを人間が読める形式（"TextNode", "ElementNode"など）で出力できるようにする提案です。

## ステータス変更
**likely_accept** → **accepted**

2026年1月28日のProposal Review Meetingにおいて、「コンセンサスに変更なし」として正式に承認されました。この提案は2026年1月21日に「likely accept」とされており、1週間の最終意見募集期間を経て異議が出なかったため、スムーズに承認に至りました。

## 技術的背景

### 現状の問題点
x/net/htmlパッケージのNodeType型は列挙型（enum）として定義されていますが、String()メソッドが実装されていません。そのため、デバッグ時にNodeTypeをPrintfで出力すると、以下のように数値のみが表示されます。

```go
// 現在の動作
var nt html.NodeType = html.ElementNode
fmt.Println(nt) // 出力: 3（数値のみで意味が不明瞭）
```

NodeTypeは以下の7つの定数で構成されています:
- ErrorNode (0): エラーノード
- TextNode (1): テキストコンテンツ
- DocumentNode (2): ドキュメントルート
- ElementNode (3): HTML要素（タグ）
- CommentNode (4): HTMLコメント
- DoctypeNode (5): DOCTYPE宣言
- RawNode (6): エスケープなしの生HTML（パーサは返さないが、Renderに渡せる）

どの定数がどの数値に対応するかを記憶しておくのは困難で、デバッグ効率が低下していました。

### 提案された解決策
Go標準の`stringer`ツールを使用して、NodeType.String()メソッドを自動生成します。これにより、列挙値が自動的に対応する定数名の文字列表現に変換されます。

なお、同じパッケージ内のTokenType型は既に手書きのString()メソッドを持っており（16年前の最初のコミットから存在）、switch文で各ケースを処理していますが、今回は保守性向上のためstringerによる自動生成が採用されました。

## これによって何ができるようになるか

デバッグ時にNodeTypeの値を直感的に理解できるようになり、開発効率が向上します。特にcmd/compileやgo/typesに関連する複雑な問題をデバッグする際、ノードの種類を即座に識別できることは大きな利点です。

### コード例

```go
// Before: 数値のみが出力される
package main

import (
    "fmt"
    "golang.org/x/net/html"
)

func main() {
    node := &html.Node{Type: html.ElementNode}
    fmt.Printf("Node type: %v\n", node.Type)
    // 出力: Node type: 3
}

// After: 人間が読める形式で出力される
package main

import (
    "fmt"
    "golang.org/x/net/html"
)

func main() {
    node := &html.Node{Type: html.ElementNode}
    fmt.Printf("Node type: %v\n", node.Type)
    // 出力: Node type: ElementNode

    // デバッグログでの活用例
    fmt.Printf("Processing %s with data: %s\n", node.Type, node.Data)
    // 出力: Processing ElementNode with data: div
}
```

## 議論のハイライト

- **類似ケースの参照**: 同様の提案は過去にも複数受理されています。例えば、regexp/syntax.Op型のString()メソッド追加（#22684）や、go/constant.Kind型への同様の機能追加提案（#46211、ただし却下）など、列挙型への可読性向上機能は一貫して評価されています。

- **既存実装との一貫性**: earthboundkid氏が指摘したように、同パッケージのTokenType型は手書きのString()メソッドを持っています。今後はstringerツールによる自動生成に統一することで、保守性が向上します。

- **迅速な実装**: 提案が「likely accept」とされた翌日（2026年1月22日）には、既に実装CL（go.dev/cl/738100）が提出されており、コミュニティの積極的な対応が見られました。

- **議論の簡潔性**: この提案は技術的に明確で論争の余地が少なかったため、2週間以内という短期間で承認に至りました。標準的な列挙型改善として、異議なく受け入れられています。

## 関連リンク

- [参考: go/constant.Kind String()提案（却下）](https://github.com/golang/go/issues/46211)
- [Proposal Issue](https://github.com/golang/go/issues/77006)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連: regexp/syntax.Op String()メソッド提案](https://github.com/golang/go/issues/22684)
