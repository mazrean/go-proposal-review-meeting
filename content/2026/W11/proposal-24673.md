---
issue_number: 24673
title: "crypto/tls: provide a way to access local certificate used to set up a connection"
previous_status: active
current_status: likely_accept
changed_at: 2026-03-11T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4042167102
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/24673
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4042167102
---
## 概要

`crypto/tls` パッケージの `ConnectionState` 構造体に `LocalCertificate` フィールドを追加し、TLS ハンドシェイクで使用されたローカル証明書へのアクセスを可能にする提案です。現在は接続相手のピア証明書（`PeerCertificates`）は取得できますが、ローカル側が提示した証明書を事後に取得する手段がありませんでした。

## ステータス変更
**active** → **likely_accept**

2026年3月11日に @aclements がweekly proposal review meetingの結果として "likely accept" と判定しました。grpc-Go チームとの調整を経て、セッション再開時（DidResume=true）には `nil` とする仕様が合意され、残存する懸念事項がほぼ解消されたことが承認の決め手となりました。

## 技術的背景

### 現状の問題点

`tls.Conn.ConnectionState()` は `PeerCertificates`（相手側証明書）を返すものの、ローカル側が実際に使用した証明書を返すフィールドが存在しません。ローカル証明書の選択は `GetCertificate` / `GetClientCertificate` コールバック、`NameToCertificate` によるSNIマッチング、設定された証明書リストのフォールバックなど複数のパスを経て動的に行われるため、事前にどの証明書が選ばれるかを確実に予測できない場合があります。

```go
// 現状: 接続後にローカル証明書を知る方法がない
state := tlsConn.ConnectionState()
fmt.Println(state.PeerCertificates)  // ピア証明書は取得可能
// state.LocalCertificate が存在しないため、
// GetCertificate コールバックで自前で保存するしかない
```

### 提案された解決策

`tls.ConnectionState` 構造体に以下のフィールドを追加します。

```go
type ConnectionState struct {
    // 既存フィールド...

    // LocalCertificate is the certificate presented to the peer, if any, during
    // the handshake. This field is only populated for connections which are not
    // resumed (DidResume is false).
    LocalCertificate *Certificate
}
```

実装CLは [go.dev/cl/708515](https://go.dev/cl/708515)（PR #75699）として既に提出されています。

## これによって何ができるようになるか

TLS 接続確立後に、その接続で実際に使用されたローカル側の証明書を `ConnectionState` から直接参照できるようになります。

### コード例

```go
// Before: ワークアラウンドとして GetCertificate コールバックで証明書を記録
var usedCert *tls.Certificate
config := &tls.Config{
    GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
        cert, err := selectCert(hello)
        usedCert = cert  // 自前で保存が必要
        return cert, err
    },
}

// After: ConnectionState から直接取得
conn, _ := tls.Dial("tcp", "example.com:443", config)
state := conn.ConnectionState()
if cert := state.LocalCertificate; cert != nil {
    // 証明書の有効期限確認
    fmt.Println("証明書の有効期限:", cert.Leaf.NotAfter)
    // SANの確認
    fmt.Println("DNS SANs:", cert.Leaf.DNSNames)
}
```

実践的なユースケースとして以下が挙げられます。

- **gRPC channelz**: 接続状態の監視ダッシュボードでローカル・リモート双方の証明書情報を表示し、期限切れ前の証明書を検出する
- **外部認可サーバー連携**: gRPC の External Authorization（ext_authz）パターンで、サーバー証明書のURI SAN / DNS SAN / Subject を認可リクエストに含める
- **証明書使用状況の統計収集**: 複数証明書を持つサーバーで、実際にどの証明書が選択されているかを計測し、不均一なロードバランシングや期限切れ間近の証明書を検知する

## 議論のハイライト

- **2018年の初期議論**: @rsc が @agl・@FiloSottile と議論し、`LocalCertificate *Certificate` の追加を検討したが、ユースケースの正当性確認が必要として保留された
- **gRPC チームによる継続的な要望**: 2018年〜2025年にかけて gRPC-Go チーム（@lyuxuan、@easwars、@ginayeh）が channelz 機能やプロキシレスサービスメッシュ、ext_authz 対応のために継続的に要望を発信し、具体的なユースケースを示した
- **セッション再開時の扱い**: @FiloSottile が「再開された接続でこのフィールドをどう扱うか」を問い、セッションチケットのサイズ増加を避けるため再開時（`DidResume=true`）は `nil` とすることで合意した
- **設定可能性の検討と見送り**: セッションチケットに証明書情報を含めるかどうかを設定できるオプション（`Config` フィールド追加）も議論されたが、現時点では見送り、将来的な問題が発生した場合に再検討する方針となった
- **約7年越しの提案**: 2018年提出から約7年間保留されていたが、gRPC の ext_authz 対応という具体的かつ強い需要が後押しとなり、2025年10月に実装CLが提出され、2026年1月にレビューキューに追加された

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/24673)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4042167102)
