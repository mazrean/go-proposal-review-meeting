---
issue_number: 76821
title: "math/big: add Rat.{Floor,Ceil} methods"
previous_status: 
current_status: active
changed_at: 2026-02-11T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3886687081
related_issues:
  - title: "Related: proposal: intmath: new package #51563"
    url: https://github.com/golang/go/issues/51563
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/76821
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3886687081
---
## 概要
`math/big.Rat`に床関数（Floor）と天井関数（Ceil）のメソッドを追加し、合理的な数値の整数部分を`*big.Int`として取得可能にする提案。さらに、`big.Int`にも床除算・天井除算のメソッド（FloorDiv、CeilDiv、およびそれぞれのMod版）を追加することで、より汎用的な整数演算をサポートする。

## ステータス変更
**(ステータスなし)** → **active**

2026年2月11日のProposal Review Meetingで、この提案がアクティブ状態に移行されました。提案自体は2025年12月に投稿され、その後議論を経て機能範囲が拡張されています。当初は`Rat.Floor`と`Rat.Ceil`のみの提案でしたが、コミュニティの議論により`Int`型にも床除算・天井除算メソッドを追加する方向に発展しました。これにより、より一般的で強力なAPIとなっています。

## 技術的背景

### 現状の問題点
現在、Goの`math/big`パッケージで有理数（`Rat`）の床や天井を求めるには、効率的で正確な方法がありません。既存のワークアラウンドとしては、以下のような方法が使われていました。

```go
// 既存のワークアラウンド（精度を保つが非効率）
// Floor
rat.Sub(rat, big.NewRat(1, 2))
rat.SetString(rat.FloatString(0))

// Ceil
rat.Add(rat, big.NewRat(1, 2))
rat.SetString(rat.FloatString(0))
```

この方法は精度は保てますが、文字列変換を伴うため非効率です。また、`float64`への変換を経由する方法もありますが、これは`big.Rat`を使う目的（任意精度演算）を損なう可能性があります。

### 提案された解決策
提案では、以下の2つのレイヤーでAPIを追加します。

**1. `big.Rat`への便利メソッド（利便性重視）**

```go
// Floor returns ⌊x⌋ (the floor of x) as a new [Int].
func (x *Rat) Floor() *Int

// Ceil returns ⌈x⌉ (the ceiling of x) as a new [Int].
func (x *Rat) Ceil() *Int
```

**2. `big.Int`への汎用的な除算メソッド（汎用性重視）**

```go
// FloorDiv sets z to ⌊x/y⌋ for y != 0 and returns z.
func (z *Int) FloorDiv(x, y *Int) *Int

// FloorDivMod sets z to ⌊x/y⌋ and m to x mod y
// for y != 0 and returns the pair (z, m).
func (z *Int) FloorDivMod(x, y, m *Int) (*Int, *Int)

// CeilDiv sets z to ⌈x/y⌉ for y != 0 and returns z.
func (z *Int) CeilDiv(x, y *Int) *Int

// CeilDivMod sets z to ⌈x/y⌉ and m to x mod y
// for y != 0 and returns the pair (z, m).
func (z *Int) CeilDivMod(x, y, m *Int) (*Int, *Int)
```

これらのメソッドは、既存の`Quo`/`QuoRem`（切り捨て除算）、`Div`/`DivMod`（ユークリッド除算）と対をなし、`big.Int`の除算機能を完全なものにします。

## これによって何ができるようになるか

数学的な計算やアルゴリズムで頻繁に必要となる床・天井演算を、任意精度で効率的に実行できるようになります。特に、有理数を扱う数値計算、暗号アルゴリズム、数論的計算などで威力を発揮します。

### コード例

```go
// Before: Engel展開の実装（既存のワークアラウンド使用）
func ToEngel(u *big.Rat) (seq []*big.Int) {
    one := big.NewRat(1, 1)
    tmp := new(big.Rat)
    for {
        // 天井を求めるには複雑な操作が必要
        tmp.Inv(u)
        // ここで何らかのワークアラウンドが必要...
        // ...
    }
    return seq
}

// After: 新APIを使った実装
func ToEngel(u *big.Rat) (seq []*big.Int) {
    one := big.NewRat(1, 1)
    tmp := new(big.Rat)
    for {
        a := tmp.Inv(u).Ceil()  // 天井を直接取得
        seq = append(seq, a)
        u.Mul(u, tmp.SetInt(a))
        u.Sub(u, one)
        if u.Num().Sign() == 0 {
            break
        }
    }
    return seq
}
```

Engel展開は、正の有理数を単位分数の特殊な形式で表現する数学的手法で、このような実装が簡潔かつ効率的に書けるようになります（[Engel expansion - Wikipedia](https://en.wikipedia.org/wiki/Engel_expansion)参照）。

また、`big.Int`のメソッドは、`Rat`を経由せずに直接整数の床除算・天井除算が可能です。

```go
// 負の除数にも対応した床除算
result := new(big.Int).FloorDiv(x, y)

// ページネーション計算など
pages := new(big.Int).CeilDiv(items, itemsPerPage)
```

## 議論のハイライト

- **当初は`Rat`のみの提案だったが、議論を経て`Int`への追加も決定**: adonovan氏の指摘により、床・天井は本質的に除算の一種であるため、`Int.CeilDiv`/`Int.FloorDiv`という名前で`Int`にも追加する方針に。これにより`Rat`を経由しない汎用的な使用が可能になった
- **既存の除算メソッドとの一貫性**: `big.Int`には既に`Quo`/`QuoRem`（切り捨て除算）、`Div`/`DivMod`（ユークリッド除算）があり、今回の追加により床除算・天井除算も揃うことで、除算の種類が完全に網羅される
- **Modを返すバリアントも提供**: 除算結果だけでなく剰余も同時に取得できる`FloorDivMod`/`CeilDivMod`も提供。これは既存の`QuoRem`/`DivMod`と同様のパターン
- **両方のレイヤーを提供することに合意**: `Int`の汎用メソッドと`Rat`の便利メソッドの両方を提供することで、汎用性と利便性を両立（adonovan氏の提案による）
- **実装PR（#76820）は既に作成済み**: 提案と同時に実装も進められており、具体的なコードレビューが可能な状態

## 関連リンク

- [Related: proposal: intmath: new package #51563](https://github.com/golang/go/issues/51563)
- [Proposal Issue](https://github.com/golang/go/issues/76821)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3886687081)
