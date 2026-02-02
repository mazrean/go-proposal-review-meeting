---
issue_number: 73450
title: "net/url: URL.Clone, Values.Clone"
previous_status: 
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "proposal: net/url: URL.Clone, Values.Clone · Issue #73450"
    url: https://github.com/golang/go/issues/73450
  - title: "Review Minutes (Active指定)"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236325
  - title: "proposal: net/url: add URL.Clone method · Issue #41733"
    url: https://github.com/golang/go/issues/41733
  - title: "net/url: clarify safe way to do URL deep-copying · Issue #38351"
    url: https://github.com/golang/go/issues/38351
  - title: "new(*expr)構文提案 #45624"
    url: https://github.com/golang/go/issues/45624
---

## 要約

## 概要
`net/url`パッケージに`URL.Clone()`と`Values.Clone()`メソッドを追加する提案です。現状、URLの安全なコピー方法が明確でないため、多くの開発者が非効率な`Parse(url.String())`パターンを使用しています。この提案は、標準ライブラリに明示的で効率的なクローン機能を追加することで、開発者の混乱を解消し、パフォーマンスを向上させることを目指しています。

## ステータス変更
**(提案なし)** → **active**

2026年1月28日、proposal review groupによって「active」ステータスに移行されました。これは、過去の提案（#41733）が「Clone は必要ない」として却下された経緯がある中で、今回は実データに基づく証拠（GitHubコード検索で21,200件のシンプルコピー、3,300件の非効率なParse/Stringパターン）を提示したことが評価されたためです。

## 技術的背景

### 現状の問題点

`net/url.URL`をコピーする際、以下の混乱が存在します:

1. **シンプルなコピーが本当に安全か不明瞭**
   ```go
   u1, _ := url.Parse("https://example.com/")
   u2 := *u1  // これは安全？
   ```

   `URL`構造体には`User *Userinfo`というポインタフィールドがあるため、シャローコピーで問題ないか判断に迷います。実際には`Userinfo`は不変型（immutable）と文書化されているため安全ですが、これを知らない開発者も多くいます。

2. **結果として非効率なワークアラウンドが蔓延**
   ```go
   // 1/8の開発者がこのような非効率な方法を使用
   u2, _ := url.Parse(u1.String())  // シリアライズ→パース
   ```

   GitHubコード検索によると、約3,300件のコードがこのパターンを使用しており、不要な文字列変換とパースのコストを支払っています。

3. **`net/http`内部には既に実装が存在**

   標準ライブラリの`net/http`パッケージは内部的に`cloneURL`関数を持っています:
   ```go
   func cloneURL(u *url.URL) *url.URL {
       if u == nil {
           return nil
       }
       u2 := new(url.URL)
       *u2 = *u
       if u.User != nil {
           u2.User = new(url.Userinfo)
           *u2.User = *u.User
       }
       return u2
   }
   ```

   この実装は`Request.Clone()`などで使用されていますが、公開APIではないため、一部のパッケージは`//go:linkname`を使ってアクセスしている状況です。

### 提案された解決策

以下の2つのメソッドを追加します:

```go
// Clone returns a copy of u
func (u *URL) Clone() *URL

// Clone returns a copy of v
func (v Values) Clone() Values
```

**`URL.Clone()`の実装方針:**
- 基本的には構造体のシャローコピー
- ただし`User`フィールドが存在する場合は、それも新しくアロケートしてコピー
- これにより「一方のクローンへの変更が他方に影響しない」ことを保証

**`Values.Clone()`の必要性:**
- `url.Values`は`map[string][]string`という入れ子構造
- シンプルなコピーでは内部のスライスが共有されてしまう
- `http.Header`には既に`.Clone()`メソッドが存在
- 実は`url.Values`と`http.Header`は同じ型表現（`map[string][]string`）なので、内部実装では`Header`の`Clone`を利用可能

## これによって何ができるようになるか

この提案により、以下のような「ベースURLから派生URLを作成する」パターンが明確かつ効率的に書けるようになります:

### コード例

