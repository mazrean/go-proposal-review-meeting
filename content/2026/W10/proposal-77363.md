---
issue_number: 77363
title: "crypto/tls: allow QUIC to configure net.Conn used on ClientHelloInfo"
previous_status: 
current_status: active
changed_at: 2026-03-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
related_issues:
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
  - title: "関連Issue: Config.Clone セッションチケット問題 (CVE-2025-68121)"
    url: https://github.com/golang/go/issues/77113
  - title: "関連Issue: QUIC最小TLSバージョン要件の削除提案"
    url: https://github.com/golang/go/issues/77631
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77363
---
## 概要

`crypto/tls` パッケージの `QUICConfig` に `ClientHelloInfoConn` フィールドを追加し、QUICスタックがTLSハンドシェイク中に `ClientHelloInfo.Conn` として使用される `net.Conn` を外部から供給できるようにするための提案です。

## ステータス変更

**(新規)** → **active**

2026年3月4日の週次Proposal Reviewミーティングにてactiveカラムに追加され、今後の正式レビューの対象となりました。提案内容への反発はなく、実装CLも既に提出されています。

## 技術的背景

### 現状の問題点

`crypto/tls` の `GetCertificate` および `GetConfigForClient` コールバックは、`tls.ClientHelloInfo` 構造体を受け取ります。この構造体には `Conn net.Conn` フィールドが含まれており、接続元・接続先のアドレス情報などを提供します。

QUICはUDPベースのプロトコルであり、TCPの `net.Conn` が持つ `Read`/`Write`/`Close` などのメソッドは意味を持ちません。そのため quic-go などのQUICスタックは、`LocalAddr` と `RemoteAddr` だけを返すフェイクの `net.Conn` をコールバックに注入するため、`tls.Config` をクローンする必要がありました。

しかし、Go 1.25.6 のセキュリティ修正（CVE-2025-68121, Issue #77113）により、`Config.Clone()` が自動生成されたセッションチケットキーをコピーしなくなりました。これにより、クローンを使ったQUICスタックのワークアラウンドがセッション再開（Session Resumption）を破壊するという問題が発生しました。

```go
// Before: quic-go のワークアラウンド（問題のある方法）
clonedConfig := tlsConfig.Clone() // セキュリティ修正後、セッションチケットキーがコピーされない
clonedConfig.GetCertificate = func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
    info.Conn = &fakeConn{localAddr: localAddr, remoteAddr: remoteAddr}
    return originalGetCertificate(info)
}
```

### 提案された解決策

`tls.QUICConfig` に `ClientHelloInfoConn net.Conn` フィールドを追加します。QUICスタックはアドレス情報のみを実装したフェイク `net.Conn` をここに設定するだけでよく、`crypto/tls` 側がそれを `ClientHelloInfo.Conn` に直接使用します。

```go
// After: 新しいAPIを使った書き方
type QUICConfig struct {
    TLSConfig           *Config
    EnableSessionEvents bool
    ClientHelloInfoConn net.Conn // ハンドシェイク中の ClientHelloInfo.Conn に使用
}

// QUICスタック側での使用例
quicConn, err := tls.QUICServer(context.Background(), &tls.QUICConfig{
    TLSConfig:           tlsConfig, // クローン不要
    ClientHelloInfoConn: &fakeConn{localAddr: localAddr, remoteAddr: remoteAddr},
})
```

## これによって何ができるようになるか

**TLS設定のクローンが不要になる**: QUICスタックは `tls.Config` をクローンせずにそのまま使用できるため、セッション再開の破損問題が根本的に解消されます。

**アドレス情報の正確な提供**: `GetCertificate` や `GetConfigForClient` コールバック内でQUIC接続のローカル・リモートアドレスを正しく参照できます。SNI（Server Name Indication）ベースの証明書選択などの実装が適切に機能します。

**将来の拡張性**: `net.Conn` インターフェース全体を受け付けることで、将来的にアドレス以外のメタデータもQUICスタックから提供できる柔軟性が生まれます。

## 議論のハイライト

- 当初提案者は2つの案を提示していた。Option 1は `HandshakeConn net.Conn` フィールド（`net.Conn` 全体を受け付ける）、Option 2は `LocalAddr` と `RemoteAddr` を個別フィールドとして追加する案。提案者自身はOption 1をわずかに支持していた。
- `neild`（Go cryptoチームのコントリビュータ）はOption 1の方向性を支持しつつ、フィールド名を `ClientHelloInfoConn` とすることを提案。用途が明確になるという理由からこの名前が採用された。
- 実装CL（go.dev/cl/745720）は既に提出されており、quic-go（`quic-go/quic-go#5571`）での動作確認も完了している。
- この提案と並行して、QUICのTLS最小バージョン要件を緩和する関連提案（Issue #77631）も提出されており、これら2つを合わせることでquic-goがTLS設定のクローンを一切行わなくて済む状態が実現できると確認されている。
- ハンドシェイク中はQUIC RFC 9000に定義された接続マイグレーションは発生しないため、5タプル（送受信アドレス・ポート・プロトコル）が固定されており、この提案には接続マイグレーションに関する問題はないとされている。

## 関連リンク

- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4000757564)
- [関連Issue: Config.Clone セッションチケット問題 (CVE-2025-68121)](https://github.com/golang/go/issues/77113)
- [関連Issue: QUIC最小TLSバージョン要件の削除提案](https://github.com/golang/go/issues/77631)
- [Proposal Issue](https://github.com/golang/go/issues/77363)
