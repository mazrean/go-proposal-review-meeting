---
issue_number: 74958
title: "go/scanner: add \\`(\\*Scanner).End()\\`"
previous_status: likely_accept
current_status: accepted
changed_at: 2026-01-28T00:00:00Z
comment_url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
related_issues:
  - title: "go/ast: (*ast.BasicLit).End() is wrong for raw literals with carriage returns"
    url: https://github.com/golang/go/issues/69861
  - title: "Proposal Issue"
    url: https://github.com/golang/go/issues/74958
  - title: "Review Minutes"
    url: https://github.com/golang/go/issues/33502#issuecomment-3814236717
  - title: "関連Issue #54941: セミコロンとコメントの順序問題"
    url: https://github.com/golang/go/issues/54941
---
## 概要
`go/scanner`パッケージに`Scanner.End()`メソッドを追加する提案です。このメソッドは、直前にスキャンしたトークンの終了位置を`token.Pos`として返すことで、トークンの正確な終了位置を簡単かつ確実に取得できるようにします。

## ステータス変更
**likely_accept** → **accepted**

この提案は2026年1月21日に「likely accept」となり、1週間の最終コメント期間を経て2026年1月28日に正式承認されました。Proposal Review Groupは、`token.NoPos`を初期値とすることに合意し、一貫性のために`Pos`、`Tok`、`Lit`のアクセサメソッドを追加することも検討しましたが、それは価値がないと判断しました。

## 技術的背景

### 現状の問題点

現在、`go/scanner`でトークンの終了位置を取得する標準的な方法が存在しません。`Scanner.Scan()`は開始位置のみを返すため、開発者は以下のようなワークアラウンドを使用せざるを得ませんでした:

```go
pos, tok, lit := s.Scan()
tokLength := len(lit)
if !tok.IsLiteral() && tok != token.COMMENT {
    tokLength = len(tok.String())
}
tokEnd := pos + token.Pos(tokLength)
```

しかし、このアプローチには以下の重大な問題があります:

1. **キャリッジリターン（`\r`）の扱い**: コメントやraw string literalでは、`\r`が実際のリテラル文字列から除外されるため、`len(lit)`が実際のソース上の長さと一致せず、終了位置の計算が不正確になります。

2. **人工的なセミコロン**: ファイル末尾で`impliedSemi==true`の場合、実際のソースには存在しない人工的な`SEMICOLON`トークンが生成されます。これにより、トークン間の空白を検査しようとするコードが予期しないパニックを起こす可能性があります。

```go
// 問題例: ファイル末尾の人工セミコロンでパニック
const src = "package a; var a int"
// ... スキャナ初期化 ...
prevEndOff := 0
for {
    pos, tok, lit := s.Scan()
    off := file.Offset(pos)

    white := src[prevEndOff:off] // tok == EOF時にパニック！
    // ...
}
```

### 提案された解決策

新しい`End()`メソッドを`Scanner`型に追加します:

```go
package scanner // go/scanner

// End returns the position immediately after the last scanned token.
// If Scanner.Scan has not been called yet, End returns token.NoPos.
func (s *Scanner) End() token.Pos {
    return s.lastTokEnd
}
```

このメソッドは、スキャナが内部で正確に把握している終了位置を直接返すため、上記のようなワークアラウンドが不要になります。

## これによって何ができるようになるか

1. **トークンの正確な範囲取得**: トークンの開始位置（`Scan()`）と終了位置（`End()`）を正確に取得でき、ソースコード解析ツールやリファクタリングツールの精度が向上します。

2. **トークン間の空白/コメント解析**: トークン間の正確な空白やコメントを解析できるようになり、フォーマッタやリンターの実装が簡単になります。

3. **エラー報告の改善**: より正確な位置情報により、ユーザーフレンドリーなエラーメッセージを生成できます。

### コード例

```go
// Before: 従来の書き方（不正確なワークアラウンド）
pos, tok, lit := s.Scan()
tokLength := len(lit)
if !tok.IsLiteral() && tok != token.COMMENT {
    tokLength = len(tok.String())
}
tokEnd := pos + token.Pos(tokLength) // \rを含む場合に不正確

// After: 新APIを使った書き方
pos, tok, lit := s.Scan()
tokEnd := s.End() // 常に正確な終了位置

// 実用例: トークン間の空白を正確に取得
prevEnd := token.NoPos
for {
    pos, tok, lit := s.Scan()
    if tok == token.EOF {
        break
    }

    if prevEnd != token.NoPos {
        // トークン間の空白を正確に取得
        whitespace := src[file.Offset(prevEnd):file.Offset(pos)]
        fmt.Printf("Whitespace: %q\n", whitespace)
    }

    prevEnd = s.End() // 正確な終了位置を保存
}
```

## 議論のハイライト

- **`Pos()`から`End()`への変更**: 当初は「次のスキャン開始位置」を返す`Pos()`メソッドが提案されましたが、@adonovanの提案により「直前のトークンの終了位置」を返す`End()`に変更されました。これにより、raw string literal内のセミコロンでスキャナが「後退」する問題を回避できます。

- **`ScanWithEnd()`の検討**: `ScanWithEnd() (start Pos, _ Token, _ string, end Pos)`という新メソッドも検討されましたが、後方互換性とシンプルさを重視して`End()`メソッドが選択されました。

- **初期値の決定**: `Scan()`未呼び出し時の戻り値として、`token.NoPos`、`file.Base()`、`file.Base() + BOMの長さ`、未定義の4つが検討され、最もシンプルな`token.NoPos`が採用されました。

- **`go/parser`への影響**: `go/parser`も同様の問題（`\r`による`End()`の不正確さ）を抱えていますが、この提案はそれを直接解決するものではありません。ただし、`go/scanner`を内部的に使用することで、`go/parser`のコメント終了位置計算も改善される見込みです（CL 694615、738700、738701で対応）。

- **関連Issue**: #54941（raw string literal内のコメント順序問題）、#41197/#69860/#69861（`\r`による`End()`位置のずれ）など、複数の既知問題の解決に寄与します。

## 関連リンク

- [go/ast: (*ast.BasicLit).End() is wrong for raw literals with carriage returns](https://github.com/golang/go/issues/69861)
- [Proposal Issue](https://github.com/golang/go/issues/74958)
- [Review Minutes](https://github.com/golang/go/issues/33502#issuecomment-3814236717)
- [関連Issue #54941: セミコロンとコメントの順序問題](https://github.com/golang/go/issues/54941)
