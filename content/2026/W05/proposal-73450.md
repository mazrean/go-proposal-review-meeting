---
issue_number: 73450
title: "net/url: URL.Clone, Values.Clone"
previous_status: discussions
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "関連Issue #38351: URL deep-copyingの安全性確認"
    url: https://github.com/golang/go/issues/38351
  - title: "関連Issue #41733: 前回のURL.Clone提案（2020年、却下）"
    url: https://github.com/golang/go/issues/41733
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/73450
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
---

## 要約

## 概要
`net/url`パッケージの`URL`と`Values`型にそれぞれ`Clone()`メソッドを追加するproposalです。現在、URLを安全かつ効率的にコピーする方法が明確でないため、多くの開発者が非効率なString/Parseのround-tripパターンを使用している問題を解決します。

## ステータス変更
**(新規)** → **active**

2026年1月28日のProposal Review Meetingで、このproposalは議論対象として正式にactive columnに追加されました（"added to minutes"）。過去に同様のproposal #41733が提案されたものの、使用頻度への疑問から却下されていましたが、今回は実際のコード検索データ（GitHub上で21,200件のshallow copy、3,300件のString/Parse pattern）を示すことで、問題の実在性を証明しています。

## 技術的背景

### 現状の問題点

`net/url.URL`をコピーする際、以下の混乱が生じています:

1. **安全かつ効率的な方法が不明瞭**: `url2 := *url1`が正しい方法ですが、`User *Userinfo`フィールドがポインタであることから、deep copyが必要かどうか判断に迷う開発者が多数います。

2. **非効率なパターンの蔓延**: 約1/8の開発者が不必要にシリアライズ/デシリアライズを行っています:
   ```go
   url2, _ := url.Parse(url1.String())
   ```
   このパターンは、常に`nil`であるべきエラーを処理する必要があり、パフォーマンスも劣ります。

3. **net/http内部との乖離**: `net/http`パッケージ内部では、`cloneURL`という関数が存在し、`*Userinfo`まで含めてdeep copyを行っています。しかしこれは公開APIではなく、一般の開発者は利用できません。

4. **Values型の深いコピーの困難さ**: `Values`は`map[string][]string`という入れ子構造のため、単純な代入では浅いコピーになり、スライスの要素が共有されてしまいます。現在は148件のコードが`ParseQuery(v.Encode())`パターンを使用しています。

### 提案された解決策

以下の2つのメソッドを追加します:

```go
// Clone returns a copy of u
func (u *URL) Clone() *URL

// Clone returns a copy of v
func (v Values) Clone() Values
```

`URL.Clone()`の実装は、`net/http.cloneURL`と同様に、`*Userinfo`フィールドも含めた完全なコピーを行います。これにより、「Userinfoは不変だからshallow copyで良い」という微妙な前提知識に依存せず、確実に安全なコピーが得られます。

## これによって何ができるようになるか

ベースURLを使った複数リクエストの生成パターンが、明瞭かつ効率的になります。このパターンは、設定から取得したベースURLに対して、個別のパスやクエリパラメータを追加してリクエストを送る場合によく使われます。

### コード例

```go
// Before: 従来の書き方（非効率なString/Parseパターン）
base, _ := url.Parse("http://api.example.com/v1")

func DoRequest() error {
  u, err := url.Parse(base.String()) // 不要なエラーハンドリング
  if err != nil {
    return err // 実際には発生しないエラー
  }
  u.Path = path.Join(u.Path, "users")
  // uを使ってリクエスト
}

// After: 新APIを使った書き方
base, _ := url.Parse("http://api.example.com/v1")

func DoRequest() error {
  u := base.Clone() // 明確かつ効率的
  u.Path = path.Join(u.Path, "users")
  // uを使ってリクエスト
}
```

`Values`についても同様に、クエリパラメータのベースセットから派生パターンを作る際に便利です:

```go
// Before: Encode/ParseQueryのround-trip
baseParams, _ := url.ParseQuery("limit=10&sort=asc")
derivedParams, _ := url.ParseQuery(baseParams.Encode())
derivedParams.Set("offset", "20")

// After: 明瞭なClone
baseParams, _ := url.ParseQuery("limit=10&sort=asc")
derivedParams := baseParams.Clone()
derivedParams.Set("offset", "20")
```

## 議論のハイライト

- **過去の拒否理由と今回の違い**: 2020年の#41733では、Russ Cox氏が「使用頻度が不明」「`u2 := *u`で十分」という理由で反対しましたが、今回は具体的な使用データ（21,200件 vs 3,300件）により実需を証明しています。

- **Userinfoのコピー戦略**: Damien Neil氏（@neild）は、「`*Userinfo`は不変だが、理論上は`*u2.User = *url.User("username")`で変更可能。確実性のためdeep copyすべき」という立場を明確にしました。`http.Header.Clone()`との一貫性も考慮されています。

- **`http.Header`との並行性**: `http.Header`（実質的に`map[string][]string`）は既に`Clone()`メソッドを持ち、`url.Values`も同じ構造であるため、APIの一貫性の観点からも`Values.Clone()`は妥当です。

- **新しい`new(expr)`構文との関係**: Issue #45624で提案されている`new(*u)`構文が実装されれば、`Clone()`の実装がより簡潔になります（`u2 := new(*u)`）。ただし、`Userinfo`のコピーは依然として必要です。

- **ドキュメントvs API**: 「ドキュメントで説明するだけで十分」という意見もありましたが、GitHubの実データは明確にAPIレベルのサポートが必要であることを示しています。

## 関連リンク
- [Proposal Issue](https://github.com/golang/go/issues/73450)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連Issue #38351: URL deep-copyingの安全性確認](https://github.com/golang/go/issues/38351)
- [関連Issue #41733: 前回のURL.Clone提案（2020年、却下）](https://github.com/golang/go/issues/41733)
- [net/http内部のcloneURL実装](https://github.com/golang/go/blob/go1.19/src/net/http/clone.go#L22)
- [http.Header.Clone() documentation](https://pkg.go.dev/net/http)

## 関連リンク

- [関連Issue #38351: URL deep-copyingの安全性確認](https://github.com/golang/go/issues/38351)
- [関連Issue #41733: 前回のURL.Clone提案（2020年、却下）](https://github.com/golang/go/issues/41733)
- [Proposal Issue](https://github.com/golang/go/issues/73450)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
