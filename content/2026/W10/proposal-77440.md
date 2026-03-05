---
issue_number: 77440
title: "net/http: pluggable HTTP/3"
previous_status: 
current_status: active
changed_at: 2026-03-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77440
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
  - title: "関連Issue: net/http HTTP/3サポート追跡 #32204"
    url: https://github.com/golang/go/issues/32204
  - title: "関連Issue: x/net/http3 実験的実装 #70914"
    url: https://github.com/golang/go/issues/70914
  - title: "関連Issue: HTTP version selection API #67814"
    url: https://github.com/golang/go/issues/67814
---
## 概要

`net/http` パッケージの `Transport` および `Server` に、外部パッケージが提供するHTTP/3実装を差し込める仕組みを追加するproposalです。HTTP/3そのものを標準ライブラリに取り込むのではなく、外部実装を既存の`net/http` APIと連携させるための「プラグイン機構」を整備することを目的としています。

## ステータス変更

**(新規)** → **active**

2026年3月4日に開催されたProposal Review Meeting（@aclements, @adonovan, @bradfitz, @cherrymui, @griesemer, @ianlancetaylor, @neild, @rolandshoemaker 参加）で議事録に追加され、週次レビューの対象となるactiveステータスに移行しました。同時期に基本的な実装CL（go.dev/cl/740120）も提出されており、議論と実装が並行して進んでいます。

## 技術的背景

### 現状の問題点

HTTP/3（QUICトランスポート上のHTTP）は既にサードパーティライブラリ（`github.com/quic-go/quic-go` など）として実装が存在しますが、`net/http.Transport` や `net/http.Server` と統合する標準的な方法がありませんでした。HTTP/3を使いたいユーザーは `net/http` の既存コード（コネクションプーリング、ミドルウェア等）を流用できず、HTTP/3専用の独自クライアント/サーバーを別途構築する必要がありました。

HTTP/2は `golang.org/x/net/http2` として実装し、後に標準ライブラリへ取り込みました。しかしこの経緯で生じた依存関係の複雑さはいまだ整理中であり、HTTP/3について同じアプローチは取らないことが明確にされています。HTTP/3はQUICスタックを含む重量な依存であり、全 `net/http` ユーザーに強制すべきではないためです。

### 提案された解決策

エクスポートAPIの変更は最小限に抑え、既存の仕組みを拡張することで実現します。

**1. `http.Protocols` へのHTTP/3追加**

唯一のエクスポートAPIの追加です。

```go
// HTTP3 reports whether p includes HTTP/3.
func (p Protocols) HTTP3() bool

// SetHTTP3 adds or removes HTTP/3 from p.
func (p *Protocols) SetHTTP3(ok bool)
```

**2. クライアント側の登録（`Transport.RegisterProtocol`の拡張）**

既存の `Transport.RegisterProtocol` メソッドをスキーム `"http/3"` で呼び出すことで、HTTP/3の `RoundTripper` を登録します。内部的には `dialClientConner` インターフェースを実装している必要があります（非エクスポート）。

**3. サーバー側の登録（`Server.TLSNextProto`の拡張）**

既存の `Server.TLSNextProto` マップに `"http/3"` キーでハンドラ関数を設定することで、HTTP/3サーバー実装を登録します。

## これによって何ができるようになるか

HTTP/3実装を `net/http` のエコシステムに統合し、既存コードの変更を最小限にHTTP/3を利用できるようになります。

### コード例

```go
// Before: HTTP/3専用ライブラリを独立して使用する必要があり、
// net/httpの機能（コネクションプーリング、ミドルウェア等）が流用できない
h3client := http3.NewClient()
resp, err := h3client.Get("https://example.com/")

// After: 既存のnet/httpクライアントにHTTP/3を差し込む
tr := &http.Transport{}

// 外部HTTP/3実装を登録
http3.RegisterTransport(tr)

// HTTP/3を有効化
tr.Protocols = new(http.Protocols)
tr.Protocols.SetHTTP3(true)

// 通常どおりRoundTripを実行（HTTP/3が使用される）
resp, err := tr.RoundTrip(req)
```

```go
// サーバー側: HTTP/3対応サーバーの起動
srv := &http.Server{Addr: "localhost:8000"}

// 外部HTTP/3実装を登録
http3.RegisterServer(srv)

// HTTP/3の有効化
srv.Protocols = new(http.Protocols)
srv.Protocols.SetHTTP3(true)

// HTTP/3でリッスン開始
srv.ListenAndServeTLS(certFile, keyFile)
```

**主なユースケース:**

- QUICベースの低遅延通信が求められるAPIクライアント（モバイルネットワーク環境など）
- HTTP/3対応のウェブサーバーをGoで構築したい場合に、既存の `net/http` ハンドラをそのまま流用する
- 将来的に `net/http.Transport` がHTTPS DNSレコードやAlt-Svcヘッダに基づいてHTTP/1・HTTP/2・HTTP/3を自動選択するための基盤整備

## 議論のハイライト

- **エクスポートAPIは最小限に留める設計方針**: `dialClientConner` や `http3ServerHandler` などの内部インターフェースは非エクスポートとし、このAPIを使うパッケージ数は「一桁前半」と想定されるため、ユーザーへの露出を最小化しています。

- **HTTP/2の失敗を繰り返さない**: HTTP/2を `net/http` に内包したことで生じた依存関係の複雑さを教訓に、HTTP/3は外部パッケージとして維持する方針が明確化されています。HTTP/3は重量な依存であり、全ユーザーに強制すべきではないという合意があります。

- **既存メカニズムの再利用**: `Transport.RegisterProtocol`（本来はftp等のカスタムスキーム向け）と `Server.TLSNextProto`（本来はTLSハンドシェイク後のプロトコル切り替え用）を転用することで、新しいエクスポートAPIの追加を極力避けています。これは「意図されない使い方」であることを提案者自身も認めています。

- **HTTP/3選択の方針は保守的**: 現時点では `Transport.Protocols` が HTTP/3のみ（HTTP/1・HTTP/2を含まない）に設定された場合にのみHTTP/3を使用します。Alt-Svc・HTTPS DNSレコード・Happy Eyeballs的な並列接続試行などの高度な選択戦略は将来の課題とされています。

- **関連proposition `x/net/http3`（#70914）はホールド**: 本proposal（#77440）の審議と並行して、`golang.org/x/net/http3` の実験的実装proposal（#70914）は本proposalの決着を待つためホールドに置かれています。

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/77440)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4000757564)
- [関連Issue: net/http HTTP/3サポート追跡 #32204](https://github.com/golang/go/issues/32204)
- [関連Issue: x/net/http3 実験的実装 #70914](https://github.com/golang/go/issues/70914)
- [関連Issue: HTTP version selection API #67814](https://github.com/golang/go/issues/67814)
