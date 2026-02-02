---
issue_number: 54297
title: "must: Do"
previous_status: discussions
current_status: hold
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Proposal review meeting minutes · Issue #33502 · golang/go"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "proposal: spec: generic methods for Go · Issue #77273 · golang/go"
    url: https://github.com/golang/go/issues/77273
  - title: "関連Issue #49085: メソッドへの型パラメータ許可"
    url: https://github.com/golang/go/issues/49085
  - title: "proposal: must: Do · Issue #54297 · golang/go"
    url: https://github.com/golang/go/issues/54297
---
## 概要

proposal #54297は、エラーが発生した場合にpanicする汎用ヘルパー関数`must.Do`を標準ライブラリに追加する提案です。元々は`url.MustParse`の追加提案でしたが、より汎用的な`must`パッケージへと議論が発展しました。

## ステータス変更

**discussions** → **hold**

2026年1月28日の提案レビュー会議で、この提案は一旦保留(hold)となりました。保留の理由は、**#77273（ジェネリックメソッドの提案）の解決を待つため**です。ジェネリックメソッドが実装されれば、`func (*testing.T) Must[T any](v T, err error) T`のような、より便利なAPIが実現可能になります。

## 技術的背景

### 現状の問題点

Go言語では、エラーを返す関数を呼び出す際、通常は明示的なエラーチェックが必要です。しかし、初期化コードやグローバル変数の設定など、引数が定数で確実に成功することが分かっている場面では、このボイラープレートコードが冗長になります。

```go
// Before: 従来の書き方（冗長なコード）
import "net/url"

var baseURL *url.URL

func init() {
    u, err := url.Parse("http://example.com/")
    if err != nil {
        panic(err)
    }
    baseURL = u
}

// または
var baseURL = func() *url.URL {
    u, err := url.Parse("http://example.com/")
    if err != nil {
        panic(err)
    }
    return u
}()
```

現在の標準ライブラリには、このパターンを簡潔に書くための`regexp.MustCompile`や`template.Must`などの関数が個別に存在しますが、汎用的な解決策はありません。

### 提案された解決策

Tailscale社が開発した[must.Get](https://pkg.go.dev/tailscale.com/util/must)パターンをベースに、以下のようなシグネチャの汎用関数を標準ライブラリに追加する案が議論されました:

```go
// 値を返す関数用
func Get[T any](v T, err error) T {
    if err != nil {
        panic(err)
    }
    return v
}

// 2つの値を返す関数用
func Get2[T any, U any](v1 T, v2 U, err error) (T, U) {
    if err != nil {
        panic(err)
    }
    return v1, v2
}

// エラーのみを返す関数用
func Do(err error) {
    if err != nil {
        panic(err)
    }
}
```

## これによって何ができるようになるか

グローバル変数の初期化やテストコードにおいて、定数引数で呼び出される関数のエラーハンドリングが大幅に簡潔になります。

### コード例

```go
// After: 新APIを使った書き方
import "errors" // または "must" パッケージ

var baseURL = errors.Must(url.Parse("http://example.com/"))

// テストコードでの使用例
func TestSomething(t *testing.T) {
    data := errors.Must(json.Marshal(testStruct))
    loc := errors.Must(time.LoadLocation("America/New_York"))
    // ...
}

// bytes.Bufferなど、実質的に失敗しない操作での使用
func packData() []byte {
    var buf bytes.Buffer
    errors.Must(0, buf.WriteByte('x'))
    return buf.Bytes()
}
```

## 議論のハイライト

- **パッケージ名の議論**: `must`という新パッケージを作るか、既存の`errors`パッケージに入れるかで意見が分かれました。一つの関数のためだけに新パッケージを作ることへの懸念がありました。

- **`must.Do`の是非**: `must.Do(err)`はエラーハンドリングのショートカットとして悪用される可能性があり、特にmain関数での使用が懸念されました。一方で、`bytes.Buffer`への書き込みなど、実質的に失敗しない操作には有用との意見もありました。

- **使用実態の調査**:
  - Tailscaleの200KLoC規模のコードベースでは、1,265箇所で使用され、最多は`json.Marshal`（18%）でした
  - 定数引数のみでの使用は極めて少なく、ほとんどが動的な引数での使用でした
  - テストコードでの使用が全体の約2/3を占めていました

- **testing.Mustの可能性**: テスト用の`testing.Must(t, err)`または`testing.T.Must[T any](v T, err error) T`の追加も議論されましたが、現在の型システムでは`testing.T.Must`の実装が困難です。ジェネリックメソッドが実装されれば、この問題は解決します。

- **vet検査による制限**: 定数引数のみでの使用を強制する`vet`検査を追加する案も提案されましたが、実際の使用パターンとは合致しないことが判明しました。

- **`url.MustParse`への原点回帰**: 議論の結果、元々の提案だった`url.MustParse`だけを追加する案も再検討されましたが、実際の使用頻度が低いことが分かりました。

## 関連リンク

- [Proposal review meeting minutes · Issue #33502 · golang/go](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [proposal: spec: generic methods for Go · Issue #77273 · golang/go](https://github.com/golang/go/issues/77273)
- [関連Issue #49085: メソッドへの型パラメータ許可](https://github.com/golang/go/issues/49085)
- [proposal: must: Do · Issue #54297 · golang/go](https://github.com/golang/go/issues/54297)