```go
// Before: 従来の書き方（混乱を招く or 非効率）

// パターン1: シャローコピー（これが正解だが自信が持てない）
base, _ := url.Parse("https://api.example.com/v1")
u1 := *base
u1.Path = path.Join(u1.Path, "users")

// パターン2: 非効率なシリアライズ/パース
u2, _ := url.Parse(base.String())
u2.Path = path.Join(u2.Path, "posts")

// パターン3: ResolveReferenceを使う（冗長）
u3 := base.ResolveReference(&url.URL{Path: "comments"})
// ただしベースURLに末尾スラッシュが必要など、直感的でない


// After: 新APIを使った書き方
base, _ := url.Parse("https://api.example.com/v1")

u1 := base.Clone()
u1.Path = path.Join(u1.Path, "users")

u2 := base.Clone()
u2.Path = path.Join(u2.Path, "posts")

// 明確で効率的、かつ安全
```

```go
// url.Valuesのクローン例

// Before: 非効率な方法
origQuery := url.Values{"page": []string{"1"}, "limit": []string{"10"}}
queryStr := origQuery.Encode()
newQuery, _ := url.ParseQuery(queryStr)
newQuery.Set("page", "2")

// After: 効率的な方法
origQuery := url.Values{"page": []string{"1"}, "limit": []string{"10"}}
newQuery := origQuery.Clone()
newQuery.Set("page", "2")
```

### 典型的なユースケース

1. **API クライアントライブラリ**: 設定からベースURLを読み込み、各リクエストで異なるパスやクエリパラメータを設定
2. **HTTP ミドルウェア/プロキシ**: リクエストURLを加工する際、元のURLを保持しながら新しいURLを作成
3. **テストコード**: 同じベースから複数のテストケース用URLを生成
4. **データベース接続文字列の構築**: ベースとなる接続URLから、異なるデータベース名やパラメータを持つURLを複数生成

## 議論のハイライト

- **過去の提案（#41733）では却下されていた**: 2020年の提案では「Clone は多くの型に必要になるが、URL は特別ではない」として却下。しかし今回は実データ（GitHubコード検索）で需要の高さを証明。

- **`UserInfo`のコピーについて**: 初期の議論では「`u2 := *u`で十分では？」という意見があったが、最終的には「`UserInfo`も念のためコピーする実装が望ましい」という結論に。理由は、ほとんどのURLには`UserInfo`が含まれないため追加コストは無視でき、含まれる場合でもクローン間の独立性が保証される方が安全。

- **`url.Values.Clone()`の自然さ**: `http.Header`に既に`.Clone()`が存在し、`url.Values`は同じ型構造を持つため、対称性の観点からも追加が自然。コメントでは「これは実質2つの提案だが、どちらも理にかなっている」との評価。

- **`new(*expr)`構文との相性**: Go 1.24で予定されている`new(*expr)`構文（#45624）により、実装がさらにシンプルになる見込み:
  ```go
  func (u *URL) Clone() *URL {
      if u == nil { return nil }
      u2 := new(*u)
      if u.User != nil {
          u2.User = new(*u.User)
      }
      return u2
  }
  ```

- **なぜ今承認されたか**:
  1. 実データによる需要の証明（21,200件のコピーパターン、3,300件の非効率パターン）
  2. net/httpに既に同等の内部実装が存在し、一部パッケージが非公式にアクセスしている実態
  3. Damian Neil（@neild）による詳細な技術的分析コメントが、実装の正当性を裏付けた

## Sources:
- [go/src/net/http/clone.go at master · golang/go](https://github.com/golang/go/blob/master/src/net/http/clone.go)
- [proposal: net/url: add URL.Clone method · Issue #41733](https://github.com/golang/go/issues/41733)
- [proposal: net/url: URL.Clone, Values.Clone · Issue #73450](https://github.com/golang/go/issues/73450)
- [net/url: clarify safe way to do URL deep-copying · Issue #38351](https://github.com/golang/go/issues/38351)
- [http package - net/http - Go Packages](https://pkg.go.dev/net/http)
- [url package - net/url - Go Packages](https://pkg.go.dev/net/url)

## 関連リンク

- [proposal: net/url: URL.Clone, Values.Clone · Issue #73450](https://github.com/golang/go/issues/73450)
- [Review Minutes (Active指定)](https://github.com/golang/go/issues/33502#issuecomment-3814236325)
- [proposal: net/url: add URL.Clone method · Issue #41733](https://github.com/golang/go/issues/41733)
- [net/url: clarify safe way to do URL deep-copying · Issue #38351](https://github.com/golang/go/issues/38351)
- [new(*expr)構文提案 #45624](https://github.com/golang/go/issues/45624)
