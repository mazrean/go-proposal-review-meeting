---
issue_number: 67817
title: "x/net/http2: deprecate WriteScheduler"
previous_status: discussions
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/67817
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "proposal: x/net/http2: add support for RFC 9218 priorities · Issue #75500 · golang/go"
    url: https://github.com/golang/go/issues/75500
  - title: "ストリーム枯渇のバグ報告 #58804"
    url: https://github.com/golang/go/issues/58804
  - title: "HTTP/2の標準ライブラリ統合 #67810"
    url: https://github.com/golang/go/issues/67810
---

## 要約

## 概要
`x/net/http2`パッケージの`WriteScheduler`インターフェースとその関連API（優先度スケジューラ、ランダムスケジューラなど）を非推奨化する提案です。HTTP/2実装を標準ライブラリに統合するプロジェクト（#67810）の一環として、古く複雑で問題のあるストリーム優先度制御機構を削除し、よりシンプルな実装へ移行することを目指しています。

## ステータス変更
**discussions** → **active**

2026年1月28日のProposal Review Meetingで、本提案が再びactiveステータスに戻されました。これは、RFC 9218優先度サポート（#75500）がGo 1.27向けに実装完了したことを受けての動きです。RFC 9218という新しい優先度制御の仕組みが利用可能になったことで、古いWriteScheduler APIを非推奨化する前提条件が整いました。

## 技術的背景

### 現状の問題点

`WriteScheduler`インターフェースは、HTTP/2ストリームへのデータ書き込み順序を制御するための機構で、現在3つの実装が存在します：

1. **ラウンドロビン（デフォルト）**: 各ストリームに順番にデータを送信
2. **ランダム**: ランダムにストリームを選択
3. **プライオリティ**: RFC 7540の優先度スキームに基づく制御

このうち、プライオリティスケジューラには以下の深刻な問題があります：

- **バグによるストリーム枯渇**: 同じ優先度を持つストリーム群において、新しく作成された短命のリクエストが長時間実行中のストリームを無限に枯渇させる可能性があります（#58804で詳細に報告）
- **CPU使用率の高さ**: O(n)の計算量で、大量のストリームがあると著しくCPUを消費します
- **非推奨化されたRFC 7540優先度スキームの実装**: RFC 9113（HTTP/2の最新版）で「複雑すぎて実装が不統一であり、成功しなかった」として非推奨化されています

さらに、`WriteScheduler`インターフェース自体にも以下の制約があります：

- すべてのフレームをスケジューラを通す必要があり、HTTP/2サーバー実装を過度に制約
- RFC 7540の優先度スキームに特化した設計で、現代的なRFC 9218優先度シグナルへの対応が困難
- ストリームIDのみを受け取り、URLなどのリクエスト情報にアクセスできないため、サーバー側での優先度制御ができない

### 提案された解決策

以下のAPIを非推奨化します：

- `WriteScheduler` インターフェース
- `FrameWriteRequest`
- `OpenStreamOptions`
- `NewPriorityWriteScheduler`
- `NewRandomWriteScheduler`
- `PriorityWriteSchedulerConfig`
- `Server.NewWriteScheduler`

非推奨化後、1年後に削除予定です。削除後、サーバーは`NewWriteScheduler`フィールドの値を無視します。

## これによって何ができるようになるか

本提案自体は既存機能の削除ですが、背景にあるRFC 9218実装（#75500、Go 1.27で利用可能）により、より優れた優先度制御が可能になります：

### RFC 9218の優先度制御

- **`urgency`**: 0〜7の数値（低い値ほど高優先度、デフォルト3）
- **`incremental`**: boolean値（false = 部分的なチャンクで即座に利益を得られる、デフォルトfalse）

これにより、以下のような優先度制御が可能です：

```go
// Before: 複雑なWriteSchedulerの設定が必要
server := &http2.Server{
    NewWriteScheduler: func() http2.WriteScheduler {
        return http2.NewPriorityWriteScheduler(&http2.PriorityWriteSchedulerConfig{
            // 複雑な設定...
        })
    },
}

// After (Go 1.27): RFC 9218により自動的に適切な優先度制御
// クライアントがPriorityヘッダーで urgency と incremental を指定
// GET /critical-resource HTTP/2
// Priority: u=0, i  (urgency=0, incremental=true)

// サーバー側で優先度制御を無効化したい場合
server := &http.Server{
    // http2.Server の新しいフィールド
    DisableClientPriority: true,
}
```

### 実践的なメリット

1. **ストリーム枯渇の解決**: RFC 9218のラウンドロビン実装により、長時間実行ストリームが枯渇する問題が解消
2. **パフォーマンス向上**: WebPageTestでの実測で、ページ表示速度が大幅に改善（Speed Indexで1秒以上、LCPで2.5秒の改善事例）
3. **シンプルなAPI**: 低レベルなフレームスケジューリングではなく、HTTPリクエストレベルでの優先度制御が可能

## 議論のハイライト

- **ユーザーからの懸念（pkramme氏）**: プライオリティスケジューラを本番環境で成功裏に使用しており、削除によりエンドユーザーのパフォーマンスが大幅に低下する（LCP、FCPで数秒の悪化）という実測データを提示。代替手段なしの削除に強く反対

- **RFC 9218実装への合意**: 提案者のneild氏は、現在のWriteSchedulerが問題があることを認めつつ、RFC 9218サポート実装を条件に非推奨化を進めることに合意

- **段階的な移行**: 2025年9月にRFC 9218実装提案（#75500）が提出され、2026年1月に承認・実装完了。これにより本提案を進める前提条件が整った

- **実装完了の確認**: 2026年1月28日のミーティングで、griesemer氏がpkramme氏にRFC 9218実装完了（Go 1.27で利用可能）を確認し、本提案への懸念が残っているか確認

- **保留期間**: 2025年9月から2026年1月まで、RFC 9218実装完了を待つため一時的に「hold」ステータスとされていた

## 関連リンク
- [Proposal Issue](https://github.com/golang/go/issues/67817)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [RFC 9218優先度サポート実装 #75500](https://github.com/golang/go/issues/75500)
- [ストリーム枯渇のバグ報告 #58804](https://github.com/golang/go/issues/58804)
- [HTTP/2の標準ライブラリ統合 #67810](https://github.com/golang/go/issues/67810)
- [実装CL: SETTINGS_NO_RFC7540_PRIORITIES](https://go.dev/cl/729141)
- [実装CL: RFC 9218 incremental設定](https://go.dev/cl/729140)
- [実装CL: PRIORITY_UPDATEフレームバッファリング](https://go.dev/cl/728401)

---

**Sources:**
- [http2 package - golang.org/x/net/http2 - Go Packages](https://pkg.go.dev/golang.org/x/net/http2)
- [RFC 9218: Extensible Prioritization Scheme for HTTP](https://www.rfc-editor.org/rfc/rfc9218.html)
- [proposal: x/net/http2: add support for RFC 9218 priorities · Issue #75500 · golang/go](https://github.com/golang/go/issues/75500)

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/67817)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [proposal: x/net/http2: add support for RFC 9218 priorities · Issue #75500 · golang/go](https://github.com/golang/go/issues/75500)
- [ストリーム枯渇のバグ報告 #58804](https://github.com/golang/go/issues/58804)
- [HTTP/2の標準ライブラリ統合 #67810](https://github.com/golang/go/issues/67810)
