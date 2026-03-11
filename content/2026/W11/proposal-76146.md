---
issue_number: 76146
title: "x/crypto/ssh: add AuthCallback to ClientConfig"
previous_status: active
current_status: likely_accept
changed_at: 2026-03-11T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4042167102
related_issues:
  - title: "最終提案コメント"
    url: https://github.com/golang/go/issues/76146#issuecomment-3980536259
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/76146
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4042167102
---
## 概要

`x/crypto/ssh` パッケージの `ClientConfig` に `AuthCallback` フィールドを追加することで、SSH クライアント認証を動的に制御できるようにするプロポーザルです。サーバーの応答や認証状況に応じて認証方式をリアルタイムで選択できるようになります。

## ステータス変更
**active** → **likely_accept**

2026年3月11日のProposal Review Meetingにて、@aclements を含むレビューチームが提案内容を精査した結果、「likely accept」に移行しました。初期の関数シグネチャに関する懸念（スタイル上の一貫性不足、将来の拡張性）に対し、提案者が構造体型の導入と型エイリアスによる改善を行い、コメントも丁寧に整備されたことで受け入れ判断に至りました。現在は最終コメント期間（Last Call for Comments）の状態にあります。

## 技術的背景

### 現状の問題点

現在の `x/crypto/ssh` パッケージでは、認証方式は `ClientConfig.Auth` に静的に列挙されており、クライアントは `none` 認証を最初に試みた後、リストの先頭から順に認証を試行します。この設計には以下の制限があります。

- サーバーが返すメタデータや部分成功（partial success）の結果に基づいた動的な認証方式の選択ができない
- ランタイムのコンテキストやポリシーに基づいて認証プロセスを中断する手段がない
- `ConnMetadata` や `NegotiatedAlgorithms`（ネゴシエート済みアルゴリズム情報）は認証完了後にしかアクセスできず、認証中には参照できない

### 提案された解決策

`ClientConfig` に `AuthCallback ClientAuthCallback` フィールドを追加します。`ClientAuthCallback` は名前付き関数型で、認証試行前に毎回呼び出されます。コールバックには `*ClientAuthContext` が渡され、以下の情報にアクセスできます。

```go
type ClientAuthContext struct {
    Metadata              ConnMetadata
    Algorithms            NegotiatedAlgorithms
    AllowedMethods        []string  // サーバーが許可する認証方式（RFC 4252 準拠の名称）
    PartialSuccessMethods []string  // 部分的に成功した認証方式
    TriedMethods          []string  // 失敗した認証方式
}

type ClientAuthCallback func(ctx *ClientAuthContext) (AuthMethod, error)
```

戻り値の意味は以下のとおりです。

- `(AuthMethod, nil)`: 指定した認証方式を次に試みる（`ClientConfig.Auth` に含まれていなくてもよい）
- `(nil, nil)`: 通常のフロー（`ClientConfig.Auth` の未試行メソッドを順に使用）に従う
- `(nil, error)`: 認証プロセスを即座に中断し、エラーを返す

`AuthCallback` が未設定の場合は従来の動作と完全に互換性があります。

## これによって何ができるようになるか

このコールバックにより、SSH認証の高度な制御が可能になります。特にセキュリティ要件の厳しいシステムやエンタープライズ向けSSHプロキシで有用です。

### コード例

```go
// Before: 静的な認証方式の列挙（柔軟な制御が困難）
config := &ssh.ClientConfig{
    User: "alice",
    Auth: []ssh.AuthMethod{
        ssh.PublicKeys(signer),
        ssh.Password("fallback-password"),
    },
}

// After: AuthCallback による動的な認証方式の選択
config := &ssh.ClientConfig{
    User: "alice",
    Auth: []ssh.AuthMethod{
        ssh.PublicKeys(signer),
    },
    AuthCallback: func(ctx *ssh.ClientAuthContext) (ssh.AuthMethod, error) {
        // 公開鍵認証が失敗した場合のみパスワード認証にフォールバック
        for _, tried := range ctx.TriedMethods {
            if tried == "publickey" {
                return ssh.Password("fallback-password"), nil
            }
        }
        // 部分成功（多要素認証）の場合は別のメソッドを選択
        if len(ctx.PartialSuccessMethods) > 0 {
            return ssh.KeyboardInteractive(promptFunc), nil
        }
        return nil, nil // デフォルトのフローに従う
    },
}
```

実践的なユースケースとしては以下が挙げられます。

1. **条件付きフォールバック**: 公開鍵認証が失敗した場合のみパスワード認証を試みる（常にパスワード認証を試みるリスクを回避）
2. **多段階認証（MFA）のサポート**: `PartialSuccessMethods` を参照して、部分的な成功後に追加の認証ステップを動的に選択する
3. **セキュリティポリシーの適用**: 認証失敗回数や接続メタデータを検査し、一定条件でポリシー違反として接続を中断する（Teleport などのSSHプロキシでの活用が期待されている）

## 議論のハイライト

- **構造体の導入**: 初期提案は関数シグネチャに複数のパラメータを直接渡す形式だったが、@aclements の指摘により将来的な拡張性を考慮して `ClientAuthContext` 構造体にまとめる形に変更された
- **型エイリアスの追加**: スタイルの一貫性を確保するため、関数型に `ClientAuthCallback` という名前付き型が付与された
- **`ClientConfig.Auth` との関係の明確化**: コールバックが返す `AuthMethod` は `ClientConfig.Auth` に含まれていなくてもよい点、`(nil, nil)` を返した後もコールバックが再び呼ばれる点が明示的にドキュメントされた
- **`Algorithms` フィールドの採用**: `ConnMetadata` は `AlgorithmsConnMetadata` インターフェースを実装しているため型アサーションでアクセス可能だが、APIの明示性と利便性のために `ClientAuthContext` に直接含める設計が採用された
- **Teleport での需要**: 大規模なSSHアクセス管理ツールである Teleport でのセキュリティ改善に必要な機能として、コミュニティから強い要望が寄せられており、実用的な需要が受け入れ判断を後押しした

## 関連リンク

- [最終提案コメント](https://github.com/golang/go/issues/76146#issuecomment-3980536259)
- [Proposal Issue](https://github.com/golang/go/issues/76146)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4042167102)
