---
issue_number: 24673
title: "crypto/tls: provide a way to access local certificate used to set up a connection"
previous_status: 
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "proposal: crypto/tls: provide a way to access local certificate used to set up a connection · Issue #24673 · golang/go"
    url: https://github.com/golang/go/issues/24673
  - title: "Review Minutes #33502"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
---

## 要約

## 概要
TLS接続において、リモート側の証明書（`PeerCertificates`）は取得できるのに対し、ローカル側が使用した証明書の情報を取得する方法がなかったため、`ConnectionState`に`LocalCertificate`フィールドを追加する提案です。この情報は接続のデバッグや証明書の使用統計収集に役立ちます。

## ステータス変更
**(未定義)** → **active**

このproposalは2018年4月に提出され、約7年にわたる議論を経て2026年1月28日にactiveステータスへ移行しました。長期化の理由として、当初は明確なユースケースの不足が指摘されましたが、gRPC（特にchannelz機能や外部認可サーバー連携）からの継続的な要求により、実用的な必要性が認められた経緯があります。また、2025年10月にはCL（Change List）#708515として実装が提出されており、具体的なAPI設計が確定したことも承認につながったと考えられます。

## 技術的背景

### 現状の問題点
`crypto/tls`パッケージの`ConnectionState()`メソッドは、TLS接続のセキュリティ情報を提供しますが、リモート側の証明書（`PeerCertificates`）のみを含み、ローカル側が使用した証明書の情報がありませんでした。

この問題が実務上で困る理由：
- 複数の証明書を設定している場合、どの証明書が選択されたか予測困難
- `GetCertificate`や`GetClientCertificate`コールバックが`nil`を返した場合、`NameToCertificate`にフォールバックする内部ロジックがあり、最終的な選択結果を呼び出し側から把握できない
- 接続に使用された証明書の有効期限や検証チェーンの長さなどをデバッグできない

### 提案された解決策
`ConnectionState`構造体に以下のフィールドを追加：

```go
// LocalCertificate はハンドシェイク時にローカル側が送信した証明書
// サーバー・クライアント双方で利用可能
// ハンドシェイクで証明書を交換しなかった場合はnil
// （例: クライアント証明書を提供せずに接続したクライアント側）
LocalCertificate *Certificate
```

実装は、TLS 1.2/1.3のクライアント・サーバー双方のハンドシェイク処理（計4箇所）において、証明書を選択・送信した時点で`c.localCertificate`に保存し、`ConnectionState()`取得時にこの値を返す形になっています。

## これによって何ができるようになるか

### 1. gRPC Channelzでの接続診断
gRPCの[channelz](https://grpc.io/blog/a-short-introduction-to-channelz/)機能では、接続の現在の状態をユーザーに提示しますが、ローカル証明書情報がないため完全な診断ができませんでした。この機能により以下が可能に：
- どの証明書をピアに提示したか確認
- 証明書の有効期限を監視
- 失敗した証明書の履歴をチャネルトレースに記録

### 2. 外部認可サーバーとの連携
gRPCは[Envoyの外部認可サーバー](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter)との統合を進めており、RPCごとの認可判断を行う際に、サーバー側の証明書情報（特にURI SAN、DNS SAN、Subject）を認可サーバーに送信する必要があります。`LocalCertificate`がないとこの情報を取得できず、実装不可能でした。

### 3. 証明書使用統計の収集
サービスオーナーがどの証明書がどの程度使用されているかを把握し、有効期限が近い証明書や最適でない証明書（長い検証チェーンを持つなど）の使用を検出できます。

### 4. アウトオブバンド認証との統合
サーバーから返されるカスタム認証方法名（例: "web-auth"）を検知し、ブラウザベースの認証フローへリダイレクトするなど、従来の枠を超えた認証フローを実装できます。

## コード例

### Before: 従来の静的な認証

```go
// Before: ローカル証明書情報を取得する方法がなかった
conn, err := tls.Dial("tcp", "example.com:443", config)
if err != nil {
    log.Fatal(err)
}
state := conn.ConnectionState()
// state.PeerCertificates は利用可能
// しかし、ローカル側が送信した証明書は取得不可能

// After: LocalCertificateで取得可能
conn, err := tls.Dial("tcp", "example.com:443", config)
if err != nil {
    log.Fatal(err)
}
state := conn.ConnectionState()
// ローカル証明書が送信された場合（例: mTLS）
if state.LocalCertificate != nil {
    leaf := state.LocalCertificate.Leaf
    if leaf != nil {
        log.Printf("使用した証明書: Subject=%s, NotAfter=%s",
            leaf.Subject, leaf.NotAfter)
        // URI SANを取得（Envoy外部認可などで必要）
        if len(leaf.URIs) > 0 {
            log.Printf("URI SAN: %s", leaf.URIs[0])
        }
    }
}
```

## 議論のハイライト

- **初期の懸念（2018年4月）**: @rscと@FiloSottileは、TLS 1.3での複数証明書の扱いやフィールド設計について慎重な検討が必要と指摘し、説得力のあるユースケースを求めた

- **gRPCからの継続的な要求**: 2018年、2021年、2022年、2023年、2025年と複数回にわたり、gRPC-Goチームが実用的なニーズを説明。特にproxyless service mesh（2023年に一般公開）での実運用での必要性を強調

- **実装の提出（2025年10月）**: PR #75699として実装が提出され、API設計が具体化。`LocalCertificate *Certificate`として、証明書を交換しなかった場合は`nil`とするシンプルな設計が採用された

- **設計の合理性**: `GetCertificate`/`GetClientCertificate`の結果を保存するアプローチでは不完全（内部フォールバックロジックをカバーできない）であり、`ConnectionState`にフィールドを追加するのが最も包括的な解決策と判断された

- **デバッグ用途への特化**: 実装のコミットメッセージでも「この情報は主にデバッグに有用（predominantly useful when debugging）」と明記されており、本番環境での診断・監視用途を想定

## Sources:
- [proposal: crypto/tls: provide a way to access local certificate used to set up a connection · Issue #24673 · golang/go](https://github.com/golang/go/issues/24673)
- [tls package - crypto/tls - Go Packages](https://pkg.go.dev/crypto/tls)
- [A short introduction to Channelz | gRPC](https://grpc.io/blog/a-short-introduction-to-channelz/)
- [channelz: export local certificate used by a TLS connection · Issue #4435 · grpc/grpc-go](https://github.com/grpc/grpc-go/issues/4435)
- [External Authorization — envoy 1.38.0-dev-b859f4 documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter)

## 関連リンク

- [proposal: crypto/tls: provide a way to access local certificate used to set up a connection · Issue #24673 · golang/go](https://github.com/golang/go/issues/24673)
- [Review Minutes #33502](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
