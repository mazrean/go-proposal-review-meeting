---
issue_number: 67817
title: "x/net/http2: deprecate WriteScheduler"
previous_status: active
current_status: likely_accept
changed_at: 2026-02-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3872311559
related_issues:
  - title: "RFC 9218優先度方式の実装"
    url: https://github.com/golang/go/issues/75500
  - title: "優先度スケジューラーのストリーム飢餓問題"
    url: https://github.com/golang/go/issues/58804
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/67817
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3872311559
  - title: "親プロジェクト：HTTP/2の標準ライブラリ統合"
    url: https://github.com/golang/go/issues/67810
---
## 概要
`x/net/http2` パッケージの `WriteScheduler` インターフェースと関連機能を非推奨化する提案です。WriteSchedulerはHTTP/2ストリームへのデータ書き込み順序を制御しますが、実装が複雑で、RFC 7540で定義された優先度方式（現在は非推奨）に基づいており、デフォルトのラウンドロビンスケジューラーで十分な性能が得られることから、廃止が提案されています。

## ステータス変更
**active** → **likely_accept**

RFC 9218優先度方式（#75500）がGo 1.27で実装されたことにより、既存の優先度スケジューラーを使用していたユーザーが代替手段を持つことになりました。プロポーザルレビューグループは、この新しい実装により非推奨化を進めることが適切と判断し、最終コメント期間（likely_accept）に移行しました。

## 技術的背景

### 現状の問題点

`x/net/http2` パッケージは、HTTP/2ストリームへのデータ書き込み順序を決定するために、以下の3つのWriteScheduler実装を提供しています：

1. **ラウンドロビン**（デフォルト）：ストリーム間で公平に順番に書き込み
2. **ランダム**：ランダムにストリームを選択
3. **優先度**：RFC 7540の優先度方式に基づく

しかし、優先度スケジューラーには以下の深刻な問題があります：

- **ストリーム飢餓の発生**：同じ優先度のストリーム間で、新規リクエストが既存の長時間実行中のストリームを無限にブロックする可能性がある（#58804で報告）
- **CPU負荷が高い**：O(n)の計算量で、ストリーム数が増えるとCPU使用率が大幅に増加
- **非推奨の仕様に基づく**：RFC 9113でRFC 7540の優先度方式が非推奨化された。RFC 9113は「この優先度システムは複雑で、実装が一貫していなかった」と述べている

```go
// 問題例：優先度スケジューラーでのストリーム飢餓
// 新規リクエストが次々と作成されると、既存の長時間実行中のストリームが
// データを受信できなくなる（無限にブロックされる）
```

### 提案された解決策

以下のAPI要素を非推奨化し、1年後に削除：

- `WriteScheduler` インターフェース
- `FrameWriteRequest`
- `NewPriorityWriteScheduler`
- `NewRandomWriteScheduler`
- `OpenStreamOptions`
- `PriorityWriteSchedulerConfig`
- `Server.NewWriteScheduler`

この非推奨化は、`x/net/http2` を標準ライブラリに移行するプロジェクト（#67810）の一環です。

## これによって何ができるようになるか

**開発者への影響**：

- **ほとんどのユーザーは影響を受けません**：デフォルトのラウンドロビンスケジューラーを使用している場合、何も変更する必要はありません
- **優先度制御が必要なユーザー**：RFC 9218優先度方式（#75500でGo 1.27に実装）に移行できます。こちらはシンプルで拡張可能な優先度制御を提供します

### RFC 9218 vs RFC 7540の違い

```go
// RFC 7540（非推奨、複雑）：
// - ストリーム依存関係の複雑なツリー構造
// - 256段階の重み付け
// - 実装が複雑でCPU負荷が高い

// RFC 9218（新方式、シンプル）：
// - urgency: 0-7の8段階（0が最高優先度）
// - incremental: true/false（部分レスポンスの有効性）
// - 実装がシンプルで拡張可能
```

**性能面の改善**：

実際のベンチマーク（#67817のコメントより）では、優先度スケジューラーとラウンドロビンの比較で：
- WebPageTestで1秒以上のSpeed Index改善（特定のケース）
- LCPタイミングで2.5秒の改善（特定のケース）

ただし、これらは古いRFC 7540実装での結果であり、新しいRFC 9218実装ではより良い性能が期待されます。

## 議論のハイライト

- **ユーザーからの懸念**：@pkrammeは当初、代替手段なしでの非推奨化に強く反対。本番環境で優先度スケジューラーを使用しており、ラウンドロビンに比べてLCP、FCP、Speed Indexで数秒の性能改善を観測していた

- **RFC 9218の実装が決定打**：#75500でRFC 9218優先度方式の実装が承認され、Go 1.27で実装完了。これにより@pkrammeも「RFC 9218があれば完璧」と表明し、懸念が解消

- **段階的な移行計画**：
  1. 現時点で非推奨の警告を追加
  2. 1年後にサポートを削除
  3. 削除後、サーバーは `NewWriteScheduler` フィールドの値を無視

- **なぜ今なのか**：`x/net/http2` を標準ライブラリに統合する大規模プロジェクトの一環。複雑で保守困難なAPIを整理し、よりシンプルで保守しやすい実装にすることが目的

- **代替実装の参考**：QUICHEやgrpc-goはシンプルなラウンドロビン方式を採用しており、RFC 7540の複雑な優先度方式は使用していない

## 関連リンク

- [RFC 9218優先度方式の実装](https://github.com/golang/go/issues/75500)
- [優先度スケジューラーのストリーム飢餓問題](https://github.com/golang/go/issues/58804)
- [Proposal Issue](https://github.com/golang/go/issues/67817)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3872311559)
- [親プロジェクト：HTTP/2の標準ライブラリ統合](https://github.com/golang/go/issues/67810)
