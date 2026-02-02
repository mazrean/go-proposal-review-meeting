---
issue_number: 74424
title: "x/crypto/ssh: refactor signers API"
previous_status: 
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "標準ライブラリ移行proposal (#68723)"
    url: https://github.com/golang/go/issues/68723
  - title: "関連: SSHSIG形式サポート提案 (#68197)"
    url: https://github.com/golang/go/issues/68197
  - title: "関連: MultiAlgorithmSigner追加 (#52132)"
    url: https://github.com/golang/go/issues/52132
  - title: "crypto.Signer context対応提案 (#56508)"
    url: https://github.com/golang/go/issues/56508
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/74424
---

## 要約

## 概要

このproposalは、`x/crypto/ssh`パッケージの標準ライブラリ移行に向けて、署名者（Signer）APIを大幅に再設計するものです。現在3つに分かれているSigner関連インターフェース（Signer、AlgorithmSigner、MultiAlgorithmSigner）を統合し、アルゴリズム選択とコンテキスト伝播に対応した`SignerV2`インターフェースに一本化します。

## ステータス変更

**(未設定)** → **active**

2026年1月28日のProposal Review Meetingにおいて、本proposalが「active」ステータスに昇格しました。議事録には「added to minutes」とのみ記載されており、積極的な議論を継続するための活性化措置と見られます。この決定は、x/crypto/sshの標準ライブラリ移行（#68723）という大きな文脈の中で、重要なAPI改善として位置づけられています。

## 技術的背景

### 現状の問題点

`x/crypto/ssh`パッケージは後方互換性を保つために、RSA鍵の複数の署名アルゴリズム対応を段階的に追加してきた結果、以下の3つのインターフェースが乱立しています。

- **Signer**: 基本的な署名インターフェース
- **AlgorithmSigner**: アルゴリズムを指定して署名できるインターフェース
- **MultiAlgorithmSigner**: 対応アルゴリズムを列挙できるインターフェース

また、Signer作成のための関数も4つに分かれています。

```go
NewSignerFromKey(key interface{}) (Signer, error)
NewSignerFromSigner(signer crypto.Signer) (Signer, error)
NewCertSigner(cert *Certificate, signer Signer) (Signer, error)
NewSignerWithAlgorithms(signer AlgorithmSigner, algorithms []string) (MultiAlgorithmSigner, error)
```

`NewSignerFromKey`が`interface{}`を受け取る理由は、DSA鍵（`*dsa.PrivateKey`）対応のためでしたが、DSAはOpenSSH 10.0（2025年4月リリース）で完全に削除されており、もはや不要な複雑さです。

さらに、秘密鍵パース関数も4つあり、冗長性が見られます。

```go
ParsePrivateKey(pemBytes []byte) (Signer, error)
ParsePrivateKeyWithPassphrase(pemBytes, passphrase []byte) (Signer, error)
ParseRawPrivateKey(pemBytes []byte) (interface{}, error)
ParseRawPrivateKeyWithPassphrase(pemBytes, passphrase []byte) (interface{}, error)
```

### 提案された解決策

新しい`SignerV2`インターフェースは、これらの複雑性を解消し、以下の機能を提供します。

```go
type SignerV2 interface {
    PublicKey() PublicKey

    // 後方互換性のための従来型メソッド
    Sign(rand io.Reader, data []byte) (*Signature, error)

    // コンテキストとアルゴリズム選択に対応した新メソッド
    SignContext(ctx context.Context, rand io.Reader, data []byte, algorithm string) (*Signature, error)

    // 対応アルゴリズムをリスト表示
    Algorithms() []string

    // 基礎となるcrypto.Signerへのアクセス（HSM等で有用）
    Signer() (crypto.Signer, error)
}
```

Signer作成関数も簡潔になります。

```go
// crypto.SignerからSignerV2を作成（RSA鍵の場合はSHA-1を除外）
func NewSignerV2(signer crypto.Signer) (SignerV2, error)

// 対応アルゴリズムを制限したSignerV2を作成
func NewSignerV2WithAlgorithms(signer SignerV2, algorithms []string) (SignerV2, error)

// 証明書付きSignerV2を作成
func NewCertificateSignerV2(cert *Certificate, signer SignerV2) (SignerV2, error)
```

秘密鍵のパース・マーシャルは、オプション構造体を用いた統一的な設計になります。

```go
type ParsePrivateKeyV2Options struct {
    Passphrase string
    SignatureAlgorithms []string  // デフォルトで安全なアルゴリズムのみ許可
}

func ParsePrivateKeyV2(pemBytes []byte, options *ParsePrivateKeyV2Options) (SignerV2, error)

type MarshalPrivateKeyV2Options struct {
    Comment string
    Passphrase string  // 設定時はOpenSSH形式で暗号化
    SaltRounds int     // 鍵導出の反復回数（デフォルト24）
}

func MarshalPrivateKeyV2(key SignerV2, options *MarshalPrivateKeyV2Options) (*pem.Block, error)
```

