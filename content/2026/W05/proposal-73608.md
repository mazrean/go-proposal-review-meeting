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
  - title: "関連Issue #37503"
    url: https://github.com/golang/go/issues/37503
  - title: "関連Issue #46802"
    url: https://github.com/golang/go/issues/46802
  - title: "関連Issue #65355"
    url: https://github.com/golang/go/issues/65355
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/73608
---

## 要約

## 概要
このproposalは、Go言語に新しい`GOOS`ターゲット（`GOOS=none`または`GOOS=tamago`）を追加し、OSの直接的なサポートなしにベアメタル上でGoランタイムを実行可能にすることを提案しています。現在TamaGoプロジェクトで実装されている機能の上流への統合を目指しており、AMDx64、ARM、RISC-V上でOS無しに純粋なGoアプリケーションを実行できるようにします。

## ステータス変更
**discussions** → **active**

このステータス変更は、2026年1月28日のProposal Review Meetingで行われました。提案委員会は「原則として提案のアイデアを支持する」と表明し、実装の詳細を詰める段階に入りました。特に、これまで使用していた`go:linkname`アプローチを「パッケージオーバーレイ方式」に置き換えることを要求しています。提案者は`GOOSPKG`環境変数を使った新しい実装の実証に取り組んでおり、活発な議論が継続中です。

## 技術的背景

### 現状の問題点
現在、Goアプリケーションは何らかのOS（Linux、Windows、macOSなど）上で動作することが前提となっています。組み込みシステムやセキュリティクリティカルなファームウェアでは、以下の問題があります:

- OSの存在自体が攻撃対象となる（攻撃面が広い）
- C言語で書かれたOSやランタイムへの依存が避けられない
- メモリ安全性やコンカレンシーといったGoの利点を低レイヤーで活かせない

現在のTamaGoプロジェクトは独自に`GOOS=tamago`を実装していますが、Goの上流からフォークしたバージョンを維持する必要があり、長期的な保守負担が課題です。

### 提案された解決策
Goランタイムに「外界とのインターフェース」を定義し、OS syscallの代わりにアプリケーション定義の関数を使用できるようにします。具体的には:

1. **新しい`GOOS`値**: `GOOS=none`（または`tamago`/`custom`）を追加
2. **`GOOSPKG`環境変数**: カスタムGOOS実装のパッケージパスを指定
   - 例: `GOOSPKG=github.com/example/myos-runtime@v1.0.0`
   - ローカルパス指定も可能: `GOOSPKG=${PWD}`
3. **`runtime/goos`パッケージ**: ランタイムが必要とする関数群を定義
   - `Hwinit0()`, `Hwinit1()`: ハードウェア初期化
   - `Nanotime()`: 時刻取得
   - `Printk()`: 文字出力
   - `GetRandomData()`: 乱数生成
   - `RamStart`, `RamSize`, `RamStackOffset`: メモリレイアウト定義
   - その他、オプション関数: `Idle`, `Exit`, `Task`, `ProcID`など

実装方式として、FIPS 140のパッケージオーバーレイ機構を拡張し、`GOOSPKG`で指定されたパッケージを`runtime/goos`として扱います。

## これによって何ができるようになるか

### 1. ベアメタル組み込みシステム開発
ARM/RISC-VのSoC上でOSなしにGoアプリケーションを実行できます。USB Armory、Raspberry Pi Zero、NXP i.MX6/i.MX8などの実機で動作実績があります。

### 2. セキュアなファームウェア開発
TamaGoプロジェクトでは以下が実装済み:
- **GoTEE**: Trusted Execution Environment
- **GoKey**: OpenPGP/FIDO U2Fスマートカード
- **ArmoredWitness**: トランスペアレンシーログの証人ネットワーク
- **go-boot**: 100% Go製のUEFIブートローダー

### 3. 軽量マイクロVM／Unikernel
Cloud Hypervisor、Firecracker、QEMUのmicroVM上で、Goだけで動作するユニカーネルを構築できます。

### 4. OSのユーザースペース実行（テスト用途）
Linuxユーザースペース上で`GOOS=none`アプリケーションを実行し、標準ライブラリのテストを行えます。現在のTamaGoはこの方法でGoの標準ライブラリテストをほぼ全てパスしています。

### コード例

```go
// Before: 従来はOSが必須
// Linux/Windows/macOS上でしか実行できない
package main

import "net/http"

func main() {
    http.ListenAndServe(":8080", nil) // OS依存のネットワークスタック
}
```

```go
// After: GOOS=noneでベアメタル実行可能
package main

import (
    "net/http"
    _ "github.com/example/board-support" // ボード固有初期化
)

func main() {
    // gVisorなどのユーザーランドTCP/IPスタックと組み合わせて
    // OSなしにHTTPサーバーが動作
    http.ListenAndServe(":8080", nil)
}
```

**GOOS実装パッケージの例**:
```go
// github.com/example/myboard/goos/goos.go
package goos

import "unsafe"

// 必須: メモリレイアウト定義
var (
    RamStart       uint = 0x80000000
    RamSize        uint = 0x20000000
    RamStackOffset uint = 0x100
)

// 必須: ハードウェア初期化
func Hwinit0() {
    // CPU初期化、割り込みテーブル設定など
}

func Hwinit1() {
    // メモリアロケータ起動後の初期化
}

// 必須: 時刻取得
func Nanotime() int64 {
    // ハードウェアタイマーから時刻を取得
    return getHardwareTimer()
}

// 必須: 文字出力（デバッグ用）
func Printk(c byte) {
    // UART経由で出力
    sendToUART(c)
}

// オプション: アイドル時の省電力処理
var Idle = func(until int64) {
    enterLowPowerMode(until)
}
```

## 議論のハイライト

1. **パッケージオーバーレイ方式への移行が必須**: 当初`go:linkname`を使った実装が提案されましたが、提案委員会は型安全性とインライン化のため、FIPS 140と同様のパッケージオーバーレイ方式を要求しました。

2. **GOOS名称の議論**: `none`, `noos`, `custom`, `tamago`などが候補。委員会は`none`を好みますが、カスタムOS実装も可能にする観点では`custom`も検討されています。

3. **ビルドタグとパッケージパス**: `GOOS=none`だけでなく、特定の実装を指定するためのビルドタグ構文が議論されています。`//go:build github.com/example/myos-go/runtime`のようなパッケージパス指定の可否が検討中です。

4. **循環依存の課題**: `runtime/goos`パッケージが`runtime`をインポートできないため、実装に制約があります。Goチームは`runtime`パッケージの分割（`internal/runtime/syscall`, `internal/runtime/maps`など）を進めており、この方向性と整合します。

5. **実装の成熟度**: TamaGoプロジェクトは2019年から継続開発されており、Go 1.12→1.13→1.14と順次アップデート。現在はGo 1.26対応版で、標準ライブラリテストのほぼ全てをパスしています。AMD64、ARM、ARM64、RISC-V64の4アーキテクチャをサポートし、本番環境での使用実績もあります。

6. **次のステップ**: 提案者はパッケージオーバーレイ方式の実装を進めており、`GOOSPKG`環境変数を使ったPoC（proof of concept）を公開済み。委員会は詳細が固まり次第、承認に進む意向を示しています。

## 関連リンク

- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連Issue #37503](https://github.com/golang/go/issues/37503)
- [関連Issue #46802](https://github.com/golang/go/issues/46802)
- [関連Issue #65355](https://github.com/golang/go/issues/65355)
- [Proposal Issue](https://github.com/golang/go/issues/73608)
