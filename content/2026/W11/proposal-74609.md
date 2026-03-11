---
issue_number: 74609
title: "runtime/pprof,runtime: new goroutine leak profile"
previous_status: 
current_status: active
changed_at: 2026-03-11T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4042167102
related_issues:
  - title: "関連Issue: goroutine leak detection (older) #18835"
    url: https://github.com/golang/go/issues/18835
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/74609
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4042167102
  - title: "関連Issue: GOEXPERIMENT mini-proposal #75280"
    url: https://github.com/golang/go/issues/75280
  - title: "関連Issue: goroutine leak detection (older) #13759"
    url: https://github.com/golang/go/issues/13759
---
## 概要

`runtime/pprof` パッケージに新しいプロファイルタイプ `goroutineleak` を追加し、ガベージコレクター（GC）の到達可能性解析を活用してゴルーチンリークを検出する機能を提案します。これにより、実行不可能な状態で永久にブロックされたゴルーチン（部分的デッドロック）をオンデマンドで特定できるようになります。

## ステータス変更

**(新規)** → **active**

2026年3月11日の週次プロポーザルレビューミーティング（@aclements, @adonovan, @cherrymui, @griesemer, @neild, @rolandshoemaker）にて議題に追加されました。`mknyszek` がユーザー可視の変更内容を詳細に整理したコメントを投稿し、`aclements` が「プロポーザルを前進させる準備ができている」と表明したことで、activeステータスに移行しました。GOEXPERIMENTフラグ（`goleakprofiler`）による実験的実装（#75280）がすでに受理済みであり、実運用での検証データも揃ったことが背景にあります。

## 技術的背景

### 現状の問題点

Goプログラムにおけるゴルーチンリークは、チャネルや`sync`パッケージのプリミティブを誤って使用した際に発生します。現在、[uber-go/goleak](https://github.com/uber-go/goleak) のようなサードパーティライブラリはユニットテストに限定されており、本番環境や大規模テストスイートでのリーク検出は困難でした。また、既存の goroutine プロファイルではリーク状態のゴルーチンを特定できません。

```go
// リークの例: 送信者がいないチャネルで永久にブロック
func leakyFunc() {
    ch := make(chan int)
    go func() {
        v := <-ch // このゴルーチンは永久にブロックされる
        fmt.Println(v)
    }()
    // chへの送信なし、関数終了
}
```

### 提案された解決策

GCのマーキングフェーズに新しい「リーク検出GCサイクル」を統合します。アルゴリズムは以下の手順で動作します。

1. 実行可能なゴルーチンのみをGCルートとして扱う（通常GCは全ゴルーチンをルートとする）
2. これらのルートから到達可能なメモリをマーキング
3. マーク済みの並行プリミティブでブロックされている未マークのゴルーチンを「最終的に実行可能」と判定し、GCルートに追加
4. 不動点に達するまで繰り返す
5. それでも未マークのゴルーチンを「リーク済み」として報告

検出されたゴルーチンはトレースバック出力で `[leaked]` として表示されます（従来は `[waiting]` や `[runnable]` と表示）。

## これによって何ができるようになるか

新しいプロファイルタイプ `goroutineleak` が `runtime/pprof` パッケージに追加され、以下の操作が可能になります。

- `runtime/pprof.Lookup("goroutineleak")` でプロファイルを取得
- `runtime/pprof.Profiles()` の返り値に `goroutineleak` が含まれる
- `net/http/pprof` パッケージに `/debug/pprof/goroutineleak` エンドポイントが自動追加される

### コード例

```go
// Before: サードパーティライブラリによるユニットテスト限定の検出
// uber-go/goleak はテスト終了時のみ使用可能
func TestSomething(t *testing.T) {
    defer goleak.VerifyNone(t)
    // テスト本体
}

// After: 本番環境・テストを問わずオンデマンドで検出
import (
    "runtime/pprof"
    "os"
)

// プロファイルを取得してリーク検出GCサイクルを実行
p := pprof.Lookup("goroutineleak")
p.WriteTo(os.Stdout, 1) // debug=1 でテキスト形式のトレースバックを出力
// リーク済みゴルーチンは [leaked] として表示される

// HTTPエンドポイント経由での利用（net/http/pprof インポートで自動有効化）
// curl http://localhost:6060/debug/pprof/goroutineleak
```

## 議論のハイライト

- **偽陽性ゼロの保証**: アルゴリズムの性質上、検出されたゴルーチンは確実に「永久にブロックされる」と判定できるため、誤検知が発生しません。これが既存ツールとの大きな差別化ポイントです。
- **メモリ回収の見送り**: 当初の実装ではリーク済みゴルーチンが参照するメモリを強制回収する設計でしたが、ネットワーク接続・ファイル・Cメモリなど非メモリリソースのリークに対処できないため、まずは「検出器」としての実装に絞ることで合意されました。
- **`GOEXPERIMENT` フラグの採用**: 実装CLがマージコンフリクトを蓄積し続けていた問題に対処するため、`goleakprofiler` という `GOEXPERIMENT` フラグで機能をゲーティングする案（#75280）が先行して受理されており、本番での実証データ収集が行われました。
- **`go test` 統合の可能性**: レースディテクタと同様に `go test` フラグとして統合する案が議論されましたが、デフォルト有効化すると既存のプログラムを壊す可能性があるため、現時点では見送られています。
- **Uberでの大規模実証**: Uber社内の3111テストスイートで180〜357件の構文的に異なるゴルーチンリークを検出し、本番サービスでは24時間で252件のリークレポートを生成した実績が提案の信頼性を裏付けています。

## 関連リンク

- [関連Issue: goroutine leak detection (older) #18835](https://github.com/golang/go/issues/18835)
- [Proposal Issue](https://github.com/golang/go/issues/74609)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4042167102)
- [関連Issue: GOEXPERIMENT mini-proposal #75280](https://github.com/golang/go/issues/75280)
- [関連Issue: goroutine leak detection (older) #13759](https://github.com/golang/go/issues/13759)
