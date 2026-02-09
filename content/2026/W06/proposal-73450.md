---
issue_number: 73450
title: "net/url: URL.Clone, Values.Clone"
previous_status: active
current_status: likely_accept
changed_at: 2026-02-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3872311559
related_issues:
  - title: "Proposal Issue #73450"
    url: https://github.com/golang/go/issues/73450
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/73450#issuecomment-3872310073
  - title: "過去の提案 #41733"
    url: https://github.com/golang/go/issues/41733
  - title: "URLコピーに関する議論 #38351"
    url: https://github.com/golang/go/issues/38351
---
## 概要
`net/url`パッケージに`URL.Clone()`と`Values.Clone()`メソッドを追加し、URLとクエリパラメータを安全かつ効率的にコピーできるようにする提案です。

## ステータス変更
**active** → **likely_accept**

2026年2月9日のProposal Review Meetingで「likely accept」と判定されました。議論の結果、多くの開発者が非効率的な方法でURLをコピーしている実態が明らかになり、標準的なCloneメソッドの必要性が認められました。特に、net/httpパッケージ内部に既に`cloneURL`関数が存在しており、コミュニティからの需要が確認されたことが承認の決め手となっています。

## 技術的背景

### 現状の問題点
`net/url.URL`を安全にコピーする方法が明確ではなく、多くの開発者が以下のような非効率な方法を使用しています:

```go
// 非効率な方法: シリアライズ/デシリアライズによるコピー
u2, _ := url.Parse(u1.String())
```

GitHub上のコード調査によると、約3,300件のコードがこの方法を使用しており、これは単純なコピー（約21,200件）の約1/8に相当します。この方法は本来不要なパース処理とエラーハンドリングを必要とします。

技術的には`u2 := *u1`で安全にコピーできますが、`URL`構造体に`User *Userinfo`というポインタフィールドがあるため、深いコピーが必要かどうか混乱を招いています。`Userinfo`は「不変（immutable）」と文書化されていますが、この事実はドキュメントを深く読まないと分かりません。

### 提案された解決策
以下の2つのメソッドを追加します:

```go
// Clone returns a copy of u.
func (u *URL) Clone() *URL

// Clone returns a copy of v.
func (v Values) Clone() Values
```

`URL.Clone()`は、`User`フィールドが存在する場合は深いコピーを行います。これにより、一方のクローンへの変更が他方に影響しないことが保証されます。実装は既にnet/http内部で使われている`cloneURL`関数と同等になります。

## これによって何ができるようになるか

ベースとなるURLから複数のバリエーションを安全かつ効率的に作成できるようになります。これは、設定から読み込んだURLをもとに複数のAPIエンドポイントにリクエストする場面などで特に有用です。

### コード例

```go
// Before: 従来の書き方（非効率）
base, _ := url.Parse("https://api.example.com/v1")
endpoint1, _ := url.Parse(base.String())  // パース処理が発生
endpoint1.Path = path.Join(endpoint1.Path, "users")

endpoint2, _ := url.Parse(base.String())  // 再度パース処理
endpoint2.Path = path.Join(endpoint2.Path, "posts")

// After: 新APIを使った書き方
base, _ := url.Parse("https://api.example.com/v1")
endpoint1 := base.Clone()
endpoint1.Path = path.Join(endpoint1.Path, "users")

endpoint2 := base.Clone()
endpoint2.Path = path.Join(endpoint2.Path, "posts")

// Values.Clone()の使用例
baseParams := url.Values{"key": []string{"value"}}
params1 := baseParams.Clone()
params1.Set("page", "1")

params2 := baseParams.Clone()
params2.Set("page", "2")
```

## 議論のハイライト

- **UserInfoフィールドの扱い**: 当初は`u2 := *u1`で十分という意見もありましたが、net/httpの内部実装が既に`User`フィールドを深くコピーしていることから、最終的にはUserInfoも含めて深いコピーを行う実装が支持されました
- **過去の提案との関係**: 2020年の提案#41733では「多くの型に適用できる操作だからURL特有ではない」として却下されましたが、今回は実際の使用状況データ（GitHub検索結果）により需要が実証されました
- **Values.Cloneの必要性**: `Values`は`map[string][]string`という入れ子構造のため、単純なコピーでは深いクローンができません。`http.Header`に既に`.Clone()`が存在しており、同様の構造を持つ`Values`にも同じメソッドがあるべきという一貫性の観点からも支持されました
- **パフォーマンスへの影響**: ほとんどのURLには`User`フィールドがないため、追加のアロケーションが発生するケースは稀です。一方で、確実に変更の副作用がないという安全性のメリットが大きいと判断されました
- **承認タイミング**: 2025年4月の提案から約10ヶ月後の承認となりました。実装案の明確化（特にUserInfoの扱い）と、コアチームメンバー（@neild）による詳細な分析コメントが決定を後押ししました

## 関連リンク

- [Proposal Issue #73450](https://github.com/golang/go/issues/73450)
- [Review Minutes](https://github.com/golang/go/issues/73450#issuecomment-3872310073)
- [過去の提案 #41733](https://github.com/golang/go/issues/41733)
- [URLコピーに関する議論 #38351](https://github.com/golang/go/issues/38351)
