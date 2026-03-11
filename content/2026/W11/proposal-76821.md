---
issue_number: 76821
title: "math/big: add Rat.{Floor,Ceil} methods"
previous_status: active
current_status: likely_accept
changed_at: 2026-03-11T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4042167102
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/76821
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4042167102
---
## 概要

`math/big` パッケージの `big.Int` 型に、丸めモードを指定できる汎用的な整数除算メソッド `Divide` と、便利なエイリアス定数（`Trunc`, `Floor`, `Round`, `Ceil`）を追加するプロポーザルです。当初は `Rat.Floor` / `Rat.Ceil` メソッドの追加として始まりましたが、議論を経てより汎用的な設計へと発展しました。

## ステータス変更

**active** → **likely_accept**

2026年3月11日の週次プロポーザルレビューミーティングにて、`aclements` を中心としたコアチームが最終的な API 設計について合意に達し、`likely_accept` に移行しました。当初提案のシンプルな `Rat.Floor()`/`Rat.Ceil()` から、`big.Int` 上の汎用的な丸めモード付き `Divide` メソッドへと設計が進化したことで、より広いユースケースに対応できると判断されました。

## 技術的背景

### 現状の問題点

`math/big` パッケージには有理数 `big.Rat` を整数に丸める標準的な手段がありません。例えば有理数 `3/2` の天井値（切り上げ）を求めるには、分子・分母を取り出して手作業で計算する必要があります。

```go
// Before: 天井値を手動で計算する（冗長なワークアラウンド）
func ceilRat(x *big.Rat) *big.Int {
    n := x.Num()
    d := new(big.Int).Neg(x.Denom()) // 負にして Div を流用
    return new(big.Int).Div(new(big.Int).Neg(n), d)
}
```

また `big.Int` には切り捨て除算（`Quo`/`QuoRem`）とユークリッド除算（`Div`/`DivMod`）はあるものの、床除算（floor division）や天井除算（ceiling division）に対応するメソッドが存在しませんでした。

### 提案された解決策

`big.Int` に `Divide` メソッドを追加し、既存の `RoundingMode` 型を使って丸めモードを指定できるようにします。また可読性向上のための便利な定数エイリアスも追加されます。

```go
// Divide は整数商 q と余り r を求めます:
//   q = f(x/y)
//   r = x - y*q
// mode は Trunc, Floor, Round, Ceil のいずれかを指定します。
// z が nil の場合は商を計算せず余りのみ返します。
// r が nil の場合は余りを計算しません。
func (z *Int) Divide(x, y, r *Int, mode RoundingMode) (*Int, *Int)

const (
    Trunc = ToZero        // ゼロ方向丸め（Goの/演算子と同じ）
    Floor = ToNegativeInf // 負の無限大方向丸め
    Round = ToNearestEven // 最近偶数丸め（IEEE 754デフォルト）
    Ceil  = ToPositiveInf // 正の無限大方向丸め
)
```

## これによって何ができるようになるか

### コード例

```go
// Before: Engel展開の計算（ワークアラウンド版）
func ToEngel(u *big.Rat) (seq []*big.Int) {
    one := big.NewRat(1, 1)
    tmp := new(big.Rat)
    for {
        // 天井値を手動計算: ⌈1/u⌉ = ⌈denom/num⌉
        inv := tmp.Inv(u)
        n := new(big.Int).Neg(inv.Num())
        d := new(big.Int).Neg(inv.Denom())
        a := new(big.Int).Div(n, d)
        seq = append(seq, a)
        u.Mul(u, tmp.SetInt(a))
        u.Sub(u, one)
        if u.Num().Sign() == 0 {
            break
        }
    }
    return seq
}

// After: Divide メソッドを使った簡潔な記述
func ToEngel(u *big.Rat) (seq []*big.Int) {
    one := big.NewRat(1, 1)
    tmp := new(big.Rat)
    for {
        inv := tmp.Inv(u)
        // ⌈denom/num⌉ を Ceil モードで直接計算
        a, _ := new(big.Int).Divide(inv.Num(), inv.Denom(), nil, big.Ceil)
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

具体的なユースケースとしては以下が挙げられます:

- **浮動小数点数テーブル生成**: `rsc/fpfmt` のように `10^n * 2^m` の 128 ビット近似値テーブルを生成する際に、天井値除算を2段階行う処理が自然に記述できます
- **暗号・数論アルゴリズム**: 有理数の床・天井を多用する数論的計算（連分数、Engel展開など）での記述が簡潔になります
- **固定小数点演算**: 分数の商を特定の丸めモードで整数に変換する処理が明確になります

## 議論のハイライト

- **設計の大きな転換**: 最初に提案された `func (x *Rat) Floor() *Int` というシンプルな API は、@aclements の「`*Rat` の慣用パターン（レシーバに代入して返す）に合わない」という指摘と、@griesemer の「`big.Int` に直接追加する方がより汎用的」という意見を経て、`Int.Divide` という統一インターフェースへと発展しました
- **命名論争**: `FloorDiv` か `FloorQuo` かについて活発な議論が行われました。@griesemer は既存の `Quo`（切り捨て）との一貫性から `FloorQuo` を提案しましたが、最終的には単一の `Divide` メソッドに集約されることで解消されました
- **`Round` モードの追加**: @aclements が IEEE 754 の4つの標準丸めモード（Trunc/Floor/Round/Ceil）のうち3つだけ実装するのは不自然と指摘し、最近偶数丸め（`ToNearestEven`）も含めることが決定されました
- **既存 `RoundingMode` 型の再利用**: @aclements の提案により、`math/big` にすでに存在する `RoundingMode` 型と同じ定数を流用することで、パッケージ内の一貫性が保たれることになりました
- **`z` の optional 化**: 商が不要な場合（余りのみ取得したい場合）に `z` を `nil` にできるよう設計が改良され、不要な計算コストを回避できるようになりました

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/76821)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4042167102)
