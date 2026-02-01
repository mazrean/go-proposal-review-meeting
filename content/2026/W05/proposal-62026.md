---
issue_number: 62026
title: "crypto/uuid: add API to generate and parse UUID"
previous_status: discussions
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "関連提案: crypto/rand にUUIDv4/v7関数を追加"
    url: https://github.com/golang/go/issues/76319
  - title: "過去の却下提案 #23789 (2018年)"
    url: https://github.com/golang/go/issues/23789
  - title: "過去の却下提案 #28324 (2018年)"
    url: https://github.com/golang/go/issues/28324
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/62026
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
---

## 要約

## 概要

Go言語の標準ライブラリに`crypto/uuid`パッケージを追加し、UUIDの生成とパース機能を提供する提案です。特に広く使われているUUID version 4（ランダム）とversion 7（時刻ベース・ソート可能）の生成に焦点を当て、最小限かつ実用的なAPIを目指しています。

## ステータス変更

**(新規)** → **active**

2026年1月28日のProposal Review Meetingで、この提案がactiveステータスに移行しました。これは、長年にわたって却下されてきたUUID標準ライブラリ化の議論が、ついに実現に向けて具体的な検討段階に入ったことを意味します。activeステータスへの移行は、提案の詳細な仕様について活発な議論が進行中であり、実装に向けた合意形成が進んでいることを示しています。

## 技術的背景

### 現状の問題点

現在、Go開発者はUUIDを使用する際、外部パッケージ（特に`github.com/google/uuid`）に依存せざるを得ません。このパッケージは10万以上のGoプロジェクトで使用されており、サーバー/DB系アプリケーションでは事実上の必須依存関係となっています。しかし以下の課題があります:

- 外部依存を毎回追加する必要がある
- `github.com/google/uuid`パッケージがメンテナンス不足の状態にある
- C#、Java、JavaScript、Python、Rubyなど他の主要言語は標準ライブラリでUUIDをサポートしており、Goは例外的な存在

過去に2回（#23789、#28324）UUIDの標準ライブラリ化が提案されましたが、いずれも「外部パッケージで十分」として却下されていました。今回の再提案は、外部パッケージのメンテナンス問題と、UUIDが広く使われる基本的な構成要素であることが再認識されたことが背景にあります。

### 提案された解決策

最終的に合意に向かっているAPI仕様は、@neildによって提案された最小限の設計です:

```go
package uuid // crypto/uuid

type UUID [16]byte

// Parse parses the hex-and-dash form, and no other: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func Parse[T ~string | ~[]byte](s T) (UUID, error)
func MustParse[T ~string | ~[]byte](s T) UUID

// New is an opinionated "just give me a UUID" function.
func New() UUID { return NewV4() }

func NewV4() UUID  // 完全ランダムなUUID
func NewV7() UUID  // タイムスタンプベースでソート可能なUUID

func (UUID) MarshalText() ([]byte, error)
func (UUID) AppendText([]byte) ([]byte, error)
func (*UUID) UnmarshalText([]byte) error
func (UUID) Compare(UUID) int
func (UUID) String() string
```

**重要な設計決定**:

1. **`[16]byte`型の採用**: `github.com/google/uuid`と同じ内部表現を使用することで、型変換が簡単なキャストで済む
2. **Version 4 と Version 7のみサポート**: 調査の結果、V4が全体の82%、V7が1.2%の利用率で、他の全バージョン合計（0.5%）を大きく上回る
3. **パース形式は標準形式のみ**: セキュリティとプロトコル互換性の観点から、`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`形式のみを受け付け、変則的な形式は受け付けない
4. **database/sql対応**: `database/sql`パッケージをUUID対応にし、ドライバが独自にUUIDを処理しない場合でも標準的な文字列変換で動作するようにする

## これによって何ができるようになるか

UUID生成が標準ライブラリだけで完結し、外部依存なしでユニークな識別子を扱えるようになります。特にデータベースの主キー、分散システムのID、セッショントークンなど、UUIDが必要なあらゆる場面で利用できます。

### コード例

