---
issue_number: 54297
title: "must: Do"
previous_status: discussions
current_status: hold
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Proposal Issue #54297"
    url: https://github.com/golang/go/issues/54297
  - title: "Hold決定のコメント"
    url: https://github.com/golang/go/issues/54297#issuecomment-3814235604
  - title: "Hold理由の説明"
    url: https://github.com/golang/go/issues/54297#issuecomment-3813324889
  - title: "関連提案: ジェネリックメソッド #77273"
    url: https://github.com/golang/go/issues/77273
---

## 要約

## 概要

このproposalは、Go標準ライブラリに汎用的な`must.Do`関数（またはそのバリエーション）を追加することを提案しています。元々は`url.MustParse`という特定の関数の追加提案から始まりましたが、議論を経て、エラーが発生した場合にpanicする汎用ヘルパー関数の必要性へと発展しました。

## ステータス変更

**discussions** → **hold**

この決定は、#77273（ジェネリックメソッドの提案）の解決を待つために行われました。ジェネリックメソッドが実装されれば、`func (*testing.T) Must[T any](v T, err error) T`のような形式でより洗練されたAPIを提供できる可能性があるためです。proposal review groupは、この機能が近い将来実現可能かどうかを見極めてから本proposalの判断を下すことを決定しました。

## 技術的背景

### 現状の問題点

Go言語には`regexp.MustCompile`や`template.Must`のように、エラー時にpanicする「Must」系の関数が既に存在しますが、これらは各パッケージ固有の実装です。開発者が定数引数やテストコード、初期化コードなどで「絶対に失敗しないはず」の処理を書く際、以下のような冗長なコードが必要になります：

```go
// url.Parseを使う場合の典型的なボイラープレート
u, err := url.Parse("http://example.com/")
if err != nil {
    panic(err)
}
// uを使った処理...
```

また、パッケージレベルの変数初期化では、さらに工夫が必要です：

```go
var baseURL *url.URL

func init() {
    var err error
    baseURL, err = url.Parse("http://example.com/")
    if err != nil {
        panic(err)
    }
}
```

### 提案された解決策

議論では主に3つのアプローチが検討されました：

1. **個別のMust関数**: `url.MustParse`のような、パッケージ固有の関数を追加
2. **汎用的なmust.Get関数**: ジェネリクスを使った汎用ヘルパー
3. **testing.Must**: テストコード専用のヘルパー

最も支持を集めたのは、Tailscaleで実際に使用されている以下のような実装です：

```go
package must

func Get[T any](v T, err error) T {
    if err != nil {
        panic(err)
    }
    return v
}
```

## これによって何ができるようになるか

この機能により、エラー処理のボイラープレートを削減し、より簡潔なコードを書けるようになります。特に以下のユースケースで有用です：

1. **パッケージレベルの定数的な初期化**
2. **テストコードでの前提条件の設定**
3. **ワンオフスクリプトやサンプルコード**

### コード例

```go
// Before: 従来の書き方
var baseURL *url.URL

func init() {
    var err error
    baseURL, err = url.Parse("http://example.com/")
    if err != nil {
        panic(err)
    }
}

// After: must.Getを使った書き方
var baseURL = must.Get(url.Parse("http://example.com/"))
```

```go
// Before: テストでの典型的なセットアップ
func TestSomething(t *testing.T) {
    u, err := url.Parse("http://example.com/path")
    if err != nil {
        t.Fatal(err)
    }
    // uを使ったテスト...
}

// After: よりシンプルな記述
func TestSomething(t *testing.T) {
    u := must.Get(url.Parse("http://example.com/path"))
    // uを使ったテスト...
}
```

## 議論のハイライト

- **実用性の証明**: Tailscaleのコードベースで1265箇所、主に`json.Marshal`（18%）、`http.NewRequestWithContext`（8%）、`json.Unmarshal`（7%）などで使用されている実績が報告されました

- **濫用への懸念**: `must.Do(err)`の形式が通常のエラー処理の代替として誤用される可能性について議論されました。Ian Lance Taylor氏は「エラー処理メカニズムとして使われ始めると非常に残念」と指摘しています

- **パッケージ配置の議論**: 新しい`must`パッケージを作るべきか、既存の`errors`パッケージに追加すべきかで意見が分かれました。「たった1つの関数のために新しいパッケージを作るのは概念的な負担が大きい」という意見もありました

- **定数引数への制限案**: Josh Bleecher Snyder氏から「`Do`を追加するが、vetで定数引数の場合のみ使用可能にする」というアイデアが提案されました。これにより「明らかに正しい使い方」に制限できる可能性があります

- **testing.Mustの位置づけ**: テスト用のアサーション機能との境界線について議論されました。Alan Donovan氏は「プロダクションコードでの`must`とテストでのアサーションは異なる要件を持つ」と指摘し、テスト用の`Must`追加には慎重な姿勢を示しました

- **url.MustParseの実用性**: 提案者のBrad Fitzpatrick氏自身が調査したところ、Tailscaleのコードベースでは`url.MustParse`が役立つケースはあまり多くなく、汎用的な`must.Get`の方がはるかに有用であることが判明しました

## 関連リンク

- [Proposal Issue #54297](https://github.com/golang/go/issues/54297)
- [Hold決定のコメント](https://github.com/golang/go/issues/54297#issuecomment-3814235604)
- [Hold理由の説明](https://github.com/golang/go/issues/54297#issuecomment-3813324889)
- [関連提案: ジェネリックメソッド #77273](https://github.com/golang/go/issues/77273)
- [Tailscaleの実装例](https://pkg.go.dev/tailscale.com/util/must)
- [DoltHub Blog: Golang Panic Recovery (2026)](https://www.dolthub.com/blog/2026-01-09-golang-panic-recovery/)
- [Go by Example: Panic](https://gobyexample.com/panic)

## 関連リンク

- [Proposal Issue #54297](https://github.com/golang/go/issues/54297)
- [Hold決定のコメント](https://github.com/golang/go/issues/54297#issuecomment-3814235604)
- [Hold理由の説明](https://github.com/golang/go/issues/54297#issuecomment-3813324889)
- [関連提案: ジェネリックメソッド #77273](https://github.com/golang/go/issues/77273)
