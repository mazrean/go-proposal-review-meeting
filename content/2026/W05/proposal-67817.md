---
issue_number: 67817
title: "x/net/http2: deprecate WriteScheduler"
previous_status: discussions
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "proposal: x/net/http2: add support for RFC 9218 priorities · Issue #75500"
    url: https://github.com/golang/go/issues/75500
  - title: "net/http: Stream starvation in http2 priority write scheduler · Issue #58804"
    url: https://github.com/golang/go/issues/58804
  - title: "関連 Issue #67810 - HTTP/2 を標準ライブラリに移行"
    url: https://github.com/golang/go/issues/67810
  - title: "Proposal Issue #67817"
    url: https://github.com/golang/go/issues/67817
---

## 要約

## 概要
`x/net/http2.WriteScheduler`は、HTTP/2ストリームへのデータ書き込み順序を制御するインターフェースです。本proposalは、このAPIを非推奨（deprecated）にすることを提案しています。理由は、現在の優先度ベーススケジューラーがバグを含み、実装している RFC 7540 の優先度機能が RFC 9113 で非推奨となったためです。

## ステータス変更
**discussions** → **active**

この変更は、2026年1月28日の Proposal Review Meeting で行われました。関連する #75500（RFC 9218 優先度サポートの追加）が Go 1.27 で実装完了したことを受け、本proposalが再び活発な議論対象となりました。グループメンバーの @griesemer が、RFC 9218 実装が完了した今、このproposalを進めることに問題がないか最終確認を行っています。

## 技術的背景

### 現状の問題点
`x/net/http2`パッケージは現在3つのWriteScheduler実装を提供しています：

1. **Round Robin（デフォルト）**: 全ストリームに均等に書き込みを分配
2. **Random**: ランダムな順序で書き込み
3. **Priority**: RFC 7540 の優先度スキームを実装（問題あり）

**Priority Write Scheduler の問題**:
- **ストリームの飢餓（starvation）問題**: 全ストリームの重みが同じ場合、新しいリクエストが次々と来ると、長期実行中のストリームが完全にブロックされる（#58804 で報告）
- **CPU負荷が高い**: O(n) の計算量で、多数のストリームがある場合にCPU消費が大きい
- **非推奨スキームの実装**: RFC 7540 のストリーム優先度機能は RFC 9113（2022年6月公開）で正式に非推奨化された。理由は複雑すぎて実装が一貫せず、実際にはあまり効果がなかったため

### 提案された解決策
以下の型・関数を非推奨とし、1年後にサポートを削除する：

- `FrameWriteRequest`
- `NewPriorityWriteScheduler`
- `NewRandomWriteScheduler`
- `OpenStreamOptions`
- `PriorityWriteSchedulerConfig`
- `Server.NewWriteScheduler`
- `WriteScheduler`

削除後、サーバーは `NewWriteScheduler` フィールドの値を無視し、常に適切なデフォルトスケジューラーを使用します。

## これによって何ができるようになるか

**ユーザー視点**:
- **シンプル化**: HTTP/2 の低レベル設定を意識する必要がなくなる
- **より良いデフォルト**: バグのある Priority スケジューラーを誤って使用するリスクがなくなる
- **将来的な改善の余地**: より良いスケジューラーが開発されれば、自動的にデフォルトとして適用される

**開発者視点**:
- **実装の制約緩和**: WriteScheduler インターフェースがすべてのフレームをスケジューラーに通す必要があったため、HTTP/2 サーバー実装が制約されていたが、これが解消される
- **メンテナンス負荷軽減**: バグのあるコードを維持する必要がなくなる

## 議論のハイライト

### 主要な反対意見（@pkramme氏）
- **実測データに基づく性能差**: webpagetest.com でのベンチマークにより、Priority スケジューラーと Round Robin では LCP（Largest Contentful Paint）で2.5秒、Speed Index で1秒以上の差が出ることを実証
- **本番環境での実績**: 自社で Priority スケジューラーを本番利用しており、問題なく動作。削除されると顧客のパフォーマンスが大幅に低下する
- **代替手段の不足**: 削除するなら、先に RFC 9218 のサポートを追加すべき

### 提案者の反論（@neild氏）
- **現実装は改善不可**: Priority スケジューラーはバグがあり、CPU負荷も高く、非推奨化されたスキームを実装している。修正する計画はない
- **インターフェースの問題**: `WriteScheduler` は複雑すぎて、フレームレベルでのスケジューリングはHTTPサーバーの実装を過度に制約する
- **より良い代替案**: 将来的に優先度機能をサポートするなら、リクエストレベルで優先度を指定できる、よりシンプルなインターフェース（例: ResponseController）が望ましい

### 解決への道筋
- **RFC 9218 サポートの実装**: #75500 で RFC 9218（Extensible Prioritization Scheme for HTTP）のサポートが提案され、Go 1.27 で実装完了
- **移行期間の確保**: RFC 9218 が利用可能になったことで、Priority スケジューラーのユーザーは移行可能に
- **段階的アプローチ**: まず RFC 9218 を提供し、その後 WriteScheduler API を非推奨化するという順序で進行

### RFC 9218 の特徴
- **urgency**: 0〜7の数値（0が最高優先度、デフォルト3）
- **incremental**: 部分的レスポンスで利益を得られるかを示すブール値（デフォルトfalse）
- よりシンプルで、HTTP/2 と HTTP/3 の両方で使える統一的な優先度スキーム

## 関連リンク

- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [proposal: x/net/http2: add support for RFC 9218 priorities · Issue #75500](https://github.com/golang/go/issues/75500)
- [net/http: Stream starvation in http2 priority write scheduler · Issue #58804](https://github.com/golang/go/issues/58804)
- [関連 Issue #67810 - HTTP/2 を標準ライブラリに移行](https://github.com/golang/go/issues/67810)
- [Proposal Issue #67817](https://github.com/golang/go/issues/67817)
