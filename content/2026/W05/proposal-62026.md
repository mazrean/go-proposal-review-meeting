---
issue_number: 62026
title: "crypto/uuid: add API to generate and parse UUID"
previous_status: 
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "過去の却下例 #23789 (2018年)"
    url: https://github.com/golang/go/issues/23789
  - title: "Proposal Issue #62026"
    url: https://github.com/golang/go/issues/62026
  - title: "Review Minutes (2026-01-28)"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "関連提案 #76319: crypto/rand.UUIDv4/UUIDv7（より限定的なアプローチ）"
    url: https://github.com/golang/go/issues/76319
---

## 要約

## 概要

Go標準ライブラリに`crypto/uuid`パッケージを追加し、UUIDの生成とパース機能を提供する提案です。特にバージョン4（ランダム）とバージョン7（タイムスタンプベース）のUUID生成をサポートします。

## ステータス変更

**(未設定)** → **Active**

2026年1月28日のProposal Review Meetingで「Active」ステータスとして議事録に追加されました。過去に2018年（#23789）と2019年（#28324）に同様の提案が却下されていましたが、今回は標準ライブラリ入りに向けて前向きな検討が進んでいます。

## 技術的背景

### 現状の問題点

Go開発者は現在、UUID生成のために`github.com/google/uuid`などのサードパーティパッケージを利用しています。このパッケージは10万以上のプロジェクトで使用される事実上の標準となっていますが、以下の課題があります:

1. **メンテナンス体制の不透明性**: 2025年以降、メンテナーの応答が鈍く、UUIDv8サポートのPRが半年以上放置されるなど、プロジェクトの持続可能性に懸念が生じています
2. **他言語との差**: C#、Java、JavaScript、Python、RubyなどはUUID生成機能を標準ライブラリに含んでいますが、Goは例外的に含んでいません
3. **RFC 9562の正式化**: 2024年5月にUUID仕様がRFC 9562として正式化され、UUIDv7が新たに標準化されました

### 提案された解決策

neildによって具体化された最小限のAPI案:

```go
package uuid // crypto/uuid

type UUID [16]byte

func Parse[T ~string | ~[]byte](s T) (UUID, error)
func MustParse[T ~string | ~[]byte](s T) UUID

func New() UUID        // NewV4()のエイリアス
func NewV4() UUID      // ランダムUUID生成
func NewV7() UUID      // タイムスタンプベースUUID生成

func (UUID) MarshalText() ([]byte, error)
func (UUID) AppendText([]byte) ([]byte, error)
func (*UUID) UnmarshalText([]byte) error
func (UUID) Compare(UUID) int
func (UUID) String() string
```

**設計上の重要な決定:**

- **型は`[16]byte`**: `github.com/google/uuid`と互換性があり、型キャストだけで変換可能
- **V4をデフォルト**: `New()`はV4を返す。V7はホットスポット問題やプライバシー懸念があるため
- **パース形式は厳格**: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`形式のみ受け入れ、RFC準拠を保証
- **database/sql統合**: 標準ライブラリのUUID型として、database/sqlパッケージ側でUUID対応を追加予定

## これによって何ができるようになるか

1. **サードパーティ依存の削減**: 最も基本的なUUID生成機能を標準ライブラリで提供
2. **長期的なメンテナンス保証**: Goチームによる公式サポートで、メンテナンス継続性を担保
3. **データベース統合の改善**: `database/sql`がUUID型をネイティブに認識し、ドライバーごとの実装なしで基本的な機能が動作
4. **シンプルなAPI**: 「とにかくUUIDが欲しい」というユースケースに対して`uuid.New()`一つで対応

### コード例

```go
// Before: サードパーティパッケージの利用
import "github.com/google/uuid"

id, err := uuid.NewRandom()
if err != nil {
    // 実際にはほぼ発生しないエラー処理
    return err
}
idStr := id.String()

// After: 標準ライブラリの利用
import "crypto/uuid"

id := uuid.New()  // エラーなし（crypto/randの改善により）
idStr := id.String()

// パース
parsed, err := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")

// database/sqlでの利用（自動的に文字列変換される）
db.Exec("INSERT INTO users (id, name) VALUES (?, ?)", uuid.New(), "Alice")
```

## 議論のハイライト

1. **過去2回の却下理由**: 2018年のrscのコメントでは「標準ライブラリに何が必要か十分な情報がない」として却下。当時はV4だけで十分かが不明瞭でした

2. **V7採用の是非**: 長時間の議論の末、`NewV7()`は提供するものの、デフォルトの`New()`はV4を返す結論に。理由はV7のホットスポット問題（Google Cloud Spannerなどで性能悪化）とタイムスタンプリークの懸念

3. **タイムスタンプオフセット論争**: PostgreSQL 18やPercona MySQLが実装したタイムスタンプオフセット機能（プライバシー保護）を標準で含めるべきかが議論されましたが、novelな機能として見送りに

4. **最小限のAPI方針**: 当初は`github.com/google/uuid`をそのまま標準ライブラリに含める案もありましたが、`SetNodeID`などの不要な機能を排除し、80-90%のユースケースをカバーする最小APIに収束

5. **型の選択**: `type UUID string`案も検討されましたが、(1) `github.com/google/uuid`との互換性、(2) 無効なUUIDを型レベルで防げないデメリット、(3) メモリ効率から`[16]byte`に決定

6. **RFC 9562の「Opacity」原則**: 仕様書は「UUIDをパースして内部情報を取り出すのは避けるべき」と推奨しているため、`UUID.Version()`や`UUID.Timestamp()`などのメソッドは意図的に除外

## Sources

- [RFC 9562: Universally Unique IDentifiers (UUIDs)](https://www.rfc-editor.org/rfc/rfc9562.html)
- [GitHub - google/uuid: Go package for UUIDs based on RFC 4122 and DCE 1.1](https://github.com/google/uuid)
- [Has the google/uuid Go library been discontinued or neglected? - Latenode Official Community](https://community.latenode.com/t/has-the-google-uuid-go-library-been-discontinued-or-neglected/20802)

## 関連リンク

- [過去の却下例 #23789 (2018年)](https://github.com/golang/go/issues/23789)
- [Proposal Issue #62026](https://github.com/golang/go/issues/62026)
- [Review Minutes (2026-01-28)](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連提案 #76319: crypto/rand.UUIDv4/UUIDv7（より限定的なアプローチ）](https://github.com/golang/go/issues/76319)
