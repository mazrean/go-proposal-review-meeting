---
issue_number: 77273
title: "spec: generic methods for Go"
previous_status: 
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "関連Issue #49085: Allow type parameters in methods"
    url: https://github.com/golang/go/issues/49085
  - title: "関連Issue #50981: Add generics to method"
    url: https://github.com/golang/go/issues/50981
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77273
  - title: "Review Minutes (2026-01-28)"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
---
## 概要
このproposalは、Go言語にジェネリックメソッド（型パラメータを持つメソッド）を導入するものです。現在、ジェネリック関数は存在しますが、メソッドは独自の型パラメータを宣言できません。この制限を撤廃し、メソッドをレシーバ付きのジェネリック関数として扱えるようにすることで、コードの組織化と可読性を向上させます。

## ステータス変更
**（なし）** → **active**

2026年1月28日のProposal Review Meetingにおいて、このissueが正式に議題として取り上げられ、"Active"ステータスに移行しました。これは議論が本格的に開始されたことを意味し、実装に向けた具体的な検討が始まっています。議事録では「added to minutes」とのみ記載されており、詳細な議論はこれから行われる見込みです。

## 技術的背景

### 現状の問題点

Go 1.18で導入されたジェネリクスでは、関数は型パラメータを持てますが、メソッドは独自の型パラメータを宣言できません。メソッドはレシーバ型の型パラメータを利用できるのみです。

```go
// これは可能（ジェネリック関数）
func Map[T, U any](slice []T, f func(T) U) []U { ... }

// これは不可能（メソッドに独自の型パラメータを追加できない）
type Stream[T any] struct { ... }
func (s *Stream[T]) Map[U any](f func(T) U) *Stream[U] { ... }  // コンパイルエラー
```

この制限により、以下のような問題が発生しています:

1. **名前空間の問題**: 同じパッケージ内に複数の類似型がある場合、パッケージレベル関数では名前の衝突を避けるため `MapHashSet`、`MapTreeSet` のように冗長な命名が必要
2. **メソッドチェーンの不可能性**: `x.a().b().c()` のような流暢なAPIが書けない
3. **標準ライブラリの不整合**: `math/rand/v2.Rand`型はジェネリック関数`N[T Integer](n T) T`に対応するメソッドを提供できない

### 提案された解決策

メソッド宣言の構文を関数宣言と同様に拡張し、型パラメータを許可します:

**現在の構文:**
```ebnf
MethodDecl = "func" Receiver MethodName Signature [ FunctionBody ] .
```

**新しい構文:**
```ebnf
MethodDecl = "func" Receiver MethodName [ TypeParameters ] Signature [ FunctionBody ] .
```

重要な制約として、**インターフェースメソッドは型パラメータを持てません**。つまり、ジェネリックメソッドはインターフェースを満たすことができません。

## これによって何ができるようになるか

### 1. 同一パッケージ内での名前の統一

複数の集合型を同じパッケージで提供する場合、すべてに同じメソッド名を使えます:

```go
type HashSet[E comparable] struct { ... }
func (s *HashSet[E]) Map[F any](f func(E) F) *HashSet[F] { ... }

type TreeSet[E cmp.Ordered] struct { ... }
func (s *TreeSet[E]) Map[F any](f func(E) F) *TreeSet[F] { ... }

// パッケージレベル関数だと MapHashSet, MapTreeSet のように分ける必要があった
```

### 2. メソッドチェーンとメソッド値

```go
type Reader struct { ... }
func (*Reader) Read[E any]([]E) (int, error) { ... }

var r Reader
r.Read([]int{1, 2, 3})     // 型推論で動作
r.Read[string]([]string{})  // 明示的な型引数も可能

// メソッド値も利用可能
readFunc := r.Read[byte]  // func([]byte) (int, error) 型の関数値
```

### 3. 標準ライブラリの改善

`math/rand/v2.Rand`型にジェネリック`N`メソッドを追加できます（現在は関数のみ存在）:

```go
type Rand struct { ... }
func (r *Rand) N[T Integer](n T) T { ... }

var rng Rand
rng.N(100)        // 0-99のint
rng.N(uint64(100)) // 0-99のuint64
```

### コード例

```go
// Before: パッケージレベル関数を使った回避策
type Stream[T any] struct { data []T }
func MapStream[T, U any](s *Stream[T], f func(T) U) *Stream[U] {
    return &Stream[U]{...}
}

var s Stream[int]
result := MapStream(MapStream(s, toString), toUpper) // ネストして読みにくい

// After: ジェネリックメソッドを使った自然な書き方
func (s *Stream[T]) Map[U any](f func(T) U) *Stream[U] {
    return &Stream[U]{...}
}

result := s.Map(toString).Map(toUpper) // 左から右に読める
```

## 議論のハイライト

- **インターフェース満たさない問題への懸念**: `Read[E any]([]E) (int, error)`メソッドは`io.Reader`を満たさないため、混乱を招く可能性がある。これはFAQで最も多く聞かれる質問になると予測されている（Merovius氏）

- **発想の転換**: これまでGoチームは「メソッドはインターフェースを満たすためのもの」という視点からジェネリックメソッドを却下してきたが、この提案は「メソッドはコード組織化のツールでもある」という視点への転換を意味する

- **完全な後方互換性**: 既存の制限を取り除くだけなので、既存コードは一切影響を受けない。将来的にインターフェースメソッドへの型パラメータ追加の道も閉ざさない

- **実装の実現可能性**: メソッド呼び出しは静的に解決できるため、ジェネリック関数呼び出しに書き換え可能。技術的には実装可能だが、import/exportフォーマットの変更が必要で、ツールエコシステム全体への影響は大きい（griesemer氏）

- **関数型identity規則の調整が必要**: 型パラメータセクションも含めて関数型の同一性を判定する必要がある。これにより、ジェネリックメソッドが誤ってインターフェースを満たすことを防ぐ

- **Reflectionでのアクセス不可**: `reflect`パッケージ経由ではジェネリックメソッドにアクセスできない（ジェネリック関数と同様の制限）

## 関連リンク

- [関連Issue #49085: Allow type parameters in methods](https://github.com/golang/go/issues/49085)
- [関連Issue #50981: Add generics to method](https://github.com/golang/go/issues/50981)
- [Proposal Issue](https://github.com/golang/go/issues/77273)
- [Review Minutes (2026-01-28)](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
