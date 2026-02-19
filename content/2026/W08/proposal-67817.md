---
issue_number: 67817
title: "x/net/http2: deprecate WriteScheduler"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-02-18T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3923200976
related_issues:
  - title: "関連Issue: x/net/http2をstdへ移行 (#67810)"
    url: https://github.com/golang/go/issues/67810
  - title: "関連Issue: 優先度スケジューラーのバグ報告 (#58804)"
    url: https://github.com/golang/go/issues/58804
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/67817
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3923200976
  - title: "関連Issue: RFC 9218優先度実装 (#75500)"
    url: https://github.com/golang/go/issues/75500
---
## 概要
`x/net/http2` パッケージに存在するカスタムHTTP/2書き込みスケジューラー選択機能（`WriteScheduler` インターフェース等）を非推奨（deprecated）にするproposalです。既存のスケジューラーはバグが多く、廃止済みのRFC 7540優先度スキームに基づいているため、RFC 9218への移行が完了したことを受けて正式に受理されました。

## ステータス変更
**likely_accept** → **accepted**

RFC 9218優先度スキームの実装（Issue #75500）がGo 1.27向けに完了したことで、主な反対意見（代替手段なしでの削除への懸念）が解消されました。代替策が提供された後、コンセンサスに変更がなかったため、2026年2月18日に@aclementsが正式にacceptedと宣言しました。

## 技術的背景

### 現状の問題点
`x/net/http2` パッケージは、HTTP/2ストリームへのデータ書き込み順序を制御する `WriteScheduler` インターフェースと、以下の3つの実装を提供していました。

- **ラウンドロビン**（デフォルト）: 全ストリームに均等にデータを配分
- **ランダム**: ストリームをランダム順に処理
- **優先度**（`NewPriorityWriteScheduler`）: RFC 7540の優先度ツリーを実装

問題点は以下の通りです。

1. 優先度スケジューラーにはバグがあり（Issue #58804でストリーム飢餓が報告）、CPU消費が高い
2. 優先度スケジューラーが実装するRFC 7540の優先度スキームはRFC 9113により廃止済み
3. `WriteScheduler` インターフェースはHTTPサーバー実装を過度に制約する（全フレームをスケジューラー経由で渡す必要がある）
4. ユーザー定義スケジューラーの利用者はほとんど存在せず、現代のRFC 9218優先度シグナルをサポートするためには根本的な再設計が必要

```go
// 現在：ユーザーがカスタムスケジューラーを設定できる
server := &http2.Server{
    NewWriteScheduler: func() http2.WriteScheduler {
        return http2.NewPriorityWriteScheduler(nil) // バグあり、廃止予定
    },
}

// または独自実装
type myScheduler struct{}
func (s *myScheduler) OpenStream(streamID uint32, options http2.OpenStreamOptions) {}
func (s *myScheduler) CloseStream(streamID uint32) {}
func (s *myScheduler) AdjustStream(streamID uint32, priority http2.PriorityParam) {}
func (s *myScheduler) Push(wr http2.FrameWriteRequest) {}
func (s *myScheduler) Pop() (wr http2.FrameWriteRequest, ok bool) {}
```

### 提案された解決策
以下のAPI群を非推奨とし、1年後に削除（その後は `NewWriteScheduler` フィールドが無視される）することを提案しています。

- `WriteScheduler`（インターフェース）
- `FrameWriteRequest`（構造体）
- `OpenStreamOptions`（構造体）
- `PriorityWriteSchedulerConfig`（構造体）
- `NewPriorityWriteScheduler`（コンストラクタ）
- `NewRandomWriteScheduler`（コンストラクタ）
- `Server.NewWriteScheduler`（設定フィールド）

代替として、RFC 9218に基づく新しい優先度スケジューラー（Issue #75500）が実装されました。クライアントが `priority` ヘッダーまたは `PRIORITY_UPDATE` フレームで優先度を指定することで、サーバー側が適切に処理します。

## これによって何ができるようになるか

`WriteScheduler` APIの非推奨化により、`x/net/http2` を `net/http` の標準実装として統合するプロジェクト（Issue #67810）が前進します。

具体的な影響は以下の通りです。

1. **HTTP/2サーバー実装の簡素化**: `WriteScheduler` インターフェースの制約がなくなり、サーバー実装の内部最適化が可能になる
2. **RFC 9218への移行**: クライアントが `priority` ヘッダー（`urgency` 0〜7、`incremental` フラグ）で優先度を指定でき、サーバーがそれを尊重する
3. **安全なデフォルト動作**: バグのある優先度スケジューラーへの依存が排除される

### コード例

```go
// Before: カスタムWriteSchedulerを使った優先度制御（廃止予定）
s := &http.Server{
    Handler: myHandler,
}
http2.ConfigureServer(s, &http2.Server{
    NewWriteScheduler: http2.NewPriorityWriteScheduler,
})

// After: RFC 9218によるクライアント主導の優先度指定（HTTPヘッダーで指定）
// クライアント側: リクエストヘッダーに優先度を付与
req, _ := http.NewRequest("GET", "https://example.com/critical.css", nil)
req.Header.Set("Priority", "u=1")  // urgency=1（高優先度）

// 画像など後回しでよいリソース
req2, _ := http.NewRequest("GET", "https://example.com/image.jpg", nil)
req2.Header.Set("Priority", "u=5, i")  // urgency=5、incremental=true

// サーバー側: 設定不要。RFC 9218スケジューラーが自動的にクライアント指定の優先度を尊重
server := &http.Server{
    // DisableClientPriority: true  // 必要に応じてラウンドロビンに戻せる
}
```

## 議論のハイライト

- **反対意見とその解決**: `@pkramme` 氏が実際のウェブページ測定データ（WebPageTest）を示し、優先度スケジューラーがラウンドロビンより最大2.5秒LCP改善、300msのSpeed Index改善をもたらすことを実証。代替手段なしでの削除に強く反対した
- **代替策の確約**: `@aclements` がRFC 9218優先度実装（Issue #75500）の評価中はproposalを保留とし、代替手段の提供を優先した
- **実装完了で懸念解消**: RFC 9218スケジューラーがGo 1.27向けに実装されたことを `@griesemer` が確認し、`@pkramme` 氏も「問題なし」と承認
- **設計上の根本的な問題**: `@neild` 氏が `WriteScheduler` の構造的問題（フレーム単位の制御がサーバー実装を過度に制約、RFC 7540ベースの設計でRFC 9218に不適合、ストリームURLを受け取れないためサーバー主導の優先度付けが不可能）を詳細に説明
- **一時保留（Hold）からの再開**: 2025年9月にHold状態に移行し、RFC 9218実装の進捗を待って2026年1月に審査を再開。RFC 9218実装の完了確認後、2週間でlikely_acceptからacceptedへ移行した

## 関連リンク

- [関連Issue: x/net/http2をstdへ移行 (#67810)](https://github.com/golang/go/issues/67810)
- [関連Issue: 優先度スケジューラーのバグ報告 (#58804)](https://github.com/golang/go/issues/58804)
- [Proposal Issue](https://github.com/golang/go/issues/67817)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3923200976)
- [関連Issue: RFC 9218優先度実装 (#75500)](https://github.com/golang/go/issues/75500)
