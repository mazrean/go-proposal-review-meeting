---
issue_number: 77075
title: "os/exec: add Cmd.Clone method"
previous_status: 
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Related Issue #76746 - Cmd.Start should fail when called more than once"
    url: https://github.com/golang/go/issues/76746
  - title: "Issue #46699 - proposal: os/exec: allow setting Context in Cmd"
    url: https://github.com/golang/go/issues/46699
  - title: "net/http: clarify use-cases of WithContext vs Clone on requests · Issue #53413"
    url: https://github.com/golang/go/issues/53413
  - title: "Proposal Issue #77075"
    url: https://github.com/golang/go/issues/77075
---
## 概要
`os/exec.Cmd`に対して`Clone`メソッドを追加する提案です。このメソッドは、`Cmd`の設定をすべてコピーした新しい`Cmd`インスタンスを返すもので、`Start`を複数回呼び出す際の問題を安全に解決します。

## ステータス変更
**(なし)** → **active**

2026年1月28日に、Go proposal review groupによって「active」ステータスに移行されました。これは、週次のproposal review meetingsで正式にレビューされることを意味します。

## 技術的背景

### 現状の問題点
Go 1.26で導入されたCL 728642により、`os/exec.Cmd`は誤用を検出するための防御的な挙動が強化されました。具体的には、`Start`の呼び出し失敗後に同じ`Cmd`インスタンスで再度`Start`を呼び出すことを明示的にエラーとして扱うようになりました（issue #76746）。

この変更自体はAPIの意図通りの挙動ですが、Google社内のコードで多数の潜在的な誤用が動的エラーとして顕在化しました。従来、`Cmd`は以下のような理由で再利用できません：

- `Run`、`Output`、`CombinedOutput`を一度呼び出すと再利用不可
- 内部状態（プロセス情報、パイプ、goroutineなど）が残存する
- 公式ドキュメントで明示的に「再利用不可」と記載

多くのケースでは、新しい`Cmd`を作成することでこの問題を回避できますが、以下の課題があります：

```go
// 現状のワークアラウンド: 全フィールドをコピー
cmd = &exec.Cmd{Path: cmd.Path, Args: cmd.Args, Env: cmd.Env, Dir: cmd.Dir, ...}
```

**この方法の問題点：**

1. **脆弱性**: 将来`Cmd`に新しいエクスポートフィールドが追加されても、コピーコードは更新されない可能性が高く、潜在的なバグを生む
2. **非公開フィールドの存在**: `ctx`フィールド（`CommandContext`でのみ設定）はエクスポートされておらず、どのメソッドからもアクセスできないため、完全なコピーを書くことが不可能
3. **大規模な書き換えが必要**: 場合によってはアプリケーション全体の構造を変更しなければならない

### 提案された解決策
`Cmd`に`Clone`メソッドを追加し、設定をすべてコピーした新しいインスタンスを安全に生成できるようにします。

```go
package exec // "os/exec"

// Clone returns a new Cmd that is a copy of the old one with respect to
// its configuration, and on which Start has never been called.
func (*Cmd) Clone() *Cmd
```

## これによって何ができるようになるか

`Start`呼び出しが失敗した際や、同じコマンド設定を複数回実行したい場合に、安全かつ簡潔に`Cmd`を複製できます。特に以下のようなシナリオで有用です：

1. **リトライロジック**: コマンド実行失敗時の再試行
2. **並行実行**: 同じ設定の複数プロセスを並行して起動
3. **テスト**: 同一の設定を持つコマンドを繰り返し実行

### コード例

```go
// Before: 従来の書き方（脆弱で不完全）
cmd := exec.CommandContext(ctx, "myprogram", "--flag")
cmd.Dir = "/tmp"
cmd.Env = []string{"VAR=value"}

if err := cmd.Start(); err != nil {
    // エラー時に再試行したい場合、手動で全フィールドをコピー
    cmd2 := &exec.Cmd{
        Path: cmd.Path,
        Args: cmd.Args,
        Dir: cmd.Dir,
        Env: cmd.Env,
        // ... 他の全フィールド（漏れがちで、ctxはコピー不可能）
    }
    err = cmd2.Start()
}

// After: Clone()を使った書き方
cmd := exec.CommandContext(ctx, "myprogram", "--flag")
cmd.Dir = "/tmp"
cmd.Env = []string{"VAR=value"}

if err := cmd.Start(); err != nil {
    // Clone()で安全に複製（ctx含む全設定が正確にコピーされる）
    cmd2 := cmd.Clone()
    err = cmd2.Start()
}
```

## 議論のハイライト

- **リリースブロッカー議論**: 当初、CL 728642（`startCalled`フィールドに`atomic.Bool`を使用）がcopylocksの静的解析警告を引き起こすため、Go 1.26のリリースブロッカーとされました。最終的にはCL 734200で通常の`bool`に変更され、静的解析の警告を回避しつつ、`Clone`の検討は別途進められることになりました

- **深いコピーか浅いコピーか**: `SysProcAttr`（ポインタ）やスライスフィールドをどう扱うかについて議論がありましたが、`net/http.Request.Clone`を参考に設計する方向性が示されました。`Request.Clone`は新しいcontextを受け取り、深いコピーを作成します

- **Context引数の是非**: `net/http.Request.Clone(ctx)`のように新しいcontextを引数で受け取るべきか議論がありました。一部のユーザーは「Contextをコピーすると不正確な場合がある」と指摘しましたが、提案者は「すべてをコピーする（Clone）か何もコピーしない（`&Command{...}`）かの中間を目指すべきでない」と回答しています

- **関連する古い問題**: コメント欄では、`LookPath`がchrootやカスタムPATH環境変数を考慮しない問題（#39341、#53996、#73910）など、`os/exec`の既存の制約について言及されましたが、提案者はこれらは別問題であり、`Clone`はそれらの解決を妨げも促進もしないと明言しています

- **Google社内での実データ**: Google内部のテストでは、「`os/exec.Cmd`を複数回起動できることに依存しているユーザーは発見されなかった」ことが報告されており、厳格化の影響は限定的であることが示唆されています

## 関連リンク

- [Related Issue #76746 - Cmd.Start should fail when called more than once](https://github.com/golang/go/issues/76746)
- [Issue #46699 - proposal: os/exec: allow setting Context in Cmd](https://github.com/golang/go/issues/46699)
- [net/http: clarify use-cases of WithContext vs Clone on requests · Issue #53413](https://github.com/golang/go/issues/53413)
- [Proposal Issue #77075](https://github.com/golang/go/issues/77075)
