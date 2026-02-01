---
issue_number: 76146
title: "x/crypto/ssh: add AuthCallback to ClientConfig"
previous_status: discussions
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "関連Issue #23461: Partial Success処理の修正"
    url: https://github.com/golang/go/issues/23461
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "関連Issue #64974: サーバー側動的認証メソッド選択"
    url: https://github.com/golang/go/issues/64974
  - title: "Proposal Issue #76146"
    url: https://github.com/golang/go/issues/76146
  - title: "関連Issue #61447: サーバー側多段階認証"
    url: https://github.com/golang/go/issues/61447
---

## 要約

## 概要
この提案は、`x/crypto/ssh`パッケージのクライアント認証プロセスに動的な制御機能を追加するものです。現在、SSH認証は`ClientConfig.Auth`で事前に指定したメソッドを順番に試す静的なフローですが、新しい`AuthCallback`フックを追加することで、サーバーからの応答や接続メタデータに基づいて認証メソッドを動的に選択できるようになります。

## ステータス変更
**(新規)** → **active**

この提案は2026年1月28日にProposal Review Groupによってactiveステータスに移行されました。Teleportプロジェクトが実際のセキュリティ改善のためにこの機能を必要としており、複数の組織のセキュリティに影響する重要な機能として認識されています。

## 技術的背景

### 現状の問題点

現在の`x/crypto/ssh`パッケージでは、クライアント認証は次のように動作します:

1. 最初に`none`認証を試行（常に最初）
2. サーバーが返した利用可能な認証メソッドリストに基づき、`ClientConfig.Auth`に設定されたメソッドを順番に試す
3. 認証が成功するか、すべてのメソッドが失敗するまで繰り返す

この静的なアプローチには以下の制限があります:

- **動的な認証判断ができない**: サーバーから返されるメタデータやpartial success（部分成功）に応じて認証メソッドを変更できない
- **認証プロセスの中断が困難**: 実行時のコンテキストやポリシーに基づいて認証を中止する明確な方法がない
- **ネゴシエーション情報へのアクセス制限**: `ConnMetadata`や`NegotiatedAlgorithms`は認証完了後にしかアクセスできず、認証中には利用できない

RFC 4252では、認証失敗時に`partial success`フラグを返すことで多段階認証をサポートしていますが、現在のクライアント実装ではこれに動的に対応する仕組みがありません。

### 提案された解決策

`ClientConfig`に新しいオプショナルなコールバックフィールドを追加:

```go
// AuthCallback is an optional hook invoked before each authentication
// attempt. The callback can inspect the connection metadata, negotiated
// algorithms, and the authentication methods that are supported by the
// server, have already succeeded (in case of partial success), or previously
// failed.
//
// Return values:
//
// - non-nil error: the authentication process stops immediately.
// - non-nil AuthMethod and nil error: the returned AuthMethod will be
//   attempted next, instead of the methods defined in [ClientConfig.Auth].
// - nil [AuthMethod] and nil error: the methods defined in
//   [ClientConfig.Auth] will be attempted.
AuthCallback func(conn ConnMetadata, algorithms NegotiatedAlgorithms,
    supportedAuthMethods, succeededAuthMethods, failedAuthMethods []string) (AuthMethod, error)
```

このコールバックは以下のタイミングで呼び出されます:

1. 初回の`none`認証後、サーバーがサポートするメソッドが判明した時点
2. 以降の各認証試行の前

戻り値の意味:

- **非nilの`AuthMethod`**: そのメソッドを次に試行
- **非nilの`error`**: 認証プロセスを即座に中止
- **両方nil**: 通常の認証フロー（`ClientConfig.Auth`）に従う

`AuthCallback`が設定されていない場合、動作は完全に従来通りで後方互換性が保たれます。

## これによって何ができるようになるか

### 1. サーバーメタデータに基づく動的な認証選択

接続確立時にネゴシエートされたアルゴリズムやサーバー情報を確認して、最適な認証メソッドを選択できます。

### 2. Partial Success（部分成功）への対応

多段階認証が必要な場合、サーバーが返す部分成功のステータスに応じて、次の認証ステップを動的に決定できます。例えば、パスワード認証が成功した後、特定の公開鍵認証を要求するようなケースです。

### 3. ポリシーベースの認証中断

特定の条件下で認証を安全に中断できます。例えば、許可されていないアルゴリズムがネゴシエートされた場合や、組織のセキュリティポリシーに違反する場合などです。

### 4. セキュリティ強化のための高度な制御

Teleportのようなエンタープライズセキュリティソリューションでは、認証プロセス中にサーバーから得られる情報を元に、より厳格なセキュリティポリシーを適用できます。

### コード例

```go
// Before: 従来の静的な認証設定
config := &ssh.ClientConfig{
    User: "user",
    Auth: []ssh.AuthMethod{
        ssh.Password("password"),
        ssh.PublicKeys(signer),
    },
    HostKeyCallback: ssh.FixedHostKey(hostKey),
}

// After: 動的な認証メソッド選択
config := &ssh.ClientConfig{
    User: "user",
    Auth: []ssh.AuthMethod{
        ssh.Password("password"),
        ssh.PublicKeys(signer),
    },
    HostKeyCallback: ssh.FixedHostKey(hostKey),
    AuthCallback: func(
        conn ssh.ConnMetadata,
        algorithms ssh.NegotiatedAlgorithms,
        supportedAuthMethods, succeededAuthMethods, failedAuthMethods []string,
    ) (ssh.AuthMethod, error) {
        // サーバーがサポートするメソッドを確認
        if len(succeededAuthMethods) > 0 {
            // パスワード認証が成功した場合、特定の鍵のみを使う
            return ssh.PublicKeys(specificSigner), nil
        }

        // セキュリティポリシー違反の場合は中断
        if algorithms.HostKey == "weak-algorithm" {
            return nil, fmt.Errorf("host key algorithm not allowed by policy")
        }

        // 通常のフローに従う
        return nil, nil
    },
}
```

## 議論のハイライト

- **Teleportでの実用ケース**: Gravitational社のTeleportプロジェクトがこの機能を必要としており、数百から数千の組織のSSHアクセスのセキュリティ改善に直結することが強調されています

- **サーバー側の関連実装**: 過去に[Issue #61447](https://github.com/golang/go/issues/61447)でサーバー側の多段階認証サポートが追加されており、今回のクライアント側の提案はその対となる機能です

- **Partial Successの歴史**: [Issue #23461](https://github.com/golang/go/issues/23461)でクライアント側のpartial success処理に問題があったことが報告されており、この提案はその課題への包括的な解決策となります

- **実装CL**: [CL 717140](https://go.dev/cl/717140)として実装が既に提出されており、提案者のdrakkan氏は[SFTPGo](https://github.com/drakkan/sftpgo)プロジェクトでx/crypto/sshの多段階認証機能の実装経験があります

- **後方互換性**: `AuthCallback`はオプショナルなフィールドであり、設定しない場合は従来通りの動作が保証されるため、既存コードへの影響はありません

## 関連リンク
- [Proposal Issue #76146](https://github.com/golang/go/issues/76146)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連Issue #61447: サーバー側多段階認証](https://github.com/golang/go/issues/61447)
- [関連Issue #64974: サーバー側動的認証メソッド選択](https://github.com/golang/go/issues/64974)
- [関連Issue #23461: Partial Success処理の修正](https://github.com/golang/go/issues/23461)
- [実装CL 717140](https://go.dev/cl/717140)
- [SFTPGo Project](https://github.com/drakkan/sftpgo)
- [Teleport Project](https://github.com/gravitational/teleport)

## Sources
- [golang.org/x/crypto/ssh Package Documentation](https://pkg.go.dev/golang.org/x/crypto/ssh)
- [RFC 4252: The Secure Shell (SSH) Authentication Protocol](https://datatracker.ietf.org/doc/html/rfc4252)
- [SFTPGo Dynamic User Creation Documentation](https://docs.sftpgo.com/2.6/dynamic-user-mod/)
- [SFTPGo External Authentication Documentation](https://docs.sftpgo.com/2.6/external-auth/)

## 関連リンク

- [関連Issue #23461: Partial Success処理の修正](https://github.com/golang/go/issues/23461)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連Issue #64974: サーバー側動的認証メソッド選択](https://github.com/golang/go/issues/64974)
- [Proposal Issue #76146](https://github.com/golang/go/issues/76146)
- [関連Issue #61447: サーバー側多段階認証](https://github.com/golang/go/issues/61447)
