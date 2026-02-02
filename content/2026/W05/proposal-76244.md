---
issue_number: 76244
title: "all: transition ppc64/linux (big-endian) from ELFv1 to ELFv2 in Go 1.27"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "関連Issue #34850: ppc64 BE port削除提案（クローズ済み）"
    url: https://github.com/golang/go/issues/34850
  - title: "all: require POWER8 support for ppc64, to match ppc64le · Issue #19074 · golang/go"
    url: https://github.com/golang/go/issues/19074
  - title: "all: end support for ppc64/linux (big-endian, ELFv1) in Go 1.27 · Issue #76244 · golang/go"
    url: https://github.com/golang/go/issues/76244
  - title: "Review Minutes（2026-01-28）"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814233279
---
## 概要
Go 1.27において、linux/ppc64（ビッグエンディアン）ターゲットがELFv1 ABIからELFv2 ABIへ移行します。現在のlinux/ppc64は内部リンクのみ（CGO非対応）であるため、この変更はユーザーに対して透過的と見込まれています。

## ステータス変更
**likely_accept** → **accepted**

この決定は、当初のproposal（ELFv1対応の終了）から大きく方針転換されました。コミュニティメンバーからの要望により、ppc64ビッグエンディアン対応を廃止するのではなく、ELFv2への移行によってポートを維持する方向で承認されました。

## 技術的背景

### 現状の問題点
- **サポート状況の不透明さ**: 当初提案では「アクティブにサポートされているpowerpc64 ELFv1 Linuxディストリビューションが存在しない」とされ、メンテナが離職したことでOSU（オレゴン州立大学）のコミュニティビルダーの継続的な更新が困難になっていました
- **ELFv1の時代遅れ**: ELFv1は旧式のABIであり、近年のツールチェーン（LLVM、Zig等）はELFv2のみをサポートする方向に移行しています
- **CGOの非対応**: 現在のlinux/ppc64ターゲットは内部リンクのみで、CGOや外部リンクが利用できません

### 提案された解決策
Issue内の議論を経て、以下の技術的アプローチが採用されました:

1. **ABI切り替え**: linux/ppc64ターゲットをELFv1からELFv2に切り替え（CL#734540で実装）
2. **ターゲット名維持**: `GOOS=linux GOARCH=ppc64`という既存のターゲット名はそのまま維持
3. **透過的な移行**: 現在のlinux/ppc64は静的リンクのみのため、ELFv2システム上でもバイナリは動作し、変更は透過的
4. **ビルダーの更新**: 後続作業として、ビルダーをELFv2システムイメージに更新（ArchPOWER等のディストリビューションを検討）

### ELFv1とELFv2の技術的差異
- **関数呼び出し規約**: ELFv1ではTOCポインタ（r2）を呼び出し元が設定するのに対し、ELFv2では呼び出し先が設定します
- **関数ディスクリプタ**: ELFv1では外部公開関数ごとに`.opd`セクションに関数ディスクリプタを格納しますが、ELFv2では不要
- **モダン化**: ELFv2はmusl、FreeBSD等の最新環境で標準となっており、リトルエンディアンのppc64leも既にELFv2を使用

## これによって何ができるようになるか

### 既存機能の継続
- ppc64ビッグエンディアン対応が維持され、ArchPOWER、Chimera Linux、Gentoo（musl）、Adélie Linux等のELFv2ディストリビューションでGoが利用可能になります

### 将来的な可能性
- **CGOサポート**: ELFv2への移行により、CGO対応が容易になる可能性があります（現時点では非対応のまま）
- **コード簡素化**: ELFv1固有のロジックを削除でき、ppc64leとppc64の実装がエンディアンス以外ほぼ同一になります

### コード例

現在のlinux/ppc64は内部リンクのみのため、ユーザーコードレベルでの変更は不要です:

```go
// Go 1.26以前（ELFv1）
// GOOS=linux GOARCH=ppc64 go build main.go
// → ELFv1バイナリが生成されるが、外部リンク不可

// Go 1.27以降（ELFv2）
// GOOS=linux GOARCH=ppc64 go build main.go
// → ELFv2バイナリが生成される
// ELFv1/ELFv2どちらのLinuxカーネルでも動作（透過的）
```

**重要**: POWER8以降のCPUが必須です（Go 1.9以降の要件）。POWER4系（G5など）では動作しません。

## 議論のハイライト

- **当初提案の変更**: 最初は「ppc64ビッグエンディアン対応の終了」が提案されたが、コミュニティメンバー（@Gelbpunkt氏）から「ELFv2へ移行して継続」という代替案が提示されました
- **実装の容易さ**: CL#734540で示されたように、ELFv2への切り替えに必要なコード変更は非常に少なく、既存のppc64le/openbsd実装を活用できました
- **ビルダー維持の課題**: ポート継続の条件として、コミュニティによるELFv2ビルダーの提供と保守が必要です。OSUOLとの調整が進行中
- **後方互換性**: ELFv1システムのサポートは明示的に終了しますが、実際にはELFv1ディストリビューション自体がほぼ存在しないため影響は限定的
- **透過性の確認**: 「現在の静的リンクのELFv1バイナリはELFv2システム上でも動作するか？」という点が議論され、Linuxカーネルは両ABI対応のため問題ないことが確認されました
- **Go 1.26での事前通知**: リリースノートで「Go 1.27でELFv2に移行する」旨を事前告知し、ユーザーに準備期間を提供

## 関連リンク

- [関連Issue #34850: ppc64 BE port削除提案（クローズ済み）](https://github.com/golang/go/issues/34850)
- [all: require POWER8 support for ppc64, to match ppc64le · Issue #19074 · golang/go](https://github.com/golang/go/issues/19074)
- [all: end support for ppc64/linux (big-endian, ELFv1) in Go 1.27 · Issue #76244 · golang/go](https://github.com/golang/go/issues/76244)
- [Review Minutes（2026-01-28）](https://github.com/golang/go/issues/33502#issuecomment-3814233279)
