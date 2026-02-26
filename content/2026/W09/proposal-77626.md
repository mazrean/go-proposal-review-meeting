---
issue_number: 77626
title: "crypto/mldsa: new package"
previous_status: 
current_status: active
changed_at: 2026-02-25T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3962620065
related_issues:
  - title: "関連Issue: crypto post-quantum roadmap #64537"
    url: https://github.com/golang/go/issues/64537
  - title: "Proposal Issue #77626"
    url: https://github.com/golang/go/issues/77626
  - title: "Review Minutes (2026-02-25)"
    url: https://github.com/golang/go/issues/33502#issuecomment-3962620065
  - title: "関連Issue: crypto/mlkem: new package #70122"
    url: https://github.com/golang/go/issues/70122
---
## 概要

`crypto/mldsa` パッケージの新規追加を提案するものです。Go 1.26でFIPS 204に基づくML-DSA（Module-Lattice-Based Digital Signature Algorithm）の内部実装が追加されており、Go 1.27でそれを公開APIとして公開することを目的としています。ML-DSAは量子コンピュータ耐性を持つポスト量子署名アルゴリズムであり、NIST標準（FIPS 204）として2024年に制定されました。

## ステータス変更

**(新規)** → **active**

2026年2月25日のWeekly Proposal Review Meeting（@aclements、@adonovan、@bradfitz、@cherrymui、@griesemer、@ianlancetaylor、@neild、@rolandshoemaker 参加）にて、本proposalが正式にactiveリストへ追加されました。Go 1.26でML-DSAの内部実装が完了しており、提案者(@FiloSottile)がすでにプレビュー実装（`filippo.io/mldsa`）を公開して動作検証を済ませていることから、Activeとして議論を開始するに十分と判断されたと考えられます。

## 技術的背景

### 現状の問題点

Go 1.24で`crypto/mlkem`（鍵カプセル化）が公開された一方、量子耐性デジタル署名であるML-DSAは`crypto/internal/fips140/mldsa`として内部実装のみ存在しており、標準ライブラリのユーザーは利用できません。外部ライブラリ（Cloudflare CIRCL、Trail of Bits実装など）を使用するか、内部パッケージへの直接アクセスに頼らざるを得ない状況です。

```go
// Before: サードパーティライブラリに依存せざるを得ない
import "github.com/cloudflare/circl/sign/mldsa/mldsa44"

scheme := mldsa44.Scheme()
pub, priv, _ := scheme.GenerateKey()
```

### 提案された解決策

FIPS 204で定義された3つのパラメータセット（ML-DSA-44、ML-DSA-65、ML-DSA-87）をサポートする`crypto/mldsa`パッケージを標準ライブラリに追加します。`PrivateKey`は`crypto.Signer`インターフェースを実装し、既存の暗号エコシステムとの統合を容易にします。また`crypto`パッケージに`MLDSAMu`という新たな`crypto.Hash`定数を追加し、External µ（事前ハッシュ済みメッセージ代表値）を使った署名に対応します。

秘密鍵はシードのみをサポートし、semi-expanded形式は「サイズが大きく、読み込みが遅く、かつより危険」という理由で意図的に除外されています。

## これによって何ができるようになるか

Go標準ライブラリのみで量子耐性署名の生成・検証が可能になります。TLSクライアント証明書、ドキュメント署名、ソフトウェア配布物の署名など、長期的なセキュリティが要求される用途でポスト量子アルゴリズムを使用できます。

### コード例

```go
// After: 標準ライブラリのみで量子耐性署名を実現
import "crypto/mldsa"

// 鍵生成
sk, err := mldsa.GenerateKey(mldsa.MLDSA44())
if err != nil {
    log.Fatal(err)
}
pk := sk.PublicKey()

// 署名（通常署名）
message := []byte("hello, post-quantum world")
sig, err := sk.Sign(nil, message, nil)
if err != nil {
    log.Fatal(err)
}

// コンテキストを使った署名（用途ごとに署名を分離）
opts := &mldsa.Options{Context: "document-signing-v1"}
sig, err = sk.Sign(nil, message, opts)

// 決定論的署名（ランダム性なし）
sig, err = sk.SignDeterministic(message, nil)

// 検証
if err := mldsa.Verify(pk, message, sig, opts); err != nil {
    log.Fatal("署名検証失敗:", err)
}

// crypto.Signer インターフェースとしても使用可能
var signer crypto.Signer = sk
```

PKIX/PKCS#8形式でのML-DSA鍵のマーシャリング・パーシングも`crypto/x509`の既存関数（`MarshalPKCS8PrivateKey`、`ParsePKCS8PrivateKey`など）を拡張して対応します（RFC 9881準拠）。

## 議論のハイライト

- **`Options.Context`の型**: `string`か`[]byte`かで議論があった。FIPS 204はバイト列として定義しているが、提案者は「イミュータブルで不変の区切り値を表す」という意味合いから`string`を採用。`crypto/ed25519`の`Options`と同様の設計。
- **`crypto.MLDSAMu`定数の追加**: External µ（外部µ）サポートのために`crypto.Hash`に新しいiota値を追加。`crypto.MD5SHA1`と同様に実装のないセンチネル値として機能し、ハードウェア実装を`crypto.Signer`経由で利用できるようにするための設計。
- **`HashML-DSA`の非サポート**: RFC 9881の「Rationale for Disallowing HashML-DSA」に従い、外部ハッシュはExternal µで代替可能とした。
- **`SignDeterministic`メソッドの設計**: 決定論的署名オプションを`Options`フィールドではなく別メソッドとして分離。`Options`は`Sign`と`Verify`の両方で共有されるが、決定論的署名は署名のみに関係するため。
- **X.509証明書サポートの範囲**: WebPKIやブラウザがポスト量子署名の取り扱いをまだ決定していないため、今回はコア機能のみ提供し、X.509証明書サポートは将来の提案に委ねる。

## 関連リンク

- [関連Issue: crypto post-quantum roadmap #64537](https://github.com/golang/go/issues/64537)
- [Proposal Issue #77626](https://github.com/golang/go/issues/77626)
- [Review Minutes (2026-02-25)](https://github.com/golang/go/issues/33502#issuecomment-3962620065)
- [関連Issue: crypto/mlkem: new package #70122](https://github.com/golang/go/issues/70122)
