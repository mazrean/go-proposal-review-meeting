---
issue_number: 77273
title: "spec: generic methods for Go"
previous_status: active
current_status: likely_accept
changed_at: 2026-02-11T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3886687081
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77273
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3886687081
  - title: "関連Issue #49085"
    url: https://github.com/golang/go/issues/49085
  - title: "関連Issue #50981"
    url: https://github.com/golang/go/issues/50981
---
## 概要
このproposalは、Go言語において具象型のメソッド宣言に型パラメータを許可し、ジェネリックメソッドを導入することを提案しています。これにより、関数と同様にメソッドでもジェネリクスを使えるようになりますが、重要な制限として、これらのジェネリックメソッドはインターフェースメソッドを満たすことはできません。

## ステータス変更
**active** → **likely_accept**

2026年2月11日のProposal Review Meetingにて「likely accept」(承認見込み)に移行しました。これは、約3週間にわたる活発な議論を経て、提案の技術的妥当性とGoコミュニティからの強い支持が認められたためです。レビューグループは最終コメント期間を設けており、重大な反対意見がなければ正式に承認される見込みです。

## 技術的背景

### 現状の問題点

現在のGoでは、関数は型パラメータを持てますがメソッドは持てません。メソッドは受信型がジェネリック型の場合のみその型パラメータを使えますが、メソッド自身が新たな型パラメータを宣言することはできません。

```go
// 現在: これは許可されていない
type Reader struct{}
func (r *Reader) Read[E any](p []E) (int, error) { ... }  // コンパイルエラー

// 現在の回避策: パッケージレベルの関数として定義
func Read[E any](r *Reader, p []E) (int, error) { ... }
```

この制限の理由は、メソッドの主な役割をインターフェース実装と見なしてきたためです。ジェネリックメソッドを許可すればインターフェースでもジェネリックメソッドが必要になりますが、Goの構造的型付け(暗黙的インターフェース実装)の特性上、どの型パラメータのインスタンス化が実行時に必要になるかをコンパイル時に判断できないため、効率的な実装が困難です。

### 提案された解決策

このproposalは視点の転換を提案します。**メソッドはインターフェース実装だけでなく、型に関連付けられた関数として、コードの整理や可読性向上にも有用**という観点です。

**構文変更**:
```go
// 新しい構文
MethodDecl = "func" Receiver MethodName [ TypeParameters ] Signature [ FunctionBody ] .
```

**重要な制約**:
- ジェネリックメソッドはインターフェースメソッドを満たせない(型パラメータの有無が型同一性の条件となるため)
- インターフェースの構文は変更されない(インターフェースメソッドに型パラメータは追加できない)
- リフレクションではアクセス不可(インスタンス化されていないジェネリック関数と同様)

## これによって何ができるようになるか

### コード例

**Before: パッケージレベル関数での回避策**
```go
package maphash

type Hash struct{ ... }

// 型パラメータが必要な機能はパッケージ関数として定義
func WriteComparable[T comparable](h *Hash, x T)

// 使用時
var h maphash.Hash
maphash.WriteComparable(&h, "hello")  // パッケージ名を毎回記述
```

**After: ジェネリックメソッドを使った書き方**
```go
package maphash

type Hash struct{ ... }

// メソッドとして定義可能
func (h *Hash) WriteComparable[T comparable](x T)

// 使用時
var h maphash.Hash
h.WriteComparable("hello")  // エディタの補完が効きやすく、直感的
```

### 実践的なユースケース

1. **同一パッケージ内の複数型で同名メソッドを使用**:
```go
type HashSet[E comparable] struct{ ... }
func (s *HashSet[E]) Map[F any](f func(E) F) *HashSet[F] { ... }

type TreeSet[E cmp.Ordered] struct{ ... }
func (s *TreeSet[E]) Map[F any](f func(E) F) *TreeSet[F] { ... }

// パッケージ関数では MapHashSet, MapTreeSet のように名前を変える必要があった
```

2. **型に強く結びついた機能の整理**:
```go
type Parser struct{ ... }

// パーサーに固有のジェネリック操作をメソッドとして整理
func (p *Parser) Parse[T Node](input string) (T, error) { ... }
```

3. **データストア操作の可読性向上**:
```go
type Datastore struct{ ... }
func (d *Datastore) Save[T any](ctx context.Context, value T) error { ... }

// datastore.Save[int](ctx, value) より読みやすい
// Save(datastore, ctx, value) よりも発見しやすい
```

## 議論のハイライト

- **強い支持**: Issue #49085(2021年10月)では900以上の肯定的リアクションがあり、Goコミュニティから長年強く要望されてきた機能

- **インターフェース実装の制約に関する懸念**: `Read[T any]([]T) (int, error)`が`io.Reader`を満たせないことに混乱が生じる可能性。しかしRustなど他言語でも同様の制限があり、適切なエラーメッセージで対処可能との見解

- **言語の直交性への影響**: Ian Lance Taylorは「現在はどのメソッドもインターフェースに貢献できるが、この提案では一部のメソッドはできなくなる」という懸念を表明。しかし「それでも価値がある可能性はある」とコメント

- **段階的拡張の可能性**: 将来的にジェネリックインターフェースメソッドが実装される可能性を完全には排除しないが、このproposalの範囲外。現時点では実装方法が不明

- **実装の実現可能性**: メソッド呼び出しはコンパイル時に静的に解決できるため、技術的には実装可能。パーサーは既に型パラメータを受け付けており(エラーを出すのみ)、変更は比較的小規模

- **言語の簡素化**: Robert Griesemerは「これは制限の除去であり、関数とメソッドの不一致を解消するため、ある意味では言語の簡素化」と説明

- **AI時代の読みやすさ**: 一部からAIコード生成時代には可読性の重要度が下がるとの意見もあったが、「レビュー・デバッグ時の可読性は依然重要」として退けられた

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/77273)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3886687081)
- [関連Issue #49085](https://github.com/golang/go/issues/49085)
- [関連Issue #50981](https://github.com/golang/go/issues/50981)