重要な変更点として、RFC 1423準拠の旧式PEM暗号化（DES-CBC、AES-CBC使用）は非対応となり、OpenSSH形式の暗号化のみをサポートします。

## これによって何ができるようになるか

1. **統一されたシンプルなAPI**: 3つのインターフェースと4つのコンストラクタを、1つのインターフェースと3つのコンストラクタに集約。コードの見通しが大幅に改善されます。

2. **コンテキスト伝播のサポート**: HSMやクラウドKMS（AWS KMS、Google Cloud KMS等）を使用する際に、タイムアウト、キャンセル、トレース情報をSigner実装に伝播できるようになります。これは標準ライブラリのcrypto.Signer向け提案（#56508）と整合性があります。

3. **セキュアなデフォルト設定**: RSA鍵に対してSHA-1ベースの`ssh-rsa`アルゴリズムをデフォルトで無効化し、安全な`rsa-sha2-256`、`rsa-sha2-512`のみを許可。レガシーアルゴリズムが必要な場合は明示的に指定が必要です。

4. **柔軟なアルゴリズム制限**: `SignatureAlgorithms`オプションにより、鍵の種類を問わず使用可能なアルゴリズムを事前に指定可能。以下のようなコードが実現できます。

### コード例

```go
// Before: 従来の書き方（複雑な型アサーションが必要）
signer, err := ssh.ParsePrivateKey(pemBytes)
if err != nil {
    return err
}
// MultiAlgorithmSignerでアルゴリズムを制限したい場合
if algSigner, ok := signer.(ssh.AlgorithmSigner); ok {
    signer, err = ssh.NewSignerWithAlgorithms(algSigner,
        []string{ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSASHA512})
}

// After: 新APIを使った書き方（オプションでアルゴリズム制限が可能）
signer, err := ssh.ParsePrivateKeyV2(pemBytes, &ssh.ParsePrivateKeyV2Options{
    SignatureAlgorithms: []string{
        ssh.KeyAlgoRSASHA256,
        ssh.KeyAlgoRSASHA512,
        ssh.KeyAlgoED25519,
        // ssh-rsaは含めない（セキュア設定）
    },
})
// 鍵種別に関わらず、対応するアルゴリズムのみがフィルタされる
```

```go
// Before: コンテキストを伝播できない
signature, err := signer.Sign(rand.Reader, data)

// After: コンテキストを使ってタイムアウトやキャンセルを制御
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
signature, err := signer.SignContext(ctx, rand.Reader, data, "rsa-sha2-256")
```

```go
// Before: 暗号化された鍵のマーシャル（複雑な手順）
block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
// パスフレーズ付き暗号化は更に複雑

// After: オプション構造体でシンプルに
block, err := ssh.MarshalPrivateKeyV2(signer, &ssh.MarshalPrivateKeyV2Options{
    Comment: "my-key",
    Passphrase: "secret",
    SaltRounds: 32, // セキュリティ強化
})
```

## 議論のハイライト

- **コンテキスト対応の要請**: @hslatmanより、KMS利用時にコンテキスト伝播が必須との指摘を受け、`SignContext`メソッドが提案に追加されました（標準ライブラリの#56508提案との整合性を考慮）。

- **旧式PEM暗号化のサポート**: @imirkinより、既存の暗号化PEM鍵の互換性喪失への懸念が示されましたが、RFC 1423の暗号化方式（DES-CBC等）は既に標準ライブラリで非推奨（`x509.DecryptPEMBlock`）であり、OpenSSH形式への移行が推奨されています。ただし、この点は他のメンテナーとの議論継続事項です。

- **レガシーアルゴリズムの扱い**: `ssh-rsa`（SHA-1使用）をデフォルトで無効化する方針に対し、@imirkinより「安全でないアルゴリズムを有効化する際の簡便性」の要望があり、`ParsePrivateKeyV2Options.SignatureAlgorithms`による柔軟な制御が追加されました。これにより、鍵タイプを意識せずに許可アルゴリズムを指定可能です。

- **`Signer()`メソッドの命名**: @hslatmanより、`PrivateKey()`よりも`Signer()`の方が返り値の型（`crypto.Signer`）に合致するとの提案があり、採用されました。

- **エージェントAPIのコンテキスト対応**: @arianvpより、`ServeAgent`メソッドも`SignerV2`と整合的にコンテキスト対応すべきとの要望があり、@drakkanは別proposalで対応予定と回答しています。

- **V2接尾辞の理由**: `x/crypto/ssh`での移行期間中の新旧API共存のために`V2`接尾辞を使用。標準ライブラリ移行後は接尾辞を削除する計画です。

## 関連リンク

- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [標準ライブラリ移行proposal (#68723)](https://github.com/golang/go/issues/68723)
- [関連: SSHSIG形式サポート提案 (#68197)](https://github.com/golang/go/issues/68197)
- [関連: MultiAlgorithmSigner追加 (#52132)](https://github.com/golang/go/issues/52132)
- [crypto.Signer context対応提案 (#56508)](https://github.com/golang/go/issues/56508)
- [Proposal Issue](https://github.com/golang/go/issues/74424)
