---
issue_number: 17747
title: "cmd/vet: check for missing Err calls for bufio.Scanner and sql.Rows"
previous_status: discussions
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Proposal Issue #17747"
    url: https://github.com/golang/go/issues/17747
  - title: "Review Minutes (2026-01-28)"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
---

## 要約

## 概要
`bufio.Scanner`と`sql.Rows`を使用した際に、ループ終了後に必須となる`Err()`メソッドの呼び出しが欠落しているコードを検出する、新しい`go vet`チェッカーの提案です。これらのAPIは、エラー発生時も正常終了時もループが終わる設計であるため、エラーの見落としが非常に多く発生しています。

## ステータス変更
**discussions** → **active**

2026年1月28日のProposal Review Meetingで、本提案が再びactiveステータスとなりました。2025年12月にAlan Donovanが長期間のholdを解除し、実装が進められています。現在、`bufio.Scanner`向けのチェッカーが実装され（CL 730480）、大規模なコーパスでの評価が行われている段階です。

## 技術的背景

### 現状の問題点

`bufio.Scanner`と`sql.Rows`は、イテレーションパターンを使うAPIで、以下のような共通の設計になっています:

```go
scanner := bufio.NewScanner(file)
for scanner.Scan() {
    // scanner.Text()を処理
}
// ここでscanner.Err()のチェックが必要だが、忘れられることが多い
```

このAPIでは`Scan()`/`Next()`が`false`を返す理由が2つあります:
1. データの終端に到達した（正常終了）
2. エラーが発生した（異常終了）

しかし、ループを抜けただけではどちらなのか判別できません。そのため、`Err()`メソッドを呼んでエラーの有無を確認する必要がありますが、この呼び出しが頻繁に忘れられています。

**見落とされやすいエラー:**
- `bufio.Scanner`の場合: I/Oエラーに加えて、トークンサイズが`bufio.MaxScanTokenSize`（デフォルト64KB）を超えた際の`ErrTooLong`エラー
- `sql.Rows`の場合: データベースとの通信エラーや、カーソルの取得エラー

特に`ErrTooLong`は、`strings.Reader`のような「絶対に失敗しない」入力元を使っている場合でも、入力データが長すぎると発生する可能性があるため、すべてのケースでチェックが推奨されます。

### 提案された解決策

`go/analysis`フレームワークを使った新しいアナライザー（`scannererr`）を実装し、以下を検出します:

1. `bufio.Scanner`の`Scan()`がループで使用されている
2. ループ終了後、`scanner.Err()`の呼び出しがない
3. 同様のパターンを`sql.Rows`の`Next()`にも適用（予定）

**偽陽性の削減策:**
- `strings.Reader`や`bytes.Reader`のような「I/Oエラーが発生しない」入力元については、診断を抑制（ただし`ErrTooLong`は依然として発生しうる）
- `os/exec`パッケージを使った子プロセスのパイプ読み込みは、実装上の制約から偽陽性となりやすいことが判明（72%の真陽性率）

## これによって何ができるようになるか

この`go vet`チェッカーにより、開発者は以下のような潜在的なバグを早期に発見できます:

- ファイル読み込み中のI/Oエラーを見逃すケース
- 大きな入力データによるバッファオーバーフロー（`ErrTooLong`）の見逃し
- データベースクエリの途中でエラーが発生したが、一部のデータだけ処理して成功したと誤認するケース

### コード例

```go
// Before: エラーチェックが不足しているコード（問題あり）
func readLines(filename string) ([]string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var lines []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    // scanner.Err()をチェックしていない！
    // ファイルI/Oエラーや行が長すぎるエラーを見逃す
    return lines, nil
}

// After: 適切なエラーチェックを追加
func readLines(filename string) ([]string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var lines []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    // ループ終了後、必ずErr()をチェック
    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("scan error: %w", err)
    }
    return lines, nil
}
```

```go
// sql.Rowsの例
// Before: エラーチェックが不足
func getUsers(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id, name FROM users")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var users []User
    for rows.Next() {
        var u User
        if err := rows.Scan(&u.ID, &u.Name); err != nil {
            return nil, err
        }
        users = append(users, u)
    }
    // rows.Err()をチェックしていない！
    return users, nil
}

// After: 適切なエラーチェックを追加
func getUsers(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id, name FROM users")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var users []User
    for rows.Next() {
        var u User
        if err := rows.Scan(&u.ID, &u.Name); err != nil {
            return nil, err
        }
        users = append(users, u)
    }
    // ループ終了後、必ずErr()をチェック
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("rows error: %w", err)
    }
    return users, nil
}
```

## 議論のハイライト

- **初期の懸念（2016年）**: Josh Arianは「`strings.Reader`のような絶対に失敗しない入力元では、偽陽性となる」と指摘。しかしAustin Clementsが「`ErrTooLong`はどんな入力元でも発生しうる」と反論し、全ケースでチェックが必要との合意形成

- **真陽性率の調査（2026年1月）**: Alan Donovanが約22,000モジュールで実験を実施。`strings.Reader`/`bytes.Reader`を除外した場合でも、真陽性率は72%（25サンプル中18件）で、`go vet`の目標である95%に届かず。偽陽性の大半は`os/exec`の子プロセスパイプ読み込みだったが、これも「`ErrTooLong`は発生しうる」との見解でグレーゾーン扱いに

- **実践的な証拠（2022年）**: PleasingFungusの報告では、自社コードベースの約10%で`Err()`チェックが欠落しており、そのすべてがバグだったと報告。実務での重要性が裏付けられた

- **配布方針の議論**: Robert Griescmerは「`go test`で実行される高精度チェック」と「手動実行の`go vet`での積極的チェック」の2段階導入を提案。Alan Donovanは「goplsには問題なく追加でき、`go test`には不向き。`go vet`への追加が妥当」と応答

- **API設計の根本議論**: Axel Wagnerは「`Err()`パターンではなく`Close() error`にすべきでは」と提案したが、Bryan Millsが「`Close`は書き込みの失敗を示すのに対し、`Err`は読み込みの失敗を示す」と指摘し、現行API設計が妥当と結論

## 関連リンク

- [Proposal Issue #17747](https://github.com/golang/go/issues/17747)
- [Review Minutes (2026-01-28)](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
