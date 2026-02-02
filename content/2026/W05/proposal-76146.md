---
issue_number: 76146
title: "x/crypto/ssh: add AuthCallback to ClientConfig"
previous_status: 
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "@cthach氏"
    url: https://github.com/golang/go/issues/76146#issuecomment-3811842539
  - title: "proposal: x/crypto/ssh: dynamic auth method selection in ServerConfig · Issue #64974 · golang/go"
    url: https://github.com/golang/go/issues/64974
  - title: "x/crypto/ssh: Client Auth: handle partial success correctly · Issue #23461 · golang/go"
    url: https://github.com/golang/go/issues/23461
  - title: "@aclements氏"
    url: https://github.com/golang/go/issues/76146#issuecomment-3814236239
  - title: "Proposal Issue #76146"
    url: https://github.com/golang/go/issues/76146
---
## 概要

`x/crypto/ssh`パッケージのクライアント認証において、実行時に認証方法を動的に選択できる`AuthCallback`フィールドを`ClientConfig`に追加する提案です。これにより、サーバーから返されるメタデータや部分成功（partial success）の情報に基づいて、認証戦略を柔軟に変更できるようになります。

## ステータス変更
**(なし)** → **active**

2026年1月28日にプロポーザルレビューグループにより、提案が正式にレビュー対象として**active**ステータスに移行しました。Teleportなどのエンタープライズ用途でのセキュリティ強化要望が背景にあり、特に[@cthach氏](https://github.com/golang/go/issues/76146#issuecomment-3811842539)からは「何百、何千もの組織のセキュリティ改善につながる」として早期実装を求める声が寄せられています。

## 技術的背景

### 現状の問題点

現在の`x/crypto/ssh`パッケージでは、認証方法は`ClientConfig.Auth`に設定した`AuthMethod`の配列を順番に試行する**静的な仕組み**です。

```go
// 現在の実装: 認証方法は事前に固定
config := &ssh.ClientConfig{
    User: "username",
    Auth: []ssh.AuthMethod{
        ssh.PublicKeys(signer),      // 1番目に試行
        ssh.Password("password"),     // 2番目に試行
    },
    HostKeyCallback: ssh.FixedHostKey(hostKey),
}
```

この方式には以下の制限があります:

1. **動的な決定が不可能**: サーバーが返すメタデータ（サポートする認証方法、部分成功情報など）に基づいて認証方法を選択できない
2. **認証の中断ができない**: ランタイムコンテキストやポリシーに基づいて認証プロセスを中止する手段がない
3. **情報へのアクセス制限**: `ConnMetadata`や`NegotiatedAlgorithms`は認証完了後にしかアクセスできず、認証プロセス中には利用できない
4. **多段階認証への対応不足**: RFC 4252で定義された部分成功（partial success）フラグを活用した柔軟な多段階認証を実装しにくい

### 提案された解決策

`ClientConfig`に新しいオプショナルなコールバックフィールド`AuthCallback`を追加します:

```go
type ClientConfig struct {
    // ... 既存フィールド

    // AuthCallback は各認証試行の前に呼び出されるオプショナルなフック。
    // コールバックは接続メタデータ、ネゴシエートされたアルゴリズム、
    // サーバーがサポートする認証方法、既に成功した方法（部分成功時）、
    // 失敗した方法を検査できる。
    //
    // 戻り値:
    // - 非nilエラー: 認証プロセスが即座に停止
    // - 非nil AuthMethodとnil エラー: 返されたAuthMethodが次に試行される
    //   （ClientConfig.Authの方法の代わり）
    // - nil AuthMethodとnil エラー: ClientConfig.Authで定義された方法が試行される
    AuthCallback func(
        conn ConnMetadata,
        algorithms NegotiatedAlgorithms,
        supportedAuthMethods []string,    // サーバーがサポートする方法
        succeededAuthMethods []string,    // 既に成功した方法（部分成功時）
        failedAuthMethods []string,       // 既に失敗した方法
    ) (AuthMethod, error)
}
```

**呼び出しタイミング**: 初回の`none`認証の後、サーバーのサポートする認証方法が判明した時点で呼び出され、以降の各認証試行の前にも呼び出されます。

**後方互換性**: `AuthCallback`が設定されていない場合、既存の動作と完全に互換性があります。

## これによって何ができるようになるか

### 1. サーバーメタデータに基づく動的な認証方法選択

サーバーが返す情報に基づいて、最適な認証方法を実行時に決定できます。例えば、サーバーが特定の認証方法をサポートしている場合のみ、その方法を試行するなど。

### 2. 多段階認証の高度な制御

RFC 4252の部分成功メカニズムを活用し、既に成功した認証ステップに基づいて次の認証方法を動的に選択できます。これはTeleportのような高度なSSHプロキシシステムでの要件です。

### 3. 認証プロセスの中断

ポリシー違反や特定の条件下で、認証プロセスを早期に中断できます。

### 4. アウトオブバンド認証との統合

サーバーから返されるカスタム認証方法名（例: "web-auth"）を検知し、ブラウザベースの認証フローへリダイレクトするなど、従来の枠を超えた認証フローを実装できます。

## コード例

### Before: 従来の静的な認証

```go
config := &ssh.ClientConfig{
    User: "username",
    Auth: []ssh.AuthMethod{
        ssh.PublicKeys(keySigner),
        ssh.Password("password"),
    },
    HostKeyCallback: ssh.InsecureIgnoreHostKey(),
}

// 問題:
// - サーバーがどの認証方法をサポートしているか分からないまま試行
// - 部分成功の情報を活用できない
// - 動的に認証戦略を変更できない
```

### After: AuthCallbackを使った動的な認証

```go
config := &ssh.ClientConfig{
    User: "username",
    Auth: []ssh.AuthMethod{
        // デフォルトの認証方法（AuthCallbackが何も返さない場合）
        ssh.PublicKeys(keySigner),
    },
    AuthCallback: func(
        conn ssh.ConnMetadata,
        algorithms ssh.NegotiatedAlgorithms,
        supported, succeeded, failed []string,
    ) (ssh.AuthMethod, error) {
        // サーバーがサポートする方法に応じて動的に選択
        if containsMethod(supported, "publickey") && containsMethod(succeeded, "password") {
            // パスワード認証が既に成功している場合のみ公開鍵認証を試行
            return ssh.PublicKeys(keySigner), nil
        }

        if containsMethod(supported, "keyboard-interactive") {
            // サーバーがキーボードインタラクティブをサポートしている場合
            return ssh.KeyboardInteractive(challengeHandler), nil
        }

        // 特定の条件で認証を中断
        if isAuthenticationBlocked(conn) {
            return nil, fmt.Errorf("authentication blocked by policy")
        }

        // デフォルトの方法を使用
        return nil, nil
    },
    HostKeyCallback: ssh.InsecureIgnoreHostKey(),
}
```

## 議論のハイライト

- **提案者**: [@drakkan氏](https://github.com/golang/go/issues/76146)（x/crypto/sshのメンバー）が2025年11月2日に提案し、同日[CL 717140](https://go.dev/cl/717140)で実装を提出
- **実装の影響範囲**: 完全に後方互換性があり、既存コードへの影響はゼロ。`AuthCallback`は純粋にオプトイン機能
- **実用的需要**: Teleport（エンタープライズSSHアクセス管理システム）での実装要望が明確に示され、セキュリティ強化の緊急性が強調されている
- **関連提案との統合**: サーバーサイドの動的認証方法選択を扱う[Issue #64974](https://github.com/golang/go/issues/64974)と対になる提案であり、クライアント・サーバー双方向での柔軟な認証制御を実現
- **技術的根拠**: RFC 4252の部分成功メカニズムと、[Issue #23461](https://github.com/golang/go/issues/23461)で議論された多段階認証の課題に対する解決策
- **レビュー状況**: 2026年1月28日に[@aclements氏](https://github.com/golang/go/issues/76146#issuecomment-3814236239)により正式にactiveステータスへ移行し、週次プロポーザルレビューミーティングでの審議対象となった

## Sources:
- [ssh package - golang.org/x/crypto/ssh - Go Packages](https://pkg.go.dev/golang.org/x/crypto/ssh)
- [proposal: x/crypto/ssh: dynamic auth method selection in ServerConfig · Issue #64974 · golang/go](https://github.com/golang/go/issues/64974)
- [x/crypto/ssh: Client Auth: handle partial success correctly · Issue #23461 · golang/go](https://github.com/golang/go/issues/23461)
- [RFC 4252 - The Secure Shell (SSH) Authentication Protocol](https://datatracker.ietf.org/doc/html/rfc4252)
- [Teleport GitHub Repository](https://github.com/gravitational/teleport)
- [client package - github.com/gravitational/teleport/api/client](https://pkg.go.dev/github.com/gravitational/teleport/api/client)

## 関連リンク

- [@cthach氏](https://github.com/golang/go/issues/76146#issuecomment-3811842539)
- [proposal: x/crypto/ssh: dynamic auth method selection in ServerConfig · Issue #64974 · golang/go](https://github.com/golang/go/issues/64974)
- [x/crypto/ssh: Client Auth: handle partial success correctly · Issue #23461 · golang/go](https://github.com/golang/go/issues/23461)
- [@aclements氏](https://github.com/golang/go/issues/76146#issuecomment-3814236239)
- [Proposal Issue #76146](https://github.com/golang/go/issues/76146)