```go
// Before: 外部パッケージへの依存
import "github.com/google/uuid"

userID := uuid.New()  // error返り値があるが常にnil
sessionID := uuid.NewV7()

// After: 標準ライブラリのみで完結
import "crypto/uuid"

userID := uuid.New()      // シンプルに取得（内部的にNewV4を呼ぶ）
sessionID := uuid.NewV7() // ソート可能なUUID（DB挿入パフォーマンス向上）

// パース（設定ファイルやDB読み込み時）
const adminID = "550e8400-e29b-41d4-a716-446655440000"
id := uuid.MustParse(adminID)  // コンパイル時定数として安全

// データベースとの統合（database/sqlが自動対応）
var user User
db.QueryRow("SELECT id, name FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Name)

// JSON等への自動マーシャリング
type User struct {
    ID   uuid.UUID `json:"id"`
    Name string    `json:"name"`
}
// MarshalText/UnmarshalTextが自動的に使われる
```

**UUIDv4 vs UUIDv7の使い分け**:

- **UUIDv4（ランダム）**: プライバシーが重要な場合、作成時刻を隠したい場合、分散DBでホットスポットを避けたい場合に推奨
- **UUIDv7（時刻ベース）**: データベースでの挿入性能が重要な場合、時系列での並び替えが必要な場合に推奨（PostgreSQL 18やPercona MySQLなどが公式サポート）

## 議論のハイライト

- **APIの最小化**: 当初は`github.com/google/uuid`の全機能を取り込む案もあったが、実際の利用統計分析（@rolandshoemaker）により、`UUID.String()`が35%、生成関数群が50%以上で、他の機能は1%未満と判明。最小限のAPIに絞られた

- **New()関数の是非**: 「何も考えずにUUIDが欲しい」ユーザー向けに`New()`を用意すべきか議論。当初UUIDv7をデフォルトにする案もあったが、UUIDv7はタイムスタンプ漏洩やホットスポット問題があるため、より安全なUUIDv4を返す`New() = NewV4()`に決定

- **タイムスタンプオフセット機能の議論**: UUIDv7でタイムスタンプをずらす機能（プライバシー保護やロック競合回避）について激しい議論があったが、「オフセットを加えても完全な匿名化にはならない」「本当に必要なら外部パッケージで実装可能」として、初期実装からは除外

- **パース形式の厳格性**: セキュリティとプロトコル互換性のため、RFC 9562の標準形式のみを受け付ける厳格な仕様に。他言語との相互運用時の予期しない動作を防ぐため、将来的にも拡張しないことを保証

- **database/sql統合**: UUIDはデータベースで頻繁に使用されるため、`database/sql`側を拡張してUUID型を認識させ、ドライバが未対応でも文字列として自動変換する仕組みを導入

- **RFC 9562の「不透明性」原則**: RFC 9562は「UUIDは可能な限り不透明な識別子として扱うべき」と推奨しているため、Version/NodeID/Timestampなどを取り出すメソッドは提供しない方針（利用率も0.07%以下）

## 関連リンク
- [Proposal Issue](https://github.com/golang/go/issues/62026)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連提案: crypto/rand にUUIDv4/v7関数を追加](https://github.com/golang/go/issues/76319)
- [過去の却下提案 #23789 (2018年)](https://github.com/golang/go/issues/23789)
- [過去の却下提案 #28324 (2018年)](https://github.com/golang/go/issues/28324)
- [RFC 9562: Universally Unique IDentifiers (UUIDs)](https://www.rfc-editor.org/rfc/rfc9562.html)
- [github.com/google/uuid パッケージ](https://pkg.go.dev/github.com/google/uuid)

---

**Sources:**
- [RFC 9562: Universally Unique IDentifiers (UUIDs)](https://www.rfc-editor.org/rfc/rfc9562.html)
- [GitHub - google/uuid: Go package for UUIDs](https://github.com/google/uuid)
- [uuid package - github.com/google/uuid - Go Packages](https://pkg.go.dev/github.com/google/uuid)

## 関連リンク

- [関連提案: crypto/rand にUUIDv4/v7関数を追加](https://github.com/golang/go/issues/76319)
- [過去の却下提案 #23789 (2018年)](https://github.com/golang/go/issues/23789)
- [過去の却下提案 #28324 (2018年)](https://github.com/golang/go/issues/28324)
- [Proposal Issue](https://github.com/golang/go/issues/62026)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
