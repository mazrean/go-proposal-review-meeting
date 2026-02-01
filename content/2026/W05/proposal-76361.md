---
issue_number: 76361
title: "x/tools/go/ast/inspector: add (\\*Cursor).Valid() bool method"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/76361#issuecomment-3814234826
  - title: "関連Issue #77195: ParentEdgeKind/ParentEdgeIndex追加提案"
    url: https://github.com/golang/go/issues/77195
  - title: "関連Issue #70859: Cursor型の導入提案（クローズ済み）"
    url: https://github.com/golang/go/issues/70859
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/76361
---

## 要約

## 概要
`x/tools/go/ast/inspector`パッケージの`Cursor`型に、カーソルが有効かどうかを判定する`Valid()`メソッドを追加する提案です。現在はゼロ値との比較や内部フィールドのnilチェックで判定していますが、これをより明示的で分かりやすいAPIとして提供します。

## ステータス変更
**likely_accept** → **accepted**

2026年1月28日のProposal Review Meetingで、議論の余地がないと判断され正式に承認されました。提案内容がシンプルで明確であり、既存の`Cursor` APIの使い勝手を向上させるものとして評価されました。

## 技術的背景

### 現状の問題点
`inspector.Cursor`を使用する際、カーソルが有効（ゼロ値でない）かどうかを確認する必要がしばしば発生します。現在は以下のような回りくどい方法で判定しています。

```go
// 方法1: ゼロ値との比較
if cur != (Cursor{}) {
    use(cur)
}

// 方法2: 内部フィールドのnilチェック（意図が不明瞭）
if cur.Inspector() != nil {
    use(cur)
}
```

特に方法2は、`Inspector()`メソッドの本来の目的が「所属するInspectorインスタンスを取得すること」であるため、有効性チェックとして使うのは意図が読み取りにくくなります。

### 提案された解決策
`Cursor`型に`Valid()`メソッドを追加し、カーソルが有効かどうかを直接的に判定できるようにします。

```go
package inspector // golang.org/x/tools/go/ast/inspector

type Cursor ...

// Valid reports whether the cursor is valid.
func (Cursor) Valid() bool
```

## これによって何ができるようになるか

AST（抽象構文木）解析ツールの開発において、カーソルの有効性チェックがより自然で読みやすくなります。特に以下のような場面で有用です。

1. **オプショナルなノード探索**: `FindNode()`や`FindByPos()`など、ノードが見つからない場合にゼロ値を返すメソッドの結果チェック
2. **条件分岐での可読性向上**: 複数の条件を組み合わせる際、意図が明確になる
3. **エラーハンドリング**: カーソルが無効な状態での操作を防ぐガード節

### コード例

```go
// Before: ゼロ値との比較（冗長）
cur, ok := parentCur.FindNode(targetNode)
if cur != (inspector.Cursor{}) {
    // cursorを使った処理
    processNode(cur)
}

// Before: Inspector()を使った判定（意図が不明瞭）
if cur.Inspector() != nil {
    processNode(cur)
}

// After: Valid()メソッドを使った判定（明確で自然）
cur, ok := parentCur.FindNode(targetNode)
if cur.Valid() {
    processNode(cur)
}
```

## 議論のハイライト

- **シンプルな提案**: 提案内容が非常に明確で、既存APIの自然な拡張として受け入れられました
- **関連提案との整合性**: Issue #77195（`ParentEdge`の分解メソッド追加）と同様に、`Cursor` APIの使い勝手を向上させる一連の改善の一部
- **実装の早さ**: likely_accept宣言後すぐに実装CL（go.dev/cl/738821）が提出されました
- **議論不要の承認**: 提案内容に異論がなく、likely_acceptから1週間後のミーティングで即座にacceptedに移行

## 関連リンク
- [Proposal Issue](https://github.com/golang/go/issues/76361)
- [Review Minutes](https://github.com/golang/go/issues/76361#issuecomment-3814234826)
- [関連Issue #70859: Cursor型の導入提案（クローズ済み）](https://github.com/golang/go/issues/70859)
- [関連Issue #77195: ParentEdgeKind/ParentEdgeIndex追加提案](https://github.com/golang/go/issues/77195)
- [実装CL](https://go.dev/cl/738821)
- [Inspector.Cursor 公式ドキュメント](https://pkg.go.dev/golang.org/x/tools/go/ast/inspector)

---

**Sources:**
- [inspector package - golang.org/x/tools/go/ast/inspector - Go Packages](https://pkg.go.dev/golang.org/x/tools/go/ast/inspector)
- [typesinternal package - golang.org/x/tools/internal/typesinternal - Go Packages](https://pkg.go.dev/golang.org/x/tools/internal/typesinternal)

## 関連リンク

- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [Review Minutes](https://github.com/golang/go/issues/76361#issuecomment-3814234826)
- [関連Issue #77195: ParentEdgeKind/ParentEdgeIndex追加提案](https://github.com/golang/go/issues/77195)
- [関連Issue #70859: Cursor型の導入提案（クローズ済み）](https://github.com/golang/go/issues/70859)
- [Proposal Issue](https://github.com/golang/go/issues/76361)
