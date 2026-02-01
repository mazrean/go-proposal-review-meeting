---
issue_number: 76244
title: "all: transition ppc64/linux (big-endian) from ELFv1 to ELFv2 in Go 1.27"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/76244#issuecomment-3814233279
  - title: "all: transition ppc64/linux (big-endian) from ELFv1 to ELFv2 in Go 1.27 · Issue #76244 · golang/go"
    url: https://github.com/golang/go/issues/76244
  - title: "Review Comment"
    url: https://github.com/golang/go/issues/33502#issuecomment-3720969090
  - title: "Review Minutes (2026-01-28)"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "proposal: remove linux/ppc64 (big endian) port"
    url: https://github.com/golang/go/issues/34850
  - title: "all: require POWER8 support for ppc64, to match ppc64le"
    url: https://github.com/golang/go/issues/19074
---

## 要約

## 概要
Linux上のビッグエンディアンPowerPC 64ビット環境（`linux/ppc64`）について、従来のELFv1 ABIから現代的なELFv2 ABIへの移行をGo 1.27で実施する提案が承認されました。

## ステータス変更
**likely_accept** → **accepted**

この決定は、2026年1月28日の週次Proposal Review Meetingで正式に承認されました。当初は`linux/ppc64`ポート自体の廃止が提案されていましたが、コミュニティからELFv2への移行要望が出されたことで、方針が大きく転換されました。議論の中で、ELFv2を使用する複数のビッグエンディアンLinuxディストリビューション（ArchPOWER、Chimera Linux、Gentoo、Adélie Linux等）が存在することが判明し、ELFv1の廃止とELFv2への移行という建設的な解決策に至りました。

## 技術的背景

### 現状の問題点

Go 1.26時点で、`linux/ppc64`（ビッグエンディアン）は以下の課題を抱えています:

- **ELFv1 ABIの使用**: 1990年代に設計された古いABI（Application Binary Interface）を使用
- **ディストリビューションサポートの欠如**: ELFv1をサポートするアクティブなLinuxディストリビューションがほぼ存在しない
- **CGOサポートの欠如**: 外部リンクが未サポートのため、CGOが利用できない
- **メンテナンス負荷**: コミュニティビルダーの維持が困難（IBM在籍者による手動メンテナンスが必要だった）

```go
// 現状: linux/ppc64 (ELFv1) の制限
// CGOは使えず、完全に静的リンクされたバイナリのみ生成可能
// GOARCH=ppc64 GOOS=linux でビルドしても外部Cライブラリとのリンクは不可
```

### 提案された解決策

Go 1.27からELFv2 ABIに切り替えます。ELFv2はELFv1の後継として設計された現代的なABIで、以下の特徴があります:

- **関数ディスクリプタの廃止**: ELFv1では各関数に`.opd`セクションの関数ディスクリプタ（エントリポイント、TOCベースアドレス、環境ポインタの3つの要素）が必要でしたが、ELFv2ではこれが不要になり効率が向上
- **TOCポインタ管理の簡素化**: 呼び出し側ではなく呼び出される側がTOCポインタ（r2レジスタ）を設定する方式に変更
- **リトルエンディアンとの統一**: `ppc64le`（リトルエンディアン）は既にELFv2を使用しており、ビッグエンディアンもELFv2にすることでエンディアンの違いのみが差異となる

## これによって何ができるようになるか

この変更により以下のメリットが得られます:

1. **完全な透明性**: 現在の`linux/ppc64`は内部リンクのみのため、既存バイナリは引き続きELFv2環境で動作します
2. **CGOサポートの道**: ELFv2移行により、将来的にCGOサポートを追加することが容易になります
3. **コードベースの簡素化**: ELFv1固有のロジックを削除でき、`ppc64le`と実質的に同じ実装になります
4. **現代的なディストリビューションのサポート**: musl libcやモダンなツールチェーン（LLD、Zigなど）との互換性が向上します

### コード例

