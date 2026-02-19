---
issue_number: 73450
title: "net/url: URL.Clone, Values.Clone"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-02-18T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3923200976
related_issues:
  - title: "Proposal Issue #73450"
    url: https://github.com/golang/go/issues/73450
  - title: "Review Minutes (2026-02-18)"
    url: https://github.com/golang/go/issues/33502#issuecomment-3923200976
  - title: "関連Issue #41733: 過去の URL.Clone 提案（却下済み）"
    url: https://github.com/golang/go/issues/41733
  - title: "関連Issue #38351: URL の安全なコピー方法の明確化"
    url: https://github.com/golang/go/issues/38351
---
## 概要

`net/url` パッケージに `URL.Clone()` メソッドと `Values.Clone()` メソッドを追加するproposalです。URLおよびクエリパラメータのコピーを安全かつ明示的に行うための標準的なAPIを提供します。

## ステータス変更
**likely_accept** → **accepted**

2026年2月18日に開催されたProposal Review Meetingにおいて、@aclements、@cherrymui、@griesemer、@ianlancetaylor、@neild、@rolandshoemaker が参加し審査が行われました。「コンセンサスに変更なし」として正式に承認されました。`likely_accept` への変更（2026年2月9日）から異論が出なかったため、そのままacceptedに移行しています。

## 技術的背景

### 現状の問題点

`net/url.URL` 構造体には `User *Userinfo` というポインタフィールドが含まれているため、単純なコピーが安全かどうか開発者には分かりにくい状況でした。`Userinfo` 型はドキュメント上「immutable（不変）」とされていますが、その事実を知らない開発者が多く、以下のような非効率なワークアラウンドが広く使われていました。

```go
// 問題のあるパターン: シリアライズ/デシリアライズのコスト
u2, _ := url.Parse(u1.String())
```

GitHub上の検索結果によると、URLのコピーパターンとして：
- `u2 := *u1`（シャローコピー）: 約21,200件
- `url.Parse(u.String())`（文字列経由の再パース）: 約3,300件

つまり全体の約1/8のユーザーが不必要なシリアライズ/デシリアライズコストを支払っていることが示されています。また `Values`（`map[string][]string` のエイリアス）についても `ParseQuery(v.Encode())` というワークアラウンドが148件確認されています。

`net/http` パッケージ内部では既に `cloneURL` という非公開関数が存在し、`Request.Clone` などで使用されていましたが、これが公開APIとして提供されていませんでした。

### 提案された解決策

以下の2つのメソッドを `net/url` パッケージに追加します。

```go
// Clone returns a copy of u.
func (u *URL) Clone() *URL

// Clone returns a copy of v.
func (v Values) Clone() Values
```

`URL.Clone()` の実装は、`User` フィールド（`*Userinfo`）が存在する場合にその深いコピーも行います。これにより、あるクローンへの変更が元のURLに影響を与えないことが保証されます。`Values.Clone()` は `map[string][]string` の入れ子構造を持つため、深いコピーが必要です。

## これによって何ができるようになるか

安全で効率的なURLのコピーを、明示的かつ慣用的な方法で行えるようになります。

### コード例

```go
// Before: 従来のワークアラウンド（再パース）
base, _ := url.Parse("https://api.example.com/v1")
u, _ := url.Parse(base.String()) // シリアライズ/デシリアライズのコストが発生
u.Path = u.Path + "/users"

// Before: シャローコピー（安全だが分かりにくい）
u2 := *base
u2.Path = u2.Path + "/items"

// After: URL.Clone() を使った明示的なコピー
u3 := base.Clone()
u3.Path = u3.Path + "/users"

u4 := base.Clone()
u4.Path = u4.Path + "/items"
```

```go
// Before: Values のコピー（再エンコード/再パース）
original := url.Values{"key": {"value1", "value2"}}
copied, _ := url.ParseQuery(original.Encode())

// After: Values.Clone() を使った深いコピー
copied := original.Clone()
copied.Set("key", "modified")
// original は変更されない
```

実践的なユースケースとして以下が挙げられます。

1. **ベースURLからの複数エンドポイント生成**: 設定から読み込んだベースURLをクローンしてパスやクエリを変更し、複数のAPIエンドポイントURLを生成する。
2. **HTTPクライアントでのリクエスト毎のURL操作**: ベースURLを保持しつつ、リクエスト毎に異なるパスやクエリパラメータを安全に設定する。
3. **クエリパラメータのバリエーション生成**: フィルタ条件が異なる複数のURLを元のクエリパラメータから効率的に生成する。

## 議論のハイライト

- **過去の提案（#41733）との関係**: 2020年に同様の提案（#41733）が行われたが、当時は「URL cloneの頻度が不明」「他の型にも同様のCloneが必要になる」という @rsc の懸念により却下された。今回の提案（#73450）では GitHub コード検索による実証データを示すことで、現実の需要を定量的に証明した点が決定的な違いとなった。
- **`Userinfo` の深いコピーについての議論**: `URL.Clone()` が `*Userinfo` フィールドを深いコピーすべきか否かが議論された。@neild は「ほとんどのURLにUserinfoはないため追加コストは小さく、確実性のために深いコピーが適切」と結論付け、これが採用された。
- **`http.Header.Clone()` との一貫性**: `http.Header`（`map[string][]string`）には既に `Clone()` が存在しており、同じ型エイリアスである `url.Values` に `Clone()` がないことは一貫性を欠くという指摘があった（@earthboundkid）。
- **実装CLの作成**: 承認直後の2026年2月18日に実装CL（[CL 746800](https://go.dev/cl/746800)・[CL 746801](https://go.dev/cl/746801)）が投稿され、迅速な実装が開始された。

## 関連リンク

- [Proposal Issue #73450](https://github.com/golang/go/issues/73450)
- [Review Minutes (2026-02-18)](https://github.com/golang/go/issues/33502#issuecomment-3923200976)
- [関連Issue #41733: 過去の URL.Clone 提案（却下済み）](https://github.com/golang/go/issues/41733)
- [関連Issue #38351: URL の安全なコピー方法の明確化](https://github.com/golang/go/issues/38351)
