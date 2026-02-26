---
issue_number: 77273
title: "spec: generic methods for Go"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-02-25T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3962620065
related_issues:
  - title: "関連Issue #49085"
    url: https://github.com/golang/go/issues/49085
  - title: "関連Issue #50981"
    url: https://github.com/golang/go/issues/50981
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77273
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3962620065
---
## 概要

GoにおいてメソッドにもType Parameterを宣言できるようにする言語仕様の変更提案です。現在のGoではジェネリクスが関数には使えるにもかかわらずメソッドには使えないという非対称性があり、この制約を取り除いて「メソッドはレシーバを持つ関数である」という本来の対称性を回復します。

## ステータス変更
**likely_accept** → **accepted**

2026年2月11日に「likely accept」となった後、2月18日のProposal Review Meetingではコミュニティから寄せられた「レシーバのType ParameterをメソッドのType Parameterの制約として使えるか」という疑問（例: `func (X[A]) Method[B A|int]()`）について議論が続きました。Robert Griesemerが調査した結果、既存のspec上の制約（型パラメータはunion termとして使えない）により不可であることが明確化されました。この懸念事項が整理されたこと、また十分な検討期間を経てコンセンサスに変化がなかったため、2026年2月25日にAustin Clementsが正式に**accepted**と宣言しました。

## 技術的背景

### 現状の問題点

現在のGoではジェネリクス（Type Parameters）は関数には宣言できますが、メソッドには宣言できません。この非対称性はGoが長年「メソッドの主な役割はインターフェイスを実装することである」という前提から来ていました。ジェネリックなインターフェイスメソッドを効率的に実装する手段が不明なため、コンクリートメソッドにも一律に禁止していました。

```go
// 現在: ジェネリック関数は書けるが...
func Map[T, U any](s []T, f func(T) U) []U { ... }

// メソッドにType Parameterを宣言できない（コンパイルエラー）
type Reader struct{}
func (r *Reader) Read[E any](p []E) (int, error) { ... } // 現在は不可
```

この制限により、型に紐づいたジェネリック操作は関数として実装するしかなく、`result.Map(transform).Filter(pred)` のような左から右へ読める自然なメソッドチェーンが書けませんでした。

### 提案された解決策

コンクリートメソッドの宣言構文にType Parameter部を追加できるようにします。関数宣言の文法を拡張し、メソッド宣言でも同様の形式を認めます。

変更前の文法:
```ebnf
MethodDecl = "func" Receiver MethodName Signature [ FunctionBody ] .
```

変更後の文法:
```ebnf
MethodDecl = "func" Receiver MethodName [ TypeParameters ] Signature [ FunctionBody ] .
```

あるいは関数とメソッドの宣言を統一して:
```ebnf
FunctionDecl = "func" [ Receiver ] identifier [ TypeParameters ] Signature [ FunctionBody ] .
```

**重要な制約**: ジェネリックなコンクリートメソッドはインターフェイスメソッドを満たしません。インターフェイス側にType Parameterを追加する構文変更は行わないため、自然な帰結として`func (H) m[P any](P) {}`は`interface { m(string) }`を満たしません。

## これによって何ができるようになるか

**1. イテレータのメソッドチェーン**

`iter.Seq[T]`のような定義型にジェネリックメソッドを追加することで、ストリーム処理を左から右に読める形で記述できます。

**2. `math/rand/v2.Rand`の完全化**

現在`rand.N[T]`関数は存在するが`Rand`型には対応するメソッドがなかった。ジェネリックメソッドでこのギャップを埋められます。

**3. ビルダーパターンや流暢なAPI設計**

```go
// Before: 従来のワークアラウンド（型変換を繰り返す書き方）
result := ConvertSlice[int](myStruct.Items())

// After: ジェネリックメソッドを使った書き方
type S struct { /* ... */ }

func (*S) Convert[T any](items []T) []T { /* ... */ }

var s S
s.Convert[int](items)   // 明示的なType Argument
s.Convert(items)        // 型推論による省略
```

**4. メソッド式・メソッド値の活用**

```go
type Reader struct{ /* ... */ }
func (*Reader) Read[E any](p []E) (int, error) { /* ... */ }

// メソッド式はジェネリック関数として得られる
fn := (*Reader).Read  // 型: [E any](*Reader, []E) (int, error)
```

## 議論のハイライト

- **「認識の転換」**: 提案者Robert Griesemerは「コンクリートメソッドはインターフェイスとは独立して有用である」という視点の転換を強調。GoのFAQに「generic methodsは追加しない」と明記されていたが、ジェネリクス導入後の数年の経験を踏まえて方針を変更。

- **インターフェイスとの非連携は設計上の意図**: ジェネリックなコンクリートメソッドがインターフェイスを実装できないことは、新たな制約ではなく既存の型同一性規則の自然な帰結。コアチームはこれを意図的な設計とし、将来ジェネリックインターフェイスメソッドの効率的な実装方法が見つかれば対応できる余地を残す。

- **実装は段階的**: パーサはすでにジェネリックメソッド構文をパースしてエラーを返す実装になっており、そのエラーを取り除くことが出発点。型チェッカーとバックエンドに変更が必要で、最も負荷が高いのはimport/export形式の更新。

- **ツールエコシステムへの影響**: `go/types`の`Signature`型はすでにレシーバとType Parameterのアクセサを持つが、クライアントコードが「どちらか一方しかない」という現在の前提に依存している可能性があり、ツールの対応に1〜2リリースサイクルかかる見込み。

- **反射（reflect）からのアクセス不可**: 未インスタンス化のジェネリック関数が`reflect`でアクセスできないのと同様に、ジェネリックメソッドも`reflect`からはアクセスできない仕様。

## 関連リンク

- [関連Issue #49085](https://github.com/golang/go/issues/49085)
- [関連Issue #50981](https://github.com/golang/go/issues/50981)
- [Proposal Issue](https://github.com/golang/go/issues/77273)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3962620065)
