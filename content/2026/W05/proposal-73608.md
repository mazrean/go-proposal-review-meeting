---
issue_number: 73608
title: "all: add bare metal support"
previous_status: discussions
current_status: active
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "過去の提案 #37503"
    url: https://github.com/golang/go/issues/37503
  - title: "過去の提案 #46802"
    url: https://github.com/golang/go/issues/46802
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/73608
---

## 要約

## 概要
Goをベアメタル環境（OS無し）で実行可能にする`GOOS=none`（または`GOOS=tamago`）の追加提案です。これにより、ハイパーバイザー、UEFIファームウェア、組み込みシステムなど、従来のOSが存在しない環境でもGoアプリケーションを実行できるようになります。現在、外部プロジェクトとして開発されている[TamaGo](https://github.com/usbarmory/tamago)の実装を上流に取り込むことを目指しています。

## ステータス変更
**discussions** → **active**

2026年1月28日のProposal Review Meetingで「active」ステータスに移行し、コメントが追加されました。議論は「commented」として継続中となっています。この決定は、過去3回の類似提案（#35956、#37503、#46802）がすべて「保守負担の増加」を理由に却下されてきた経緯を踏まえ、慎重に検討されています。

主要な議論の焦点は、この提案を**2つの異なる視点**で捉えることができる点です：

1. **通常のGOOS移植として扱う**: Go 1互換性保証が完全に適用されるべき
2. **ツリー外移植のためのフレームワークとして扱う**: 合理的な安定性を提供するが完全な互換性保証はなく、移植メンテナーが新しいGoバージョンに追従する責任を持つ

この2つの解釈の間で、Go開発チームは慎重に方向性を模索しています。

## 技術的背景

### 現状の問題点
現在、Goアプリケーションは必ずOSのシステムコール（Linux、Windows、macOSなど）に依存して実行されます。しかし、以下のような環境ではOSが存在しません：

- **ベアメタル組み込みシステム**: USB armoryなどのセキュリティデバイス
- **UEFIブートローダー**: OS起動前のファームウェア層
- **マイクロVM環境**: FirecrackerなどのサンドボックスVM
- **ハイパーバイザー環境**: 最小限のランタイムのみを持つ仮想化環境

これらの環境でGoを動かすには、現在はGoコンパイラ自体をフォークする必要があり、以下の課題があります：

- 上流の変更を追従するコストが高い
- 標準ライブラリのテストが困難
- コミュニティとの分断

### 提案された解決策

Goランタイムに**最小限のフック機構**を追加し、アプリケーションや外部のBoard Support Package（BSP）が以下の関数を実装できるようにします：

**主要なランタイムフック**:
- `runtime.hwinit0()`: ワールド開始前の初期化（Go Assemblyで実装）
- `runtime.hwinit1()`: ワールド開始後の初期化（通常のGoコードで実装可能）
- `runtime.nanotime1()`: ナノ秒単位のシステム時刻を返す（高速・no-allocが必須）
- `runtime.GetRandomData()`: 乱数生成
- `runtime.printk()`: 低レベル出力（デバッグ用）
- `runtime.RamStart`, `runtime.RamSize`: メモリレイアウト定義
- `runtime.Exit()`: プログラム終了処理

これらのフックにより、**Goランタイム自体は変更せず**、ハードウェア固有の実装を外部パッケージに委譲できます。

## これによって何ができるようになるか

### 1. セキュリティ重視のファームウェア開発
メモリ安全なGoで書かれたUEFIブートローダーやTrusted Execution Environment（TEE）を構築できます。実例として、[Transparency.dev Armored Witness](https://github.com/transparency-dev/armored-witness)プロジェクトでは、すべてのファームウェア（ブートローダー、OS、アプレット）をTamaGoで実装し、数千台のデバイスで稼働しています。

### 2. マイクロVM/Unikernel環境
Firecracker microVMやGoogle Compute Engine（AMD SEV-SNP含む）で、最小限のランタイムオーバーヘッドでGoアプリケーションを実行できます。

### 3. 組み込みシステム開発
ARMやRISC-V SoCで、Cランタイムなしで完全なGoアプリケーションを動作させられます。TinyGoと異なり、**標準Go言語仕様と標準ライブラリの100%互換性**を維持できます。

### コード例

```go
// Before: ベアメタル実行は不可能
// 従来はLinux/Windows/macOS上でのみ動作

// After: GOOS=noneでのベアメタル実行
package main

import (
    _ "github.com/usbarmory/tamago/board/usbarmory/mk2"
)

func main() {
    // Goランタイムは完全に動作
    // goroutine、channel、GC、すべて利用可能

    ch := make(chan int)
    go func() {
        ch <- 42
    }()

    result := <-ch
    println(result) // ベアメタルで動作するprintln
}
```

**BSP実装例** (アプリケーション/BSPパッケージ側):
```go
//go:linkname nanotime1 runtime.nanotime1
func nanotime1() int64 {
    // ハードウェア固有のタイマーを直接読み取り
    return read_systimer() * ARM.TimerMultiplier + ARM.TimerOffset
}

//go:linkname RamSize runtime.ramSize
var RamSize uint = 512 * 1024 * 1024  // 512MB

//go:linkname RamStart runtime.ramStart
var RamStart uint = 0x10000000
```

## 議論のハイライト

### 1. 互換性保証の範囲が最大の課題
Go開発チーム（@neild、@cherrymui）は、ランタイムフック関数が「Goの安全なサブセット」で実装される必要性を強調しています。特に`nanotime1()`は以下の制約を満たす必要があります：

- スタック拡張不可（`//go:nosplit`）
- メモリアロケーション不可
- ヒープオブジェクトへの書き込み不可（`//go:nowritebarrierrec`）

**解決策**: これらの関数はGo Assemblyで実装するか、または専用のコンパイラチェック機構を導入することで対応可能とされています。

### 2. 「GOOSポート」vs「ツリー外移植フレームワーク」
提案者の@apparentlymartと@neildは、この提案を「ニッチなOSへの移植を容易にする汎用フレームワーク」として位置づけることで、受け入れやすくなると指摘しています。完全な互換性保証ではなく「合理的な安定性」を提供する方向性です。

### 3. 保守責任の明確化
過去の提案が却下された最大の理由は「Goコアチームへの保守負担」でした。今回は以下の点で改善：

- **5年間の実績**: TamaGoプロジェクトが2020年から継続的にメンテナンスされている
- **2名の専任メンテナー**: 提案者が責任を持って保守することを約束
- **ビルダー提供**: 標準ライブラリテストを実行するCI環境を提供予定
- **実運用実績**: WithSecure、Transparency.devなどの企業/プロジェクトで数千台規模の採用

### 4. TinyGoとの棲み分け
TinyGoはLLVMベースの別実装で、主にマイクロコントローラー（RAM < 1MB）をターゲットとし、言語仕様の一部制限があります。一方、TamaGoは**標準Go処理系の最小限の修正**で、SoC（System-on-Chip）クラスのハードウェアでフル機能を提供します。TinyGoメンテナーも本提案に関心を示しています。

### 5. マルチスレッド対応
現在TamaGoはシングルスレッドですが、SMP（対称型マルチプロセッシング）対応を開発中です。ランタイムAPIには影響を与えず、`newosproc`の実装で対応予定とのことです。

## 関連リンク
- [Proposal Issue](https://github.com/golang/go/issues/73608)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [TamaGo プロジェクト](https://github.com/usbarmory/tamago)
- [GOOS=tamago実装差分](https://github.com/golang/go/compare/go1.24.4..usbarmory:tamago1.24.4)
- [Armored Witness（実運用例）](https://github.com/transparency-dev/armored-witness)
- [過去の提案 #37503](https://github.com/golang/go/issues/37503)
- [過去の提案 #46802](https://github.com/golang/go/issues/46802)

**Sources**:
- [GitHub - usbarmory/tamago: TamaGo - bare metal Go](https://github.com/usbarmory/tamago)
- [Frequently Asked Questions (FAQ) · usbarmory/tamago Wiki · GitHub](https://github.com/usbarmory/tamago/wiki/Frequently-Asked-Questions-(FAQ))
- [GitHub - transparency-dev/armored-witness](https://github.com/transparency-dev/armored-witness)
- [Armored Witness: Building a Trusted Notary for Bare Metal](https://www.secwest.net/presentations-2024/armored-witness-building-a-trusted-notary-for-bare-metal)

## 関連リンク

- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [過去の提案 #37503](https://github.com/golang/go/issues/37503)
- [過去の提案 #46802](https://github.com/golang/go/issues/46802)
- [Proposal Issue](https://github.com/golang/go/issues/73608)
