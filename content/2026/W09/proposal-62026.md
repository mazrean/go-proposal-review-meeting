---
issue_number: 62026
title: "crypto/uuid: add API to generate and parse UUID"
previous_status: active
current_status: likely_accept
changed_at: 2026-02-25T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3962620065
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/62026
  - title: "関連Issue: crypto/rand改善 #66821"
    url: https://github.com/golang/go/issues/66821
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3962620065
  - title: "最終API仕様コメント"
    url: https://github.com/golang/go/issues/62026#issuecomment-3961250320
  - title: "エコシステム分析コメント"
    url: https://github.com/golang/go/issues/62026#issuecomment-3564123274
  - title: "関連Issue: 過去の却下プロポーザル #23789"
    url: https://github.com/golang/go/issues/23789
---
## 概要

`crypto/uuid` パッケージを Go 標準ライブラリに追加し、UUID（Universally Unique Identifier、世界的に一意な識別子）の生成とパースのための API を提供するプロポーザルです。現在 Go エコシステムで最も広く使われているサードパーティライブラリ `github.com/google/uuid` の中核機能を、厳選された最小限の API として標準ライブラリに取り込むことを目指します。

## ステータス変更
**active** → **likely_accept**

2026年2月25日、`@aclements` がプロポーザルレビューグループを代表して `likely accept` と宣言しました。最終的なAPI仕様（`https://github.com/golang/go/issues/62026#issuecomment-3961250320`）が確定し、`Parse` が受け付けるフォーマットの拡充、`Nil` と `Max` の変数化、`Compare` メソッドのドキュメント改善が合意されたことを受けた判断です。

## 技術的背景

### 現状の問題点

GoはUUID生成・パースの標準APIを持たず、実質的に `github.com/google/uuid` が業界標準として機能しています。`@rolandshoemaker` のエコシステム分析によると、このパッケージの利用関数トップは `New`（36%）、`UUID.String`（35%）、`Parse`（8%）であり、極めて基本的な機能のみに集中しています。一方で同パッケージはメンテナンスが停滞しており、時代とともに肥大化したAPIの整合性も問題となっています。

また `crypto/rand` の改善（[#66821](https://github.com/golang/go/issues/66821)）によりランダム生成の失敗が事実上なくなったため、`google/uuid` の `New() (UUID, error)` というシグネチャは時代遅れになっています。

### 提案された解決策

`crypto/uuid` パッケージとして以下の最小限のAPIを追加します（2026年2月25日時点の最終仕様）。

```go
package uuid // crypto/uuid

// UUID は RFC 9562 に準拠した16バイトの値
type UUID [16]byte

var (
    Nil = UUID{}                                    // Nil UUID: 00000000-0000-0000-0000-000000000000
    Max = UUID{0xff, 0xff, ..., 0xff}               // Max UUID: ffffffff-ffff-ffff-ffff-ffffffffffff
)

// Parse は以下の複数形式を受け付ける（google/uuid との互換性を保つ）
// - f81d4fae-7dec-11d0-a765-00a0c91e6bf6        (標準形式)
// - {f81d4fae-7dec-11d0-a765-00a0c91e6bf6}      (波括弧形式)
// - urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6 (URN形式)
// - f81d4fae7dec11d0a76500a0c91e6bf6            (ハイフンなし)
func Parse(s string) (UUID, error)
func MustParse(s string) UUID

func New() UUID     // 現在は NewV4 と同等
func NewV4() UUID   // 122ビットのランダムデータ
func NewV7() UUID   // 上位48ビットにタイムスタンプを含む

func (u UUID) String() string
func (u UUID) MarshalText() ([]byte, error)
func (u UUID) AppendText(b []byte) ([]byte, error)
func (u *UUID) UnmarshalText(b []byte) error
func (u UUID) Compare(v UUID) int
```

さらに `database/sql` をUUID対応させる変更も含まれます。ドライバが独自にUUIDを処理しない場合、`database/sql` が自動的にUUIDを文字列へ変換して扱います。

## これによって何ができるようになるか

Go開発者は UUID の生成・パースにサードパーティ依存なく対応できるようになります。特にサーバーサイド・データベース連携を行うアプリケーションで恩恵が大きいです。

### コード例

```go
// Before: サードパーティライブラリが必要
import "github.com/google/uuid"

id, err := uuid.NewRandom() // エラーを返すが実際には失敗しない
if err != nil {
    log.Fatal(err)
}
fmt.Println(id.String())

// After: 標準ライブラリで完結
import "crypto/uuid"

id := uuid.New()            // エラーなし、シンプル
fmt.Println(id.String())    // f81d4fae-7dec-11d0-a765-00a0c91e6bf6

// データベース用のソート可能なUUID生成
id7 := uuid.NewV7()

// 既存のUUID文字列をパース
parsed, err := uuid.Parse("f81d4fae-7dec-11d0-a765-00a0c91e6bf6")

// 定数として使用
var emptyID = uuid.Nil

// google/uuid との相互変換（型が同じ [16]byte なので自動的にキャスト可能）
// googleUUID := googleuuid.UUID(cryptoUUID)
```

## 議論のハイライト

- **`[16]byte` を基底型とした理由**: `github.com/google/uuid` と同じ基底型を使うことで、2パッケージ間の変換が型キャスト一つで可能になる。`type UUID string` では任意の文字列が格納でき不変条件を保証できないため却下された。

- **`New()` のデフォルトはUUIDv4**: UUIDv7はタイムスタンプを含むためプライバシー上の懸念があり、また分散データベース（Cloud Spannerなど）ではUUIDv4よりパフォーマンスが悪化するケースがある。一方UUIDv4はシンプルで問題がないため、`New()` のデフォルトとなった。

- **UUIDv5（SHA-1ベース）は除外**: エコシステム分析で `NewSHA1` の利用はわずか0.05%（約20件）であり、SHA-1が暗号学的に安全でなくなった現在、新規利用が増えるとは考えにくい。実装は約10行で外部パッケージで対応可能と判断された。

- **`Parse` の許容フォーマット**: 当初は標準の `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` 形式のみの厳格なパースが検討されたが、`google/uuid` との実行時互換性を優先し、波括弧形式・URN形式・ハイフンなし形式も受け付けることに決定。

- **`Nil` と `Max` の実装形式**: 当初は関数（`func Nil() UUID`）として提案されていたが、`google/uuid` との一貫性と利便性のため変数（`var Nil = UUID{}`）に変更された。コメント欄では `const` にしたいという声もあったが、Go言語の現状では配列型の定数はサポートされていない。

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/62026)
- [関連Issue: crypto/rand改善 #66821](https://github.com/golang/go/issues/66821)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3962620065)
- [最終API仕様コメント](https://github.com/golang/go/issues/62026#issuecomment-3961250320)
- [エコシステム分析コメント](https://github.com/golang/go/issues/62026#issuecomment-3564123274)
- [関連Issue: 過去の却下プロポーザル #23789](https://github.com/golang/go/issues/23789)
