---
issue_number: 77887
title: "doc: clarify if simple helper methods on x/sys syscall data structures require a proposal"
previous_status: 
current_status: active
changed_at: 2026-03-04T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
related_issues:
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/77887
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-4000757564
  - title: "関連Issue #66850 - x/sys/windows: Add GetAce syscall"
    url: https://github.com/golang/go/issues/66850
---
## 概要

`golang.org/x/sys` パッケージにおける syscall データ構造体への「シンプルなヘルパーメソッド」追加が、正式な Proposal プロセスを必要とするかどうかを明確化し、ドキュメントを更新することを求める提案です。

## ステータス変更

**(新規)** → **active**

Proposal Review Group の @aclements が、このドキュメント整備要求をアクティブな Proposal として週次レビュー会議の議題に追加しました。@ianlancetaylor がこの問題を「ambiguous（曖昧）」と認め、Proposal Committee による判断と将来的なルール策定を推奨したことが、正式なレビューへの移行を後押ししました。

## 技術的背景

### 現状の問題点

`golang.org/x/sys/windows` パッケージへのコントリビューション審査において、レビュアー間でルールの解釈が一致していません。

具体的なケースとして、コントリビューターの @database64128 が提出した CL 744880 では、Windows API の `MIB_IF_TABLE2` 構造体に対応する `MibIfTable2` 型に以下のヘルパーメソッドを追加しようとしていました：

```go
func GetIfTable2Ex(level uint32, table **MibIfTable2) (errcode error)

type MibIfTable2 struct {
    NumEntries uint32
    Table      [1]MibIfRow2
}

// Rows はWindows管理メモリ上のテーブルをGoのスライスとして返す
func (t *MibIfTable2) Rows() []MibIfRow2 {
    return unsafe.Slice(&t.Table[0], t.NumEntries)
}
```

この `Rows()` メソッドは、C 言語の構造体をそのまま Go に移植した場合に生じる使いづらさ（`[1]MibIfRow2` 型の固定配列から実際のエントリ数分のスライスを取得する煩雑さ）を解消するための「慣用的なラッパー」です。

しかし、レビュアーの @alexbrainman はこのメソッドを「架空の（fictitious）メソッド」として、正式な Proposal が必要だと主張して削除を要求しました。一方で、以前の CL 695195 では別のレビュアーがヘルパーメソッドをむしろ増やすよう要求していました。この矛盾が一貫性の欠如を招いています。

### 提案された解決策

Proposal は新機能の追加ではなく、**ドキュメントの明確化**を求めるものです。具体的には以下を整備することを要求しています：

1. Cヘッダファイルに直接対応するメソッド → Proposal 不要
2. Cの構造体には存在しないが利便性のために追加する「ヘルパーメソッド」→ Proposal 必要か否か
3. Unix 系と Windows 系でルールが異なる場合はその違いの明文化

## これによって何ができるようになるか

このドキュメント整備が完了することで、以下が実現します：

- コントリビューターがヘルパーメソッドを追加する際に、Proposal が必要か否かを自己判断できる
- レビュアー間の一貫したフィードバックが可能になり、不必要な手戻りが減る
- `x/sys` パッケージ全体のAPI品質と一貫性が向上する

### コード例

```go
// Before: Rows()メソッドがない場合（ユーザーが自前でunsafe操作が必要）
var table *windows.MibIfTable2
windows.GetIfTable2Ex(windows.MibIfTableRaw, &table)
rows := unsafe.Slice(&table.Table[0], table.NumEntries)
for _, row := range rows {
    // 処理
}

// After: Rows()ヘルパーメソッドがある場合（慣用的なGoコード）
var table *windows.MibIfTable2
windows.GetIfTable2Ex(windows.MibIfTableRaw, &table)
for _, row := range table.Rows() {
    // 処理
}
```

## 議論のハイライト

- **Unixとの非対称性**: @ianlancetaylor によれば、Unix 系では「Cヘッダに存在する名前・機能の複製は Proposal 不要」というルールが確立しているが、Windows 系では同様のルールが不明確であると指摘されている。
- **「架空のメソッド」問題**: `Rows()` は Windows API に対応するメソッドではなく Go 独自の便宜的メソッドであるため、@alexbrainman は「公開後に変更不可能になるリスク」を懸念して慎重な姿勢を取っている。
- **レビュアー間の矛盾**: 別の CL（CL 695195）では逆にヘルパーメソッドの追加を求められており、同一リポジトリ内でレビュー基準が統一されていないことが問題の核心である。
- **Proposal プロセスの文書**: 現行の `golang/proposal` リポジトリの README は「`x/sys` への新規 syscall ラッパーと C 相当のデータ構造追加は Proposal 不要」と記載しているが、ヘルパーメソッドについては明示的に言及していない。
- **Proposal Committee への委託**: @ianlancetaylor は「この件を Proposal Committee に判断してもらい、将来のケースへのルール設定を」と提言しており、これが今回の active 昇格につながった。

## 関連リンク

- [Proposal Issue](https://github.com/golang/go/issues/77887)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-4000757564)
- [関連Issue #66850 - x/sys/windows: Add GetAce syscall](https://github.com/golang/go/issues/66850)
