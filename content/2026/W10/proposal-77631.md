---
issue_number: 77631
title: "crypto/tls: remove minimum version requirement for tls.Config used for QUIC"
previous_status: 
current_status: active
changed_at: 2026-03-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
related_issues:
  - title: "関連Issue #63722: crypto/tls: don't require Config to set MinVersion = TLS13 when using QUIC"
    url: https://github.com/golang/go/issues/63722
  - title: "関連Issue #77363: QUIC向けClientHelloInfo.Connの設定"
    url: https://github.com/golang/go/issues/77363
  - title: "関連Issue #77113: Config.Clone CVE-2025-68121"
    url: https://github.com/golang/go/issues/77113
  - title: "Proposal Issue #77631"
    url: https://github.com/golang/go/issues/77631
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
---
## 概要

`crypto/tls` パッケージでは、QUICコネクションに使用する `tls.Config` に対して `MinVersion >= VersionTLS13` を明示的に設定することを要求していますが、このproposalはその要件を撤廃し、QUIC使用時には内部で自動的にTLS 1.3以上にクランプする変更を提案するものです。

## ステータス変更
**(新規)** → **active**

2026年3月4日、@aclementsによりproposalがactiveステータスに移行されました。週次のproposal reviewミーティングでの審査対象となります。同様の問題を扱う関連issue #63722が `NeedsFix` ラベルで長期間オープンだったことや、`Config.Clone` のセキュリティ問題（CVE-2025-68121、issue #77113）によりcloneを用いたワークアラウンドへのリスクが高まったことが、このタイミングでの正式proposal化につながったと考えられます。

## 技術的背景

### 現状の問題点

RFC 9001に基づき、QUICはTLS 1.3のみをサポートします。Goの `crypto/tls` は、`QUICClient` および `QUICServer` を使用する際、`tls.Config.MinVersion` に `VersionTLS13` 以上を明示的に設定することを要求し、設定がない場合はエラーを返します。

```
"tls: Config MinVersion must be at least TLS 1.3"
```

この制約により、TLS 1.2以上を許可するTCP（HTTP/2）とTLS 1.3のみのQUIC（HTTP/3）を同一サーバーで提供する際に、同じ `tls.Config` を共有できません。現状のワークアラウンドとして、quic-goなどのライブラリはQUIC用に `Config.Clone()` を呼び出してMinVersionを上書きしていますが、この方法には以下の問題があります。

- `Config.Clone()` のCVE-2025-68121（セッションチケットキーの意図しない共有）による脆弱性リスク
- issue #77363で報告された、`ClientHelloInfo.Conn` に関するQUIC固有のワークアラウンドの複雑化
- cloneのオーバーヘッドと保守コストの増大

### 提案された解決策

`QUICClient` / `QUICServer` のコンテキストで使用される場合、`tls.Config.MinVersion` の値は変更せず、内部的に有効な最小バージョンをTLS 1.3にクランプする。これにより、元の `Config` 値は非QUICコネクション（TCPなど）向けに保持されます。

## これによって何ができるようになるか

HTTP/2とHTTP/3のデュアルスタックサーバーを、単一の `tls.Config` で管理できるようになります。設定の単純化と、不要なcloneの排除が可能になります。

### コード例

```go
// Before: HTTP/2とHTTP/3の共有設定（ワークアラウンドが必要）
baseCfg := &tls.Config{
    MinVersion:   tls.VersionTLS12, // HTTP/2用にTLS 1.2を許可
    Certificates: []tls.Certificate{cert},
}

// QUIC用にcloneしてMinVersionを上書きする必要があった
quicCfg := baseCfg.Clone()
quicCfg.MinVersion = tls.VersionTLS13 // これがないとQUICがエラーを返す

// HTTP/3サーバー
quicListener, err := tls.QUICServer(quicCfg)

// HTTP/2サーバー
tcpListener := tls.NewListener(tcpConn, baseCfg)

// After: 単一のConfigを共有可能
sharedCfg := &tls.Config{
    MinVersion:   tls.VersionTLS12, // HTTP/2ではTLS 1.2も許可
    Certificates: []tls.Certificate{cert},
}

// QUICコンテキストでは内部的にTLS 1.3にクランプされる（エラーなし）
quicListener, err := tls.QUICServer(sharedCfg)

// TCPではMinVersion=TLS 1.2のままで動作
tcpListener := tls.NewListener(tcpConn, sharedCfg)
```

実践的なユースケースとして以下が挙げられます。

1. **デュアルスタックHTTPサーバー**: 同一ホストでHTTP/2（TLS 1.2+）とHTTP/3（TLS 1.3のみ）を同じ証明書・設定で提供
2. **証明書ローテーション**: 一元管理された `tls.Config` をQUICとTCPで共有し、証明書更新を一箇所で完結
3. **マイグレーション期間中の後方互換性**: TLS 1.2クライアント向けのTCPフォールバックを維持しながらQUICを導入

## 議論のハイライト

- 同一内容の問題は2023年にissue #63722として報告済みで、`NeedsFix` ラベルが付与されていたが、正式なproposalとして整理されていなかった
- @neild（Googleのネットワーキングチームメンバー）は2023年の時点で「ユーザーにMinVersionの設定を強制することに利点はない。QUICとTCPで同じConfigを再利用できるよう暗黙的にTLS 1.3に引き上げることは合理的」と支持を表明
- CVE-2025-68121（issue #77113）により `Config.Clone()` を用いたワークアラウンドがセキュリティリスクを抱えることが明確になり、根本的な解決の必要性が高まった
- issue #77363（`ClientHelloInfo.Conn` のQUIC対応）とセットで、QUIC関連のcloneワークアラウンドを一掃する設計変更として位置付けられている
- QUICのみの `Config` には変更がなく、既存のQUICコードへの影響は最小限

## 関連リンク

- [関連Issue #63722: crypto/tls: don't require Config to set MinVersion = TLS13 when using QUIC](https://github.com/golang/go/issues/63722)
- [関連Issue #77363: QUIC向けClientHelloInfo.Connの設定](https://github.com/golang/go/issues/77363)
- [関連Issue #77113: Config.Clone CVE-2025-68121](https://github.com/golang/go/issues/77113)
- [Proposal Issue #77631](https://github.com/golang/go/issues/77631)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4000757564)
