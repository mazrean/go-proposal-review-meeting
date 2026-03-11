---
issue_number: 77631
title: "crypto/tls: remove minimum version requirement for tls.Config used for QUIC"
previous_status: active
current_status: likely_accept
changed_at: 2026-03-11T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4042167102
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77631
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4042167102
  - title: "関連Issue #63722: 元々の同様提案"
    url: https://github.com/golang/go/issues/63722
  - title: "関連Issue #77113: Config.Clone CVE-2025-68121"
    url: https://github.com/golang/go/issues/77113
  - title: "関連Issue #77363: QUIC用net.Conn設定提案"
    url: https://github.com/golang/go/issues/77363
---
## 概要
`crypto/tls` パッケージがQUIC接続に対して `tls.Config.MinVersion >= VersionTLS13` を要求している制約を撤廃し、QUIC使用時には内部的に最低バージョンをTLS 1.3にクランプする変更の提案です。これにより、HTTP/2（TLS 1.2+）とHTTP/3（TLS 1.3）を併用するデュアルスタックサーバーで `tls.Config` を共有できるようになります。

## ステータス変更
**active** → **likely_accept**

2026年3月11日のProposal Review Meetingにて、aclements氏が「QUICでは引き続きTLS 1.3以上が使われることを確認（渡された `tls.Config` 自体は変更しない）」という重要な明確化を行い、その議論を踏まえてlikely acceptと判定されました。

## 技術的背景

### 現状の問題点
RFC 9001ではQUICの暗号化にTLS 1.3が必須とされています。そのため `crypto/tls` パッケージは現在、QUIC接続を受け付ける際に `tls.Config.MinVersion` が `VersionTLS13` 以上であることを強制チェックしています。

これにより、HTTP/2（TLS 1.2+）とHTTP/3（TLS 1.3）を同一の `tls.Config` で提供したいサーバーでは、`tls.Config` をCloneしてQUIC用に `MinVersion` を上書きするワークアラウンドが必要になっていました。

しかし、Go 1.25.6のセキュリティ修正（CVE-2025-68121、Issue #77113）により `Config.Clone()` が自動生成されたセッションチケットキーをコピーしなくなったため、このClone戦略が機能しなくなりました。また、関連Issue #77363でも同様のClone問題が指摘されています。

```go
// 現状のワークアラウンド（問題のある方法）
quicConfig := tlsConfig.Clone()
quicConfig.MinVersion = tls.VersionTLS13
// ← セキュリティ修正後、セッションチケットキーが引き継がれなくなり問題が発生
```

### 提案された解決策
`crypto/tls` がQUICコンテキストで設定を使用する際、`tls.Config.MinVersion` の明示的な要求チェックを取り除き、内部的に有効最低バージョンをTLS 1.3にクランプします。渡された `tls.Config` のオリジナル値は変更せず、QUIC以外の用途（TCPなど）では引き続きそのまま適用されます。

## これによって何ができるようになるか

デュアルスタックサーバー（HTTP/2 + HTTP/3）で単一の `tls.Config` を共有できるようになります。

### コード例

```go
// Before: QUIC用にConfigをCloneする必要があった
sharedTLSConfig := &tls.Config{
    MinVersion:   tls.VersionTLS12, // HTTP/2向けにTLS 1.2を許可
    Certificates: []tls.Certificate{cert},
}

// QUIC(HTTP/3)には別途Cloneが必要
quicTLSConfig := sharedTLSConfig.Clone()
quicTLSConfig.MinVersion = tls.VersionTLS13
quicListener, _ := quic.ListenAddr(addr, quicTLSConfig, nil)

// After: 単一のConfigをそのまま共有できる
sharedTLSConfig := &tls.Config{
    MinVersion:   tls.VersionTLS12,
    Certificates: []tls.Certificate{cert},
}

// QUICは内部的にTLS 1.3以上を強制するため、Cloneが不要
quicListener, _ := quic.ListenAddr(addr, sharedTLSConfig, nil)
tcpListener := tls.NewListener(tcpConn, sharedTLSConfig)
```

実践的なメリットは以下の通りです。

- HTTP/2とHTTP/3を並行提供するサーバーのコードが簡潔になる
- セッションチケットキーの共有により、TCPとQUICで統一されたセッション再開が可能になる
- `GetCertificate` や `GetConfigForClient` などのコールバックを二重に定義する必要がなくなる

## 議論のハイライト

- **元々の類似提案（#63722）との関係**: 2023年にも同様の提案がneild氏によって支持されていたが、当時は独立したproposalプロセスが踏まれていなかった。今回はproposalとして正式化された
- **「内部クランプ」の明確化**: Review Meetingで最重要の確認事項となったのは「QUICでは引き続きTLS 1.3以上が使われる（`tls.Config` 自体を書き換えるわけではない）」という点で、これがlikely accept判定の決め手となった
- **CVE-2025-68121の影響**: `Config.Clone()` のセキュリティ修正が既存のワークアラウンドを破壊したことで、この変更の緊急性が増した
- **QUIC専用Configへの影響なし**: QUIC専用で既に `MinVersion = VersionTLS13` を設定しているConfigには何も影響がない
- **実装CLの存在**: proposalと同時にCL #745980として実装が既に提出されており、技術的な実現可能性は確認済み

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/77631)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4042167102)
- [関連Issue #63722: 元々の同様提案](https://github.com/golang/go/issues/63722)
- [関連Issue #77113: Config.Clone CVE-2025-68121](https://github.com/golang/go/issues/77113)
- [関連Issue #77363: QUIC用net.Conn設定提案](https://github.com/golang/go/issues/77363)