```go
// Before (Go 1.26まで): ELFv1
// GOARCH=ppc64 GOOS=linux でビルド
// - 内部リンクのみ
// - CGO_ENABLED=0 が事実上の制約
// - ELFv1 ABIバイナリを生成（ただしELFv2環境でも動作）

// After (Go 1.27以降): ELFv2
// GOARCH=ppc64 GOOS=linux でビルド
// - ELFv2 ABIバイナリを生成
// - 既存コードの変更は不要（完全に透明な移行）
// - 将来的にCGOサポートが追加される可能性
// - ppc64leとエンディアンのみが異なる実装に
```

実装に関しては、CL 734540で驚くほど少ないコード変更で移行が実現されています。主な変更は`cmd/internal/obj/ppc64`パッケージ内のABI識別子の切り替えのみです。

## 議論のハイライト

- **当初の提案は削除**: 提案者（元IBM）は、メンテナンスの困難さとELFv1ディストリビューションの不在を理由に`linux/ppc64`の完全削除を提案していました
- **コミュニティからの代替案**: @GelbpunktがELFv2への移行を提案し、複数の現役ディストリビューションが存在することを示しました（Zigも同様にELFv1を削除しELFv2のみサポートする方針を採用）
- **透明性の確認**: 現在の`linux/ppc64`は内部リンクのみのため、ELFv1バイナリがELFv2環境で動作することが確認され、移行が既存ユーザーに影響しないことが判明しました
- **ビルダーの確保**: OSUOSLでArchPOWERを使用した新しいELFv2ビルダーの立ち上げが進行中です。@Gelbpunktがメンテナンスにコミットする意向を示しました
- **セカンダリポートとしての継続**: この変更後も`linux/ppc64`はセカンダリポートとして維持され、コミュニティによるメンテナンスとビルダー管理が必要です
- **Go 1.26での事前告知**: Go 1.26のリリースノートにELFv2移行の予告が含まれ、ユーザーへの通知が行われています

## 関連リンク
- [Proposal Issue](https://github.com/golang/go/issues/76244)
- [Review Minutes](https://github.com/golang/go/issues/76244#issuecomment-3814233279)
- [関連Issue: proposal: remove linux/ppc64 (big endian) port #34850](https://github.com/golang/go/issues/34850)
- [関連Issue: all: require POWER8 support for ppc64, to match ppc64le #19074](https://github.com/golang/go/issues/19074)
- [実装CL 734540](https://go.dev/cl/734540)
- [Go 1.26 Release Notes - ELFv1 Deprecation](https://go.dev/doc/go1.26)
- [実装CL 738120: Pre-announcement in Go 1.26 release notes](https://go.dev/cl/738120)

## Sources
- [all: transition ppc64/linux (big-endian) from ELFv1 to ELFv2 in Go 1.27 · Issue #76244 · golang/go](https://github.com/golang/go/issues/76244)
- [Linker notes on Power ISA | MaskRay](https://maskray.me/blog/2023-02-26-linker-notes-on-power-isa)
- [Go 1.26 Release Notes - The Go Programming Language](https://go.dev/doc/go1.26)
- [proposal: remove linux/ppc64 (big endian) port](https://github.com/golang/go/issues/34850)
- [all: require POWER8 support for ppc64, to match ppc64le](https://github.com/golang/go/issues/19074)

## 関連リンク

- [Review Minutes](https://github.com/golang/go/issues/76244#issuecomment-3814233279)
- [all: transition ppc64/linux (big-endian) from ELFv1 to ELFv2 in Go 1.27 · Issue #76244 · golang/go](https://github.com/golang/go/issues/76244)
- [Review Comment](https://github.com/golang/go/issues/33502#issuecomment-3720969090)
- [Review Minutes (2026-01-28)](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [proposal: remove linux/ppc64 (big endian) port](https://github.com/golang/go/issues/34850)
- [all: require POWER8 support for ppc64, to match ppc64le](https://github.com/golang/go/issues/19074)
