---
issue_number: 77169
title: "testing/synctest: add convenience function for Sleep then Wait"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "関連提案: synctestの構文オーバーヘッド削減"
    url: https://github.com/golang/go/issues/76607
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77169
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "関連提案: testing/synctest新パッケージ"
    url: https://github.com/golang/go/issues/67434
---

## 要約

## 概要
`testing/synctest`パッケージに、仮想時計を指定時間進めた後にgoroutineの待機状態を確認する`Sleep`関数を追加する提案。この関数は`time.Sleep(d)`と`synctest.Wait()`を組み合わせた便利関数であり、テストコードで頻繁に使われるパターンを標準化します。

## ステータス変更
**likely_accept** → **accepted**

この決定は、提案委員会での議論を経て承認されました。主な理由として以下が挙げられています：

- 関数自体は極めてシンプルだが、synctestを使う**ほぼすべてのテスト**で必要とされる重要なパターンを表現している
- synctest開発者の@neildが「synctestを使うほぼすべてのテストでこの関数を書いている」と証言
- `time.Sleep()`だけを呼び出して`Wait()`を忘れるミスが頻繁に発生し、テストが競合状態に陥る問題がある
- 関数として明示的にパターンを提供することで、コードの意図が明確になり、ミスを防げる

## 技術的背景

### 現状の問題点

`testing/synctest`は並行コードのテストを支援するパッケージで、仮想時計と制御されたgoroutine実行環境（「バブル」と呼ばれる）を提供します。バブル内では以下の特徴があります：

- 時間は実際には経過せず、すべてのgoroutineがブロックされたときにのみ進む
- `time.Sleep()`は仮想時計上で動作し、実際の待機は発生しない

**現在の典型的な使い方では、2つの呼び出しが必要です**：

```go
synctest.Test(t, func(t *testing.T) {
    time.Sleep(5 * time.Second)  // 仮想時計を5秒進める
    synctest.Wait()              // すべてのgoroutineがブロックされるまで待つ
})
```

しかし、開発者は`time.Sleep()`だけを呼んで`Wait()`を忘れることが多く、これによりテストコードとテスト対象コードが同時に実行され、競合状態が発生します。

### なぜWaitが必要なのか

テストコード自体とテスト対象のシステムコードの両方が同じ時間だけSleepした場合、どちらが先に実行されるかは予測不可能です。テストコードは通常、システムコードがSleep後に「落ち着く」のを待ちたいため、`Wait()`によって他のgoroutineが完全にブロックされたことを確認する必要があります。

### 提案された解決策

以下の新しい関数を`testing/synctest`パッケージに追加：

```go
// Sleep は仮想時計をdだけ進め、
// このsynctestバブル内の他のgoroutineを実行させた後、
// バブル内のすべてのgoroutineが持続的にブロックされるまで待機します。
//
// これは以下と完全に等価です：
//
//     time.Sleep(d)
//     synctest.Wait()
//
// テストにおいて、time.Sleepだけを呼ぶよりもこちらが望ましいことが多いです。
// テスト自身とテスト対象システムを実行する別のgoroutineが
// 全く同じ時間Sleepする場合、どちらが先に実行されるかは予測不可能です。
// テスト自身は通常、Sleep後にテスト対象システムが「落ち着く」のを待ちたいため、
// Sleepがこれを実現します。
func Sleep(d time.Duration) {
    time.Sleep(d)
    Wait()
}
```

## これによって何ができるようになるか

テストコードがより簡潔になり、よくあるミスを防げます。特に、タイムアウト付きコンテキストのテストなど、時間ベースの動作を検証する場面で有用です。

### コード例

```go
// Before: 従来の書き方（2つの呼び出しが必要）
synctest.Test(t, func(t *testing.T) {
    ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
    defer cancel()

    // タイムアウト直前まで待機
    time.Sleep(5*time.Second - time.Nanosecond)
    synctest.Wait()  // 忘れがち！
    if err := ctx.Err(); err != nil {
        t.Fatalf("タイムアウト前にエラー: %v", err)
    }

    // 残りの時間を待機
    time.Sleep(time.Nanosecond)
    synctest.Wait()  // これも忘れがち！
    if err := ctx.Err(); err != context.DeadlineExceeded {
        t.Fatalf("タイムアウト後のエラーが期待と異なる: %v", err)
    }
})

// After: 新APIを使った書き方（1回の呼び出しで済む）
synctest.Test(t, func(t *testing.T) {
    ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
    defer cancel()

    // タイムアウト直前まで待機
    synctest.Sleep(5*time.Second - time.Nanosecond)
    if err := ctx.Err(); err != nil {
        t.Fatalf("タイムアウト前にエラー: %v", err)
    }

    // 残りの時間を待機
    synctest.Sleep(time.Nanosecond)
    if err := ctx.Err(); err != context.DeadlineExceeded {
        t.Fatalf("タイムアウト後のエラーが期待と異なる: %v", err)
    }
})
```

## 議論のハイライト

- **極めてシンプルだが極めて重要**: 委員会は「一方では信じられないほど些細、他方では信じられないほど些細」とコメント。機能は単純だが、頻繁に使われる重要なパターンを捉えている
- **自動化は困難**: `time.Sleep`に自動的に`Wait`を含める案も検討されたが、複数のgoroutineが同時に`Wait`を呼べない制約があり、実装が困難
- **ルートgoroutineのみ特別扱い**: ルートgoroutineだけで自動的に`Wait`する案も却下。synctestはどのgoroutineも特別扱いしない設計を維持
- **ドキュメントの明確化**: この関数には「魔法」は一切なく、単に2つの呼び出しを組み合わせただけであることをドキュメントで明示する必要がある
- **視認性の向上**: `time.Sleep`の代わりに`synctest.Sleep`を使うことで、仮想時計を使っていることがコード上で明確になり、理解しやすくなる（@apparentlymartのコメント）
- **実装済み**: 提案承認後、すぐに実装CL（go.dev/cl/740066）が提出された

## 関連リンク

- [関連提案: synctestの構文オーバーヘッド削減](https://github.com/golang/go/issues/76607)
- [Proposal Issue](https://github.com/golang/go/issues/77169)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連提案: testing/synctest新パッケージ](https://github.com/golang/go/issues/67434)
