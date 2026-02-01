---
issue_number: 24673
title: "crypto/tls: provide a way to access local certificate used to set up a connection"
previous_status: discussions
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/24673
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "TLS 1.3 Post-Handshake Authentication提案"
    url: https://github.com/golang/go/issues/40521
---

## 要約

## 概要
このproposalは、TLSコネクションで実際に使用されたローカル証明書（自分側の証明書）を`ConnectionState`構造体から取得できるようにする機能追加を提案しています。現在は`PeerCertificates`でリモート側の証明書は取得できますが、ローカル側の証明書情報にアクセスする手段がないため、デバッグや統計収集が困難という問題がありました。

## ステータス変更
**(新規)** → **active**

2026年1月28日に、proposal review groupによって「active」ステータスに移行されました。これは、2018年4月の提案から約8年越しの進展です。この決定の背景には、gRPCチームからの継続的な要望と、具体的なユースケースの明確化（特に外部認証サーバーとの連携要件）があります。CL 708515として実装が既に提出されており、技術的な実現可能性も確認されています。

## 技術的背景

### 現状の問題点
Go言語の`crypto/tls`パッケージでは、`Conn.ConnectionState()`メソッドによってTLSコネクションのセキュリティ情報を取得できます。しかし、この`ConnectionState`構造体には以下の制限があります:

- **`PeerCertificates`フィールド**: リモート側（接続相手）の証明書は取得可能
- **ローカル証明書のフィールド**: 存在しない

```go
// 現在の状況: リモート証明書は取得できる
state := conn.ConnectionState()
remoteCerts := state.PeerCertificates // OK

// しかし、ローカル証明書を取得する方法がない
// localCerts := state.LocalCertificates // このようなフィールドは存在しない
```

TLSでは、サーバー側が複数の証明書を持っている場合、クライアントのSNI（Server Name Indication）やその他の要件に応じて、どの証明書を使用するかが動的に決定されます。また、`GetCertificate`や`GetClientCertificate`コールバックを使用している場合、プログラマが直接制御していても、実際にどの証明書が選択されたかを後から確認する標準的な方法がありません。

### 提案された解決策
`ConnectionState`構造体に新しいフィールドを追加する提案が議論されています。初期の議論では以下のような形式が検討されました:

```go
type ConnectionState struct {
    // 既存のフィールド...
    PeerCertificates []*x509.Certificate

    // 提案されている新フィールド
    LocalCertificates []*x509.Certificate // または LocalCertificate *Certificate
}
```

CL 708515では、実際に使用された証明書チェーンを含む形で実装されているようです。

## これによって何ができるようになるか

この機能により、以下のような実践的なメリットが得られます:

### 1. デバッグとトラブルシューティング
サービス運用者は、実際にどの証明書が使用されたかを確認でき、以下のような問題を発見できます:
- 期限切れ間近の証明書が使用されている
- 意図しない証明書が選択されている
- 証明書チェーンが最適でない（検証パスが長すぎる）

### 2. gRPCのChannelz機能でのセキュリティ情報提供
gRPCの[Channelz](https://github.com/grpc/proposal/blob/master/A14-channelz.md)は、コネクションの現在状態をユーザーに提示する診断機能です。この機能で以下が可能になります:
- クライアントに対して提示したサーバーのアイデンティティ確認
- 証明書の有効期限の監視
- 過去に試行して失敗した証明書の記録（チャネルトレース）

### 3. 外部認証サーバーとの連携
gRPCは、Envoyの外部認証機能に相当する機能を実装中で、以下の情報を外部認証サーバーに送信する必要があります:
- サーバー証明書の最初のURI SAN（設定されている場合）
- または、最初のDNS SAN（URI SANがない場合）
- または、RFC 2253形式のSubjectフィールド

現状では`ConnectionState`からローカル証明書にアクセスできないため、この実装が不可能です。

### 4. 証明書使用統計の収集
サービス所有者は、どの証明書がどの程度使用されているかの統計を収集でき、証明書のローテーション計画やキャパシティプランニングに活用できます。

### コード例

```go
// Before: 従来は設定から証明書を参照するしかない（どれが使われたかは不明）
config := &tls.Config{
    Certificates: []tls.Certificate{cert1, cert2, cert3},
    GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
        // 複雑なロジックで証明書を選択
        return selectCert(info), nil
    },
}

conn, _ := tls.Dial("tcp", "example.com:443", config)
state := conn.ConnectionState()
// ここでローカル証明書を取得する方法がない

// After: 新APIを使った書き方（提案）
conn, _ := tls.Dial("tcp", "example.com:443", config)
state := conn.ConnectionState()

// 実際に使用されたローカル証明書にアクセス可能
localCerts := state.LocalCertificates
if len(localCerts) > 0 {
    cert := localCerts[0]
    fmt.Printf("使用された証明書: Subject=%v\n", cert.Subject)
    fmt.Printf("有効期限: %v\n", cert.NotAfter)

    // URI SANまたはDNS SANを外部認証サーバーに送信
    if len(cert.URIs) > 0 {
        identity := cert.URIs[0].String()
    } else if len(cert.DNSNames) > 0 {
        identity := cert.DNSNames[0]
    } else {
        identity := cert.Subject.String() // RFC 2253形式
    }
}
```

## 議論のハイライト

- **TLS 1.3との互換性**: Filippo Valsorda氏（@FiloSottile）が、TLS 1.3でクライアント証明書が複数回要求される可能性について確認していました。TLS 1.3ではポストハンドシェイク認証機能がありますが、Goは現在この機能をサポートしていないため、実質的な問題にはなりません。

- **証明書選択ロジックの複雑性**: `GetCertificate`や`GetClientCertificate`が`nil, nil`を返した場合、TLSパッケージは`NameToCertificate`にフォールバックします。この複雑な選択ロジックのため、プログラマが事前に使用される証明書を予測することは困難です。

- **gRPCからの継続的な要望**: 2018年の提案以降、gRPCチームから2021年、2022年、2023年、2025年と繰り返し優先度向上の要請がありました。特に、Googleのproxyless service mesh（Traffic Director）が公開されたことで、実運用環境でのニーズが高まりました。

- **実装アプローチ**: 当初は単一の証明書を返す`LocalCertificate *Certificate`が検討されましたが、証明書チェーン全体を含める可能性を考慮して`LocalCertificates []*x509.Certificate`の形式が有力視されています。

- **承認タイミング**: 2025年10月にCL 708515が提出され、その約3ヶ月後の2026年1月にactiveステータスに移行しました。長年の要望と具体的なユースケース、実装の実現可能性が揃ったことで、承認に向けて前進しています。

## 関連リンク
- [Proposal Issue](https://github.com/golang/go/issues/24673)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [実装CL 708515](https://go.dev/cl/708515)
- [gRPC関連Issue](https://github.com/grpc/grpc-go/issues/4435)
- [gRPC Channelz提案](https://github.com/grpc/proposal/blob/master/A14-channelz.md)
- [Envoy External Authorization](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter)
- [TLS 1.3 Post-Handshake Authentication提案](https://github.com/golang/go/issues/40521)

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/24673)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [TLS 1.3 Post-Handshake Authentication提案](https://github.com/golang/go/issues/40521)
