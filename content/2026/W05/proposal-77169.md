---
issue_number: 77169
title: "testing/synctest: add convenience function for Sleep then Wait"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "関連Issue: synctestのシンタックス改善提案"
    url: https://github.com/golang/go/issues/76607
  - title: "Wait/race detector改善 #74352"
    url: https://github.com/golang/go/issues/74352
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77169
  - title: "Review Comment"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/77169#issuecomment-3814233803
  - title: "元のsynctestプロポーザル #67434"
    url: https://github.com/golang/go/issues/67434
---

## 要約

## 概要
`testing/synctest`パッケージに`synctest.Sleep(d time.Duration)`関数を追加する提案。この関数は`time.Sleep(d)`と`synctest.Wait()`を連続して呼び出すだけのシンプルなヘルパーですが、synctestを使ったテストで非常によく使われるパターンをカプセル化し、`time.Sleep`だけを呼んで`Wait`を忘れる頻出ミスを防ぎます。

## ステータス変更
**likely_accept** → **accepted**

Proposal Review Committee（2026年1月28日）で承認されました。議論では当初「あまりに単純な関数なので不要では」という意見もありましたが、最終的に「この関数が捉えるパターンは非常に重要で、テストコードと被テストコードの競合を防ぐ上で不可欠」との結論に至りました。提案者のneild氏は「synctestを使うほぼ全てのテストでこの関数を書いてきた」と報告しています。

## 技術的背景

### 現状の問題点
`testing/synctest`は並行コードのテストを支援するパッケージで、仮想時計（fake clock）と「durably blocked」の概念を使って決定的なテストを可能にします。このパッケージを使ったテストでは、以下のパターンが非常に頻繁に必要になります:

```go
time.Sleep(5 * time.Second)  // 仮想時計を進める
synctest.Wait()              // 全goroutineが安定してブロックするまで待つ
```

しかし、`time.Sleep`だけを呼んで`synctest.Wait()`を忘れると、テストコードと被テストコードの間で競合状態（race）が発生します。例えば、両方が同じ時間sleepすると、どちらが先に起きるかは予測不可能です。テストコードは通常、被テストコードが「落ち着く」まで待ちたいため、`Wait()`の呼び出しが必須です。

### synctestの「durably blocked」とは
goroutineが「durably blocked」とは、**同じbubble内の他のgoroutineによってのみブロック解除できる状態でブロックされている**ことを意味します。`time.Sleep`、bubble内のチャネル操作、`sync.Cond.Wait`などが該当します。`synctest.Wait()`は、現在のgoroutine以外の全goroutineがdurably blockedになるまで待機します。

### 提案された解決策
以下のシンプルな関数を`testing/synctest`パッケージに追加します:

```go
// Sleep advances the virtual clock by d,
// allowing other goroutines in this synctest bubble to run,
// and then waits until all goroutines in this bubble
// are durably blocked.
//
// This is exactly equivalent to
//
//     time.Sleep(d)
//     synctest.Wait()
//
// In tests, this is often preferable to calling only [time.Sleep].
// If the test itself and another goroutine running the system under test
// sleeps for the exact same amount of time, it's unpredictable which
// of the two goroutines will run first. The test itself usually wants
// to wait for the system under test to "settle" after sleeping.
// This is what Sleep accomplishes.
func Sleep(d time.Duration) {
    time.Sleep(d)
    Wait()
}
```

## これによって何ができるようになるか

テストコードがより安全で読みやすくなります。特に`context.WithTimeout`などの時間ベースの機能をテストする際、競合状態を防ぎながらシンプルに書けます。

### コード例

```go
// Before: 従来の書き方（Waitを忘れると競合状態が発生）
func TestContextWithTimeout(t *testing.T) {
    synctest.Test(t, func(t *testing.T) {
        ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
        defer cancel()

        time.Sleep(5*time.Second - time.Nanosecond)
        synctest.Wait()  // これを忘れるとrace!
        if err := ctx.Err(); err != nil {
            t.Fatal("timeout前にエラー")
        }

        time.Sleep(time.Nanosecond)
        synctest.Wait()  // これも忘れるとrace!
        if err := ctx.Err(); err != context.DeadlineExceeded {
            t.Fatal("timeoutが発生しなかった")
        }
    })
}

// After: 新APIを使った書き方（より安全で明確）
func TestContextWithTimeout(t *testing.T) {
    synctest.Test(t, func(t *testing.T) {
        ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
        defer cancel()

        synctest.Sleep(5*time.Second - time.Nanosecond)  // 1行で完結、忘れにくい
        if err := ctx.Err(); err != nil {
            t.Fatal("timeout前にエラー")
        }

        synctest.Sleep(time.Nanosecond)
        if err := ctx.Err(); err != context.DeadlineExceeded {
            t.Fatal("timeoutが発生しなかった")
        }
    })
}
```

さらに、`synctest.Sleep`という名前を見るだけで「これはsynctest特有の仮想時計を進める操作だ」とコードの意図が明確になる効果もあります。

## 議論のハイライト

- **「あまりに単純すぎる」vs「だからこそ必要」**: 当初この関数は元のsynctestプロポーザルから「あまりに単純」という理由で除外されましたが、実際に使ってみると「ほぼ全てのテストで書いている」（neild氏）ことが判明
- **自動Wait化の検討と却下**: 「rootのgoroutineだけ自動的にSleep後にWaitできないか」という提案が出ましたが、「synctestではどのgoroutineも特別扱いしない」という設計方針、および「テストコードと被テストコードを区別できない」という技術的理由で却下されました
- **パターンの可視化**: この関数は技術的には単純ですが、「重要で一般的なパターンを捉える」ことで、テストコードの意図を明確にし、バグを防ぐ効果があると評価されました
- **ドキュメントの重要性**: この関数に「マジックは一切ない」ことをドキュメントで明示することが重要と指摘されました（他のsynctest機能と異なり、単なるショートカット関数）
- **関連パターン**: `synctest.Test`のネスト問題を解決する`synctestSubtest`のような他の便利関数も議論されましたが、まずは`Sleep`から始める方針となりました

## 関連リンク
- [Proposal Issue](https://github.com/golang/go/issues/77169)
- [実装CL](https://go.dev/cl/740066)
- [testing/synctest公式ドキュメント](https://pkg.go.dev/testing/synctest)
- [synctest紹介ブログ](https://go.dev/blog/synctest)
- [元のsynctestプロポーザル #67434](https://github.com/golang/go/issues/67434)
- [Wait/race detector改善 #74352](https://github.com/golang/go/issues/74352)

---

**Sources:**
- [synctest package - testing/synctest - Go Packages](https://pkg.go.dev/testing/synctest)
- [Testing concurrent code with testing/synctest - The Go Programming Language](https://go.dev/blog/synctest)
- [Go: The Testing/Synctest Package Explained | HackerNoon](https://hackernoon.com/go-the-testingsynctest-package-explained)
- [go/src/testing/synctest/synctest.go at master · golang/go](https://github.com/golang/go/blob/master/src/testing/synctest/synctest.go)

## 関連リンク

- [関連Issue: synctestのシンタックス改善提案](https://github.com/golang/go/issues/76607)
- [Wait/race detector改善 #74352](https://github.com/golang/go/issues/74352)
- [Proposal Issue](https://github.com/golang/go/issues/77169)
- [Review Comment](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [Review Minutes](https://github.com/golang/go/issues/77169#issuecomment-3814233803)
- [元のsynctestプロポーザル #67434](https://github.com/golang/go/issues/67434)
