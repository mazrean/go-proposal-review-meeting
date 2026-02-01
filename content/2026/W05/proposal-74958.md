---
issue_number: 74958
title: "go/scanner: add \\`(\\*Scanner).End()\\`"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "関連Issue #54941: セミコロンとコメントの順序問題"
    url: https://github.com/golang/go/issues/54941
  - title: "go/ast: (*ast.BasicLit).End() is wrong for raw literals with carriage returns"
    url: https://github.com/golang/go/issues/69861
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/74958#issuecomment-3814233476
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/74958
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
---

## 要約

## 概要
`go/scanner`パッケージの`Scanner`型に、スキャンされた最後のトークンの終了位置を取得する`End()`メソッドを追加する提案です。現在、トークンの終了位置を正確に取得する方法が存在せず、特にコメントやraw文字列リテラルのキャリッジリターン処理、暗黙的なセミコロントークンの扱いにおいて問題がありました。

## ステータス変更
**likely_accept** → **accepted**

2026年1月28日の提案レビューミーティングで正式に承認されました。議論の結果、`Scan()`を呼び出す前に`End()`が呼ばれた場合は`token.NoPos`を返す仕様で合意されました。これは実装が簡単で、実際のクライアントコードではこのエッジケースは問題にならないと判断されたためです。

## 技術的背景

### 現状の問題点
`go/scanner`では、`Scan()`メソッドが`(pos token.Pos, tok token.Token, lit string)`を返しますが、`pos`はトークンの**開始位置**のみを示します。トークンの終了位置を計算するには、以下のような方法が使われていましたが、複数の問題がありました。

```go
pos, tok, lit := s.Scan()
tokLength := len(lit)
if !tok.IsLiteral() && tok != token.COMMENT {
    tokLength = len(tok.String())
}
tokEnd := pos + token.Pos(tokLength)
```

**問題1: キャリッジリターンの扱い**
コメントやraw文字列リテラルでは、キャリッジリターン(`'\r'`)が`lit`に含まれないため、`len(lit)`では正確な長さが得られず、終了位置が誤って計算されます。

**問題2: 暗黙的なセミコロン**
ファイル終端で`impliedSemi==true`の場合、人工的な`SEMICOLON`トークンが生成されます。この場合、トークン間の空白を検証しようとすると、予期しない位置情報によりコードがパニックを起こします。

### 提案された解決策
`Scanner`型に新しいメソッド`End()`を追加します。

```go
// End returns the position immediately after the last scanned token.
// If Scanner.Scan has not been called yet, End returns token.NoPos.
func (s *Scanner) End() token.Pos
```

このメソッドは、`Scan()`の副作用として内部フィールド`lastTokEnd`に設定される値を返します。これにより、スキャナー自身が正確に把握している終了位置を直接取得できます。

## これによって何ができるようになるか

### トークン間の空白の正確な検証
従来は不可能だった、トークン間の空白を正確に抽出・検証する処理が可能になります。

### コード例

```go
// Before: 従来の方法（問題あり）
file := token.NewFileSet().AddFile("", -1, len(src))
var s scanner.Scanner
s.Init(file, []byte(src), nil, scanner.ScanComments)

prevEndOff := 0
for {
    pos, tok, lit := s.Scan()
    off := file.Offset(pos)

    // トークン間の空白を検証しようとするが、
    // 暗黙的セミコロンでパニックが発生
    white := src[prevEndOff:off]

    tokLength := len(lit)
    if !tok.IsLiteral() && tok != token.COMMENT {
        tokLength = len(tok.String())
    }
    prevEndOff = off + tokLength // キャリッジリターンで不正確

    if tok == token.EOF {
        break
    }
}

// After: 新APIを使った方法
file := token.NewFileSet().AddFile("", -1, len(src))
var s scanner.Scanner
s.Init(file, []byte(src), nil, scanner.ScanComments)

for {
    pos, tok, _ := s.Scan()
    end := s.End() // 正確な終了位置を取得

    off := file.Offset(pos)
    endOff := file.Offset(end)

    // トークン間の空白を安全に取得可能
    if off > 0 {
        white := src[prevEndOff:off]
        // 空白文字の検証...
    }

    prevEndOff = endOff

    if tok == token.EOF {
        break
    }
}
```

## 議論のハイライト

- **`Pos()`ではなく`End()`に変更**: 当初は「次のスキャン開始位置」を返す`Pos()`メソッドが提案されましたが、Alan Donovanの提案により「前回のトークンの終了位置」を返す`End()`に変更されました。これにより、セミコロンの扱いでスキャナーが「後戻り」する問題を回避し、仕様がより明確になりました。

- **`ScanWithEnd()`という代替案**: 新しい`ScanWithEnd() (start Pos, _ Token, _ string, end Pos)`メソッドも検討されましたが、後方互換性を保つため`End()`メソッドの追加が選ばれました。

- **初期値の選択**: `Scan()`が未呼び出しの場合の戻り値として、`token.NoPos`、`file.Base()`、BOM込みの`file.Base()`、未定義の4つの選択肢が議論され、最も実装が簡単で実用上問題がない`token.NoPos`に決定しました。

- **`go/parser`の同様の問題**: この提案は`go/scanner`を対象としていますが、コメント内で`go/parser`のASTノードの`End()`メソッドにも同様のキャリッジリターンに関する問題があることが指摘されました。この新しいAPIにより、ワークアラウンドとして`go/scanner`を利用した正確な終了位置の取得が可能になります。

- **一貫性の議論**: 現在の`Scanner` APIは`Scan()`から全ての状態を返すタプル形式ですが、`End`のために拡張することは後方互換性上不可能です。`Pos`、`Tok`、`Lit`のアクセサメソッドも一貫性のために追加できますが、価値がないと判断されました。

## 関連リンク
- [Proposal Issue](https://github.com/golang/go/issues/74958)
- [Review Minutes](https://github.com/golang/go/issues/74958#issuecomment-3814233476)
- [関連Issue #54941: セミコロンとコメントの順序問題](https://github.com/golang/go/issues/54941)
- [実装CL 738681](https://go.dev/cl/738681)
- [go/parserでの利用CL 738701](https://go.dev/cl/738701)

Sources:
- [scanner package - go/scanner - Go Packages](https://pkg.go.dev/go/scanner)
- [token package - go/token - Go Packages](https://pkg.go.dev/go/token)
- [A look at Go lexer/scanner packages](https://arslan.io/2015/10/12/a-look-at-go-lexerscanner-packages/)

## 関連リンク

- [関連Issue #54941: セミコロンとコメントの順序問題](https://github.com/golang/go/issues/54941)
- [go/ast: (*ast.BasicLit).End() is wrong for raw literals with carriage returns](https://github.com/golang/go/issues/69861)
- [Review Minutes](https://github.com/golang/go/issues/74958#issuecomment-3814233476)
- [Proposal Issue](https://github.com/golang/go/issues/74958)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
