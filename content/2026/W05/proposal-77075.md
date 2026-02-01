---
issue_number: 77075
title: "os/exec: add Cmd.Clone method"
previous_status: discussions
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Proposal Issue #77075"
    url: https://github.com/golang/go/issues/77075
  - title: "Review Minutes (2026-01-28)"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "関連Issue #76746: Cmd.Start should fail when called more than once"
    url: https://github.com/golang/go/issues/76746
  - title: "関連Proposal #73450: net/url.Clone"
    url: https://github.com/golang/go/issues/73450
  - title: "net/http.Request.Clone() does not deep copy Body"
    url: https://github.com/golang/go/issues/36095
---

## 要約

## 概要
`os/exec.Cmd`に`Clone`メソッドを追加し、設定を保持したまま新しいCmdインスタンスを安全に複製できるようにする提案です。Go 1.26での動作変更により、Cmdの再利用が厳格に制限されたことで、実践的な代替手段が必要になりました。

## ステータス変更
**(新規)** → **active**

2026年1月28日のProposal Review Meetingで、この提案は**active**ステータスに移行されました。これは「議論中のproposal」として正式に検討段階に入ったことを意味します。Goチームが週次レビューで積極的に議論を進めています。

## 技術的背景

### 現状の問題点

`os/exec.Cmd`は、一度`Start`、`Run`、`Output`、`CombinedOutput`のいずれかを呼び出すと、再利用できません。この制約は元々ドキュメントに記載されていましたが、Go 1.26で[CL 728642](https://go.dev/cl/728642)により**動的チェック**が追加され、厳格に適用されるようになりました。

具体的には、`Cmd`構造体に`atomic.Bool`型の`startCalled`フィールドが追加され、`Start`が2回呼ばれると即座にエラーを返すようになりました。これには2つの影響がありました:

1. **動的エラー**: 最初の`Start`呼び出しが失敗した場合でも、2回目の呼び出しはエラーになる
2. **静的チェック**: `atomic.Bool`の存在により、`go vet`のcopylockチェッカーが「`Cmd`をコピーしている」という警告を出すようになった

この変更により、Google内部のコードで多数の潜在的な誤用が検出されました（#76746参照）。多くのケースでは、最初の`Start`が失敗した後にリトライするため、Cmdを再利用しようとしていました。

**現在のワークアラウンド:**
```go
// すべてのフィールドを手動でコピー
cmd = &exec.Cmd{Path: cmd.Path, Args: cmd.Args, Env: cmd.Env, ...}
```

**この方法の問題点:**
- 新しいフィールドが追加されると更新が必要（忘れやすい）
- `ctx`フィールドは非公開のため、`CommandContext`で設定されたContextをコピーできない
- エラーが発生しやすく、保守性が低い

### 提案された解決策

```go
package exec // "os/exec"

// Clone returns a new Cmd that is a copy of the old one with respect to
// its configuration, and on which Start has never been called.
func (*Cmd) Clone() *Cmd
```

`Clone`メソッドは:
- すべての設定フィールド（公開・非公開含む）をコピー
- 非公開の`ctx`フィールドも正しくコピー
- 新しいCmdは`Start`が未呼び出しの状態

## これによって何ができるようになるか

開発者は、複雑なコマンド設定を一度だけ構築し、それを複数回安全に再利用できるようになります。特に以下のようなケースで有用です:

1. **リトライロジック**: 最初の実行が失敗した場合の再試行
2. **テンプレート的な使用**: 同じ設定で異なる引数を実行
3. **並行実行**: 同じコマンドを複数のゴルーチンで実行

### コード例

```go
// Before: 手動コピー（脆弱で不完全）
originalCmd := exec.CommandContext(ctx, "myapp", "--verbose")
originalCmd.Env = append(os.Environ(), "DEBUG=1")
originalCmd.Stdout = &stdout

// すべてのフィールドを手動でコピー（ctxフィールドはコピー不可能）
cmd := &exec.Cmd{
    Path: originalCmd.Path,
    Args: originalCmd.Args,
    Env: originalCmd.Env,
    Stdout: originalCmd.Stdout,
    // ... 他のフィールドも必要
    // ctx は非公開のためコピーできない！
}

// After: Cloneメソッドを使用（安全で完全）
originalCmd := exec.CommandContext(ctx, "myapp", "--verbose")
originalCmd.Env = append(os.Environ(), "DEBUG=1")
originalCmd.Stdout = &stdout

// ワンライナーで完全なコピー
cmd := originalCmd.Clone()

// 失敗時のリトライ例
for retries := 0; retries < 3; retries++ {
    cmd := originalCmd.Clone()
    if err := cmd.Run(); err == nil {
        break
    }
    time.Sleep(time.Second)
}
```

## 議論のハイライト

- **Go 1.26のリリースブロッカー問題**: 当初、atomic.Boolフィールドの追加がcopylockチェッカーを発火させ、既存コードで多数の警告が出る問題が懸念されました。チームは[CL 734200](https://go.dev/cl/734200)で`atomic.Bool`を通常の`bool`に変更し、静的チェックの問題を回避することで合意しました

- **深いコピーvs浅いコピー**: `SysProcAttr`はポインタ、他のフィールドはスライスですが、実際にはほとんどのケースで作成後に変更されないため、浅いコピーで十分とされています（#77075のprattmicコメント）

- **Contextのコピー**: `net/http.Request.Clone`と同様に、Cloneはcontextも含めてコピーします。一部のユーザーは「contextを除外したい」という要望もありましたが、提案者は「Cloneはすべてをコピーする。部分的なコピーが必要なら、Cloneの上に独自ロジックを構築すべき」と明言しました

- **類似APIの存在**: `net/http.Request.Clone(ctx)`は深いコピーを行い、新しいcontextを受け取ります。`net/url.Clone`も同時期に議論されており（#73450）、標準ライブラリ全体でCloneパターンが広がっています

- **Google内部での影響**: Googleの内部テストでは、`Start`を複数回呼び出すことに依存しているコードは発見されませんでした。問題は主に「最初の呼び出しが失敗した後の再試行」パターンでした

## 関連リンク
- [Proposal Issue #77075](https://github.com/golang/go/issues/77075)
- [Review Minutes (2026-01-28)](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連Issue #76746: Cmd.Start should fail when called more than once](https://github.com/golang/go/issues/76746)
- [実装CL 728642: 2回目のStartを常にエラーにする変更](https://go.dev/cl/728642)
- [実装CL 734200: atomic.Boolを通常のboolに戻す変更](https://go.dev/cl/734200)
- [関連Proposal #73450: net/url.Clone](https://github.com/golang/go/issues/73450)

---

**Sources:**
- [Go Release Dashboard](https://dev.golang.org/release)
- [Go os/exec Documentation](https://pkg.go.dev/os/exec)
- [net/http.Request.Clone() does not deep copy Body](https://github.com/golang/go/issues/36095)
- [inanzzz | Cloning HTTP request context](https://www.inanzzz.com/index.php/post/o75n/cloning-http-request-context-without-cancel-and-deadline-in-golang)
- [net/http.Request.Clone Documentation](https://pkg.go.dev/net/http#Request.Clone)

## 関連リンク

- [Proposal Issue #77075](https://github.com/golang/go/issues/77075)
- [Review Minutes (2026-01-28)](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連Issue #76746: Cmd.Start should fail when called more than once](https://github.com/golang/go/issues/76746)
- [関連Proposal #73450: net/url.Clone](https://github.com/golang/go/issues/73450)
- [net/http.Request.Clone() does not deep copy Body](https://github.com/golang/go/issues/36095)
