---
issue_number: 77273
title: "spec: generic methods for Go"
previous_status: discussions
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "proposal: spec: generic methods for Go · Issue #77273 · golang/go"
    url: https://github.com/golang/go/issues/77273
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "関連Issue #49085: Allow type parameters in methods (2021)"
    url: https://github.com/golang/go/issues/49085
  - title: "関連Issue #50981: Add generics to methods (2022)"
    url: https://github.com/golang/go/issues/50981
---

## 要約

## 概要
Goのメソッドにジェネリック型パラメータを許可する提案です。これまでGoは「メソッドの主な役割はインターフェースの実装」と考えていたため、型パラメータを持つメソッド（ジェネリックメソッド）を禁止していました。この提案は視点を変え、「メソッドは型に紐づく関数として単独で有用」という観点からジェネリックメソッドを導入します。

## ステータス変更
**(新規)** → **active**

2026年1月28日のProposal Review Meetingで、この提案が**active**ステータスに移行しました。提案は2026年1月22日にRobert Griesemer（Goコアチーム）により提出され、わずか6日で活発な議論対象となりました。これは、900件以上の賛同を集めた過去の関連提案（#49085、2021年10月）や、コミュニティからの長年の要望を受けてのものです。

## 技術的背景

### 現状の問題点

現在のGoでは、関数は型パラメータを宣言できますが、メソッドは宣言できません：

```go
// OK: ジェネリック関数
func Map[T, U any](slice []T, f func(T) U) []U { ... }

// NG: メソッドに型パラメータは不可
type Stream[T any] struct { ... }
func (s *Stream[T]) Map[U any](f func(T) U) *Stream[U] { ... }  // コンパイルエラー
```

この制限により、関数型スタイルのAPIやメソッドチェーン、DSL（Domain Specific Language）の実装が困難になっています。現在は回避策として、すべての型パラメータをレシーバ型に宣言する必要がありますが、これは「何回マップするか不明」な場合に実用的ではありません。

**なぜ今まで禁止されていたか：**
Goではインターフェース実装が暗黙的（動的）であり、どのジェネリックメソッドのインスタンス化が必要かをコンパイル時に知る方法がありません。たとえば、`Read[T any]([]T) (int, error)`というジェネリックメソッドを持つ型が、実行時にどの`T`で呼ばれるか予測不可能です。この問題は2021年のType Parameters Proposalで詳細に議論されました。

### 提案された解決策

提案は**視点の転換**を行います：
- **従来**: メソッド = インターフェース実装の手段
- **新提案**: メソッド = 型の名前空間に紐づく関数（インターフェースとは独立）

**構文変更：**

```ebnf
// 旧
MethodDecl = "func" Receiver MethodName Signature [ FunctionBody ] .

// 新
MethodDecl = "func" Receiver MethodName [ TypeParameters ] Signature [ FunctionBody ] .
```

**重要な制約：**
- **インターフェースメソッドには型パラメータを追加しない**（構文的に不可）
- したがって、ジェネリックメソッドはインターフェースを実装**できない**
- これは言語仕様の一貫性として明示される

## これによって何ができるようになるか

ジェネリックメソッドにより、以下のようなAPIが可能になります：

1. **メソッドチェーンを使ったストリーム処理**
2. **型安全なDSL**（テストフレームワーク、データパイプライン構築など）
3. **既存の関数をメソッドに昇格**（例: `math/rand/v2.Rand`に`N[T integer]()`メソッド）

### コード例

```go
// Before: ワークアラウンド（型パラメータを全て型に含める）
type Stream[T, U any] struct { ... }
func (s *Stream[T, U]) Map(f func(T) U) *Stream[U, ???] { ... }  // 次の型パラメータは？

// After: ジェネリックメソッドを使った自然な記述
type Stream[T any] struct { ... }
func (s *Stream[T]) Map[U any](f func(T) U) *Stream[U] {
    // Tから新しい型Uへ変換
    ...
}

// メソッドチェーンが可能に
result := stream.
    Filter(predicate).
    Map[string](toString).
    Collect()
```

```go
// 実用例: hash.Hashの改善
type Hash struct { ... }

// Before: パッケージレベル関数のみ
maphash.WriteComparable[string](h, "key")

// After: メソッドとして自然に呼べる
func (*Hash) WriteComparable[T comparable](x T) { ... }
h.WriteComparable("key")  // 型推論が効く
```

**インターフェースとの関係：**

```go
// ジェネリックメソッドはインターフェースを実装できない
type Reader struct{ … }
func (*Reader) Read[E any]([]E) (int, error) { … }

// io.Readerとは互換性がない
var _ io.Reader = &Reader{}  // コンパイルエラー
// 理由: Read[E any]([]E)とRead([]byte)は型が一致しない
```

## 議論のハイライト

- **コアチームの懸念（Ian Lance Taylor）**: 「メソッドがインターフェースを実装するかどうかのルールが複雑になる。全てのメソッドが実装に寄与できたが、今後は一部のみになる」。ただし「それでもやる価値はあるかもしれない」とも発言

- **認知的負荷の増加（Chris Hines）**: 「制限を削除する単純化に見えるが、利用者の認知的負荷は増える。どのメソッドがインターフェースを実装できるか判断が必要になる」

- **Griesemerの明確化**: 「インターフェースの構文は一切変更しない。ジェネリックメソッドは関数型の同一性チェックで自然に除外される」

- **実装パターンの議論**: インターフェースとの互換性が必要な場合、ラッパーメソッドを生成するパターンが提案されている
  ```go
  func (x X) Read[T any]([]T) (int, error) { ... }
  func (x X) ReaderOf[T any]() Reader[T] {
      type wrapper[T any] X
      return wrapper[T](x)
  }
  ```

- **Rustとの比較**: 一部のユーザーはRustがtraitメソッドでジェネリクスをサポートしている点を指摘したが、Rustでも同様の動的ディスパッチ問題があることが議論された

- **Apache Beam Goの具体例（lostluck）**: データパイプライン構築でBeam PythonやJavaと同等の表現力が得られる

- **エラーハンドリングとの相性（Ian Lance Taylor）**: メソッドチェーンはエラーを返す場合に制限がある。Goでは`(value, error)`を返すパターンが多く、純粋な関数型チェーンは限定的な領域でしか機能しない

## 関連リンク
- [Proposal Issue #77273](https://github.com/golang/go/issues/77273)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連Issue #49085: Allow type parameters in methods (2021)](https://github.com/golang/go/issues/49085) - 888件の賛同、361件のコメント
- [関連Issue #50981: Add generics to methods (2022)](https://github.com/golang/go/issues/50981)
- [Type Parameters Proposal - No parameterized methods](https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#No-parameterized-methods)
- [実装CL 738441](https://go.dev/cl/738441) - パーサーでジェネリックメソッド構文を有効化

---

Sources:
- [proposal: spec: generic methods for Go · Issue #77273 · golang/go](https://github.com/golang/go/issues/77273)
- [A Proposal for Adding Generics to Go - The Go Programming Language](https://go.dev/blog/generics-proposal)
- [An Introduction To Generics - The Go Programming Language](https://go.dev/blog/intro-generics)

## 関連リンク

- [proposal: spec: generic methods for Go · Issue #77273 · golang/go](https://github.com/golang/go/issues/77273)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連Issue #49085: Allow type parameters in methods (2021)](https://github.com/golang/go/issues/49085)
- [関連Issue #50981: Add generics to methods (2022)](https://github.com/golang/go/issues/50981)
