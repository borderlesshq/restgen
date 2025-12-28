package com.borderlesshq.restgen.lexer

import com.intellij.lexer.LexerBase
import com.intellij.psi.tree.IElementType

class SdlLexer : LexerBase() {
    private var buffer: CharSequence = ""
    private var startOffset = 0
    private var endOffset = 0
    private var tokenStart = 0
    private var tokenEnd = 0
    private var currentToken: IElementType? = null

    override fun start(buffer: CharSequence, startOffset: Int, endOffset: Int, initialState: Int) {
        this.buffer = buffer
        this.startOffset = startOffset
        this.endOffset = endOffset
        this.tokenStart = startOffset
        this.tokenEnd = startOffset
        advance()
    }

    override fun getState(): Int = 0

    override fun getTokenType(): IElementType? = currentToken

    override fun getTokenStart(): Int = tokenStart

    override fun getTokenEnd(): Int = tokenEnd

    override fun getBufferSequence(): CharSequence = buffer

    override fun getBufferEnd(): Int = endOffset

    override fun advance() {
        tokenStart = tokenEnd

        if (tokenStart >= endOffset) {
            currentToken = null
            return
        }

        val c = buffer[tokenStart]

        when {
            // Whitespace
            c.isWhitespace() -> {
                tokenEnd = tokenStart + 1
                while (tokenEnd < endOffset && buffer[tokenEnd].isWhitespace()) {
                    tokenEnd++
                }
                currentToken = com.intellij.psi.TokenType.WHITE_SPACE
            }

            // Comments (# or //)
            c == '#' || (c == '/' && tokenStart + 1 < endOffset && buffer[tokenStart + 1] == '/') -> {
                tokenEnd = tokenStart
                while (tokenEnd < endOffset && buffer[tokenEnd] != '\n') {
                    tokenEnd++
                }
                val text = buffer.subSequence(tokenStart, tokenEnd).toString()
                currentToken = when {
                    text.contains("@include") -> SdlTokenTypes.DIRECTIVE_COMMENT_INCLUDE
                    text.contains("@base") -> SdlTokenTypes.DIRECTIVE_COMMENT_BASE
                    text.contains("@models") -> SdlTokenTypes.DIRECTIVE_COMMENT_MODELS
                    else -> SdlTokenTypes.COMMENT
                }
            }

            // String
            c == '"' -> {
                tokenEnd = tokenStart + 1
                while (tokenEnd < endOffset) {
                    val ch = buffer[tokenEnd]
                    if (ch == '"') {
                        tokenEnd++
                        break
                    }
                    if (ch == '\\' && tokenEnd + 1 < endOffset) {
                        tokenEnd += 2
                    } else {
                        tokenEnd++
                    }
                }
                currentToken = SdlTokenTypes.STRING
            }

            // Punctuation
            c == '{' -> { tokenEnd = tokenStart + 1; currentToken = SdlTokenTypes.LBRACE }
            c == '}' -> { tokenEnd = tokenStart + 1; currentToken = SdlTokenTypes.RBRACE }
            c == '(' -> { tokenEnd = tokenStart + 1; currentToken = SdlTokenTypes.LPAREN }
            c == ')' -> { tokenEnd = tokenStart + 1; currentToken = SdlTokenTypes.RPAREN }
            c == '[' -> { tokenEnd = tokenStart + 1; currentToken = SdlTokenTypes.LBRACKET }
            c == ']' -> { tokenEnd = tokenStart + 1; currentToken = SdlTokenTypes.RBRACKET }
            c == ':' -> { tokenEnd = tokenStart + 1; currentToken = SdlTokenTypes.COLON }
            c == ',' -> { tokenEnd = tokenStart + 1; currentToken = SdlTokenTypes.COMMA }
            c == '!' -> { tokenEnd = tokenStart + 1; currentToken = SdlTokenTypes.EXCLAIM }
            c == '.' -> { tokenEnd = tokenStart + 1; currentToken = SdlTokenTypes.DOT }

            // Directive (@...)
            c == '@' -> {
                tokenEnd = tokenStart + 1
                while (tokenEnd < endOffset && (buffer[tokenEnd].isLetterOrDigit() || buffer[tokenEnd] == '_')) {
                    tokenEnd++
                }
                val text = buffer.subSequence(tokenStart, tokenEnd).toString()
                currentToken = when (text) {
                    "@get" -> SdlTokenTypes.DIRECTIVE_GET
                    "@post" -> SdlTokenTypes.DIRECTIVE_POST
                    "@put" -> SdlTokenTypes.DIRECTIVE_PUT
                    "@patch" -> SdlTokenTypes.DIRECTIVE_PATCH
                    "@delete" -> SdlTokenTypes.DIRECTIVE_DELETE
                    "@base" -> SdlTokenTypes.DIRECTIVE_BASE
                    "@models" -> SdlTokenTypes.DIRECTIVE_MODELS
                    "@include" -> SdlTokenTypes.DIRECTIVE_INCLUDE
                    else -> SdlTokenTypes.DIRECTIVE_OTHER
                }
            }

            // Number
            c == '-' || c.isDigit() -> {
                tokenEnd = tokenStart
                if (c == '-') tokenEnd++
                while (tokenEnd < endOffset && buffer[tokenEnd].isDigit()) {
                    tokenEnd++
                }
                if (tokenEnd < endOffset && buffer[tokenEnd] == '.') {
                    tokenEnd++
                    while (tokenEnd < endOffset && buffer[tokenEnd].isDigit()) {
                        tokenEnd++
                    }
                }
                currentToken = SdlTokenTypes.NUMBER
            }

            // Identifier or keyword
            c.isLetter() || c == '_' -> {
                tokenEnd = tokenStart
                while (tokenEnd < endOffset && (buffer[tokenEnd].isLetterOrDigit() || buffer[tokenEnd] == '_')) {
                    tokenEnd++
                }
                val text = buffer.subSequence(tokenStart, tokenEnd).toString()
                currentToken = when (text) {
                    "type" -> SdlTokenTypes.TYPE_KEYWORD
                    "input" -> SdlTokenTypes.INPUT_KEYWORD
                    "enum" -> SdlTokenTypes.ENUM_KEYWORD
                    "Calls" -> SdlTokenTypes.CALLS_KEYWORD
                    "String", "Int", "Float", "Boolean", "ID", "Time" -> SdlTokenTypes.SCALAR_TYPE
                    else -> SdlTokenTypes.IDENTIFIER
                }
            }

            else -> {
                tokenEnd = tokenStart + 1
                currentToken = com.intellij.psi.TokenType.BAD_CHARACTER
            }
        }
    }
}