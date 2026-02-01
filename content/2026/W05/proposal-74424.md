---
issue_number: 74424
title: "x/crypto/ssh: refactor signers API"
previous_status: discussions
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Proposal Issue #74424"
    url: https://github.com/golang/go/issues/74424
  - title: "関連: crypto.ContextSigner提案 #56508"
    url: https://github.com/golang/go/issues/56508
  - title: "@hslatman"
    url: https://github.com/golang/go/issues/74424#issuecomment-3127190469
  - title: "x/crypto: migrate packages to the standard library · Issue #65269"
    url: https://github.com/golang/go/issues/65269
  - title: "Review Minutes (2026-01-28)"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "@imirkin"
    url: https://github.com/golang/go/issues/74424#issuecomment-3104941123
  - title: "@meling"
    url: https://github.com/golang/go/issues/74424#issuecomment-3104310815
  - title: "proposal: x/crypto/ssh: v2 · Issue #68723"
    url: https://github.com/golang/go/issues/68723
  - title: "関連: SSHSIG support #68197"
    url: https://github.com/golang/go/issues/68197
  - title: "関連: ssh-rsa-sha2-256 handshake問題 #39885"
    url: https://github.com/golang/go/issues/39885
---

## 要約

## 概要
`x/crypto/ssh`パッケージのSignerインターフェースを統合・刷新し、将来的に標準ライブラリへ移行するための新しいV2 APIを導入する提案です。現在3つに分かれているインターフェース（Signer、AlgorithmSigner、MultiAlgorithmSigner）を、署名アルゴリズムを選択可能な単一のSignerV2に統合します。

## ステータス変更
**(新規)** → **active**

2026年1月28日のProposal Review Meetingで、この提案が**Active**ステータスに移行されました。つまり、活発な議論・レビュー段階に入り、実装に向けた具体的な検討が始まったことを意味します。

## 技術的背景

### 現状の問題点

x/crypto/sshパッケージには、歴史的経緯から3つのSignerインターフェースが存在しています：

- **Signer** - 基本の署名インターフェース
- **AlgorithmSigner** - 署名アルゴリズムを指定可能
- **MultiAlgorithmSigner** - 対応アルゴリズムのリストを返せる

これらは当初、RSA鍵が複数の署名アルゴリズム（`ssh-rsa`、`rsa-sha2-256`、`rsa-sha2-512`）をサポートする必要性から段階的に追加されました。しかし、この設計は複雑で一貫性に欠けており、特にOpenSSHが[SHA-1ベースのssh-rsaを非推奨化](https://www.openssh.org/txt/release-8.7)した現在、セキュアなデフォルト設定が難しくなっています。

また、署名操作に`context.Context`を渡せないため、KMS（Key Management Service）など外部サービスを利用する場合にタイムアウトやキャンセル、ロギング情報を伝播できないという問題があります（[#56508](https://github.com/golang/go/issues/56508)で議論）。

さらに、以下の関数群も重複・非一貫性があります：
- `NewSignerFromKey` と `NewSignerFromSigner`（DSA対応のため分離されていたが、[OpenSSH 10.0でDSAは削除](https://lwn.net/Articles/958048/)）
- PEM暗号化に非推奨の`x509.DecryptPEMBlock`を使用

### 提案された解決策

新しい**SignerV2**インターフェースを導入し、以下の特徴を持たせます：

```go
type SignerV2 interface {
    PublicKey() PublicKey
    Sign(rand io.Reader, data []byte) (*Signature, error)
    SignContext(ctx context.Context, rand io.Reader, data []byte, algorithm string) (*Signature, error)
    Algorithms() []string
    Signer() (crypto.Signer, error)
}
```

主な改善点：
1. **アルゴリズム選択の統合** - `SignContext`で署名アルゴリズムを直接指定可能
2. **Context対応** - `SignContext`によりタイムアウト・キャンセル・メタデータの伝播が可能
3. **セキュアなデフォルト** - RSA鍵ではデフォルトでSHA-1ベースの`ssh-rsa`を除外し、`rsa-sha2-256/512`のみをサポート
4. **DSAサポート削除** - OpenSSH 10.0に合わせて廃止
5. **レガシーPEM暗号化削除** - OpenSSH形式の暗号化のみサポート

## これによって何ができるようになるか

### 1. セキュアな署名がデフォルトに

従来はRSA鍵で脆弱なSHA-1ベースの`ssh-rsa`が使われる可能性がありましたが、新APIでは自動的にSHA-2ベース（`rsa-sha2-256/512`）が選択されます。古いシステムとの互換性が必要な場合のみ、明示的に`ssh-rsa`を有効化できます。

### 2. Context伝播によるKMS/HSM対応の改善

AWS KMSやハードウェアセキュリティモジュール（HSM）などクラウド・外部サービスで管理された鍵を使う場合、タイムアウトやキャンセル、分散トレーシング情報を適切に伝播できるようになります。

### 3. APIの一貫性向上

署名者の作成が一貫したパターンに：

```go
// crypto.SignerからSignerV2を作成
signer, err := ssh.NewSignerV2(cryptoSigner)

// アルゴリズムを制限
restrictedSigner, err := ssh.NewSignerV2WithAlgorithms(signer, []string{"rsa-sha2-256", "rsa-sha2-512"})

// PEMから直接パース（アルゴリズム制御も可能）
signer, err := ssh.ParsePrivateKeyV2(pemBytes, &ssh.ParsePrivateKeyV2Options{
    Passphrase: passphrase,
    SignatureAlgorithms: []string{"rsa-sha2-256", "rsa-sha2-512", "ssh-ed25519"},
})
```

### コード例

```go
// Before: 従来の書き方
signer, err := ssh.ParsePrivateKey(pemBytes)
if err != nil {
    return err
}
// RSA鍵の場合、ssh-rsaが使われる可能性がある
// アルゴリズムを制限するには追加のラッパーが必要

// After: 新APIを使った書き方
signer, err := ssh.ParsePrivateKeyV2(pemBytes, &ssh.ParsePrivateKeyV2Options{
    Passphrase: password,
    // デフォルトでセキュアなアルゴリズムのみ（ssh-rsaを除外）
})
if err != nil {
    return err
}

// Context対応の署名
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
signature, err := signer.SignContext(ctx, rand.Reader, data, "rsa-sha2-256")
```

## 議論のハイライト

- **Context対応の必要性** - [@hslatman](https://github.com/golang/go/issues/74424#issuecomment-3127190469)の指摘により`SignContext`メソッドが提案に追加され、KMS/HSMバックエンドでのContext伝播が可能に

- **レガシーサポートの範囲** - [@imirkin](https://github.com/golang/go/issues/74424#issuecomment-3104941123)は既存のレガシーPEM暗号化鍵・`ssh-rsa`アルゴリズムのサポート継続を懸念。議論の結果、`ParsePrivateKeyV2Options.SignatureAlgorithms`フィールドが追加され、必要に応じて古いアルゴリズムを明示的に有効化可能に

- **APIの柔軟性** - [@meling](https://github.com/golang/go/issues/74424#issuecomment-3104310815)がより柔軟なオプションベースのAPIを提案したが、現在の設計は`SignContext`の`algorithm`パラメータとオプション構造体のバランスを取った形に

- **後方互換性の保証** - SignerV2は既存のSignerインターフェースと互換性があり、既存のメソッド（`ServerConfig.AddHostKey`など）でそのまま使用可能。段階的な移行が可能

- **標準ライブラリ移行への準備** - x/crypto/sshパッケージは将来的に標準ライブラリへ移行予定（[#65269](https://github.com/golang/go/issues/65269)、[#68723](https://github.com/golang/go/issues/68723)）。V2サフィックスはx/crypto内でのテスト期間中のみ使用され、標準ライブラリ移行時には削除される予定

- **セキュリティファースト** - デフォルトで安全性の低いアルゴリズムを無効化し、必要な場合のみ明示的に有効化するという設計方針が一貫して維持

## 関連リンク
- [Proposal Issue #74424](https://github.com/golang/go/issues/74424)
- [Review Minutes (2026-01-28)](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [実装CL 685678](https://go.dev/cl/685678)
- [関連: SSHSIG support #68197](https://github.com/golang/go/issues/68197)
- [関連: ssh-rsa-sha2-256 handshake問題 #39885](https://github.com/golang/go/issues/39885)
- [関連: crypto.ContextSigner提案 #56508](https://github.com/golang/go/issues/56508)
- [x/crypto/sshのv2設計文書](https://go.googlesource.com/proposal/+/master/design/68723-crypto-ssh-v2.md)
- [x/crypto標準ライブラリ移行提案 #65269](https://github.com/golang/go/issues/65269)

## Sources
- [ssh package - golang.org/x/crypto/ssh - Go Packages](https://pkg.go.dev/golang.org/x/crypto/ssh)
- [OpenSSH 8.7 Release Notes](https://www.openssh.org/txt/release-8.7)
- [RFC 8332 - Use of RSA Keys with SHA-256 and SHA-512 in SSH](https://datatracker.ietf.org/doc/html/rfc8332)
- [RSA keys are not deprecated; SHA-1 signature scheme is!](https://ikarus.sg/rsa-is-not-dead/)
- [OpenSSH announces DSA-removal timeline [LWN.net]](https://lwn.net/Articles/958048/)
- [x/crypto: migrate packages to the standard library · Issue #65269](https://github.com/golang/go/issues/65269)
- [proposal: x/crypto/ssh: v2 · Issue #68723](https://github.com/golang/go/issues/68723)

## 関連リンク

- [Proposal Issue #74424](https://github.com/golang/go/issues/74424)
- [関連: crypto.ContextSigner提案 #56508](https://github.com/golang/go/issues/56508)
- [@hslatman](https://github.com/golang/go/issues/74424#issuecomment-3127190469)
- [x/crypto: migrate packages to the standard library · Issue #65269](https://github.com/golang/go/issues/65269)
- [Review Minutes (2026-01-28)](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [@imirkin](https://github.com/golang/go/issues/74424#issuecomment-3104941123)
- [@meling](https://github.com/golang/go/issues/74424#issuecomment-3104310815)
- [proposal: x/crypto/ssh: v2 · Issue #68723](https://github.com/golang/go/issues/68723)
- [関連: SSHSIG support #68197](https://github.com/golang/go/issues/68197)
- [関連: ssh-rsa-sha2-256 handshake問題 #39885](https://github.com/golang/go/issues/39885)
