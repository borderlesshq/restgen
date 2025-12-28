package com.borderlesshq.restgen.highlighting

import com.borderlesshq.restgen.lexer.SdlLexer
import com.borderlesshq.restgen.lexer.SdlTokenTypes
import com.intellij.lexer.Lexer
import com.intellij.openapi.editor.DefaultLanguageHighlighterColors
import com.intellij.openapi.editor.HighlighterColors
import com.intellij.openapi.editor.colors.TextAttributesKey
import com.intellij.openapi.editor.colors.TextAttributesKey.createTextAttributesKey
import com.intellij.openapi.fileTypes.SyntaxHighlighterBase
import com.intellij.psi.TokenType
import com.intellij.psi.tree.IElementType

class SdlSyntaxHighlighter : SyntaxHighlighterBase() {
    
    companion object {
        // Keywords
        val KEYWORD = createTextAttributesKey("SDL_KEYWORD", DefaultLanguageHighlighterColors.KEYWORD)
        
        // Scalars
        val SCALAR = createTextAttributesKey("SDL_SCALAR", DefaultLanguageHighlighterColors.PREDEFINED_SYMBOL)
        
        // Directives
        val DIRECTIVE = createTextAttributesKey("SDL_DIRECTIVE", DefaultLanguageHighlighterColors.METADATA)
        val HTTP_DIRECTIVE = createTextAttributesKey("SDL_HTTP_DIRECTIVE", DefaultLanguageHighlighterColors.STATIC_METHOD)
        
        // Identifiers
        val IDENTIFIER_KEY = createTextAttributesKey("SDL_IDENTIFIER", DefaultLanguageHighlighterColors.IDENTIFIER)
        
        // Literals
        val STRING_KEY = createTextAttributesKey("SDL_STRING", DefaultLanguageHighlighterColors.STRING)
        val NUMBER_KEY = createTextAttributesKey("SDL_NUMBER", DefaultLanguageHighlighterColors.NUMBER)
        
        // Comments
        val COMMENT_KEY = createTextAttributesKey("SDL_COMMENT", DefaultLanguageHighlighterColors.LINE_COMMENT)
        
        // Punctuation
        val BRACES = createTextAttributesKey("SDL_BRACES", DefaultLanguageHighlighterColors.BRACES)
        val BRACKETS = createTextAttributesKey("SDL_BRACKETS", DefaultLanguageHighlighterColors.BRACKETS)
        val PARENTHESES = createTextAttributesKey("SDL_PARENTHESES", DefaultLanguageHighlighterColors.PARENTHESES)
        val OPERATOR = createTextAttributesKey("SDL_OPERATOR", DefaultLanguageHighlighterColors.OPERATION_SIGN)
        
        // Bad character
        val BAD_CHARACTER_KEY = createTextAttributesKey("SDL_BAD_CHARACTER", HighlighterColors.BAD_CHARACTER)
        
        // Directive comments (# @base, # @models, # @include)
        val DIRECTIVE_COMMENT = createTextAttributesKey("SDL_DIRECTIVE_COMMENT", DefaultLanguageHighlighterColors.METADATA)

        // Include link (clickable file reference)
        val INCLUDE_LINK = createTextAttributesKey("SDL_INCLUDE_LINK", DefaultLanguageHighlighterColors.STRING)

        // Unresolved include link
        val INCLUDE_LINK_UNRESOLVED = createTextAttributesKey("SDL_INCLUDE_LINK_UNRESOLVED", DefaultLanguageHighlighterColors.STRING)

        
        // Key arrays
        private val KEYWORD_KEYS = arrayOf(KEYWORD)
        private val SCALAR_KEYS = arrayOf(SCALAR)
        private val DIRECTIVE_KEYS = arrayOf(DIRECTIVE)
        private val HTTP_DIRECTIVE_KEYS = arrayOf(HTTP_DIRECTIVE)
        private val IDENTIFIER_KEYS = arrayOf(IDENTIFIER_KEY)
        private val STRING_KEYS = arrayOf(STRING_KEY)
        private val NUMBER_KEYS = arrayOf(NUMBER_KEY)
        private val COMMENT_KEYS = arrayOf(COMMENT_KEY)
        private val BRACES_KEYS = arrayOf(BRACES)
        private val BRACKETS_KEYS = arrayOf(BRACKETS)
        private val PARENTHESES_KEYS = arrayOf(PARENTHESES)
        private val OPERATOR_KEYS = arrayOf(OPERATOR)
        private val BAD_CHARACTER_KEYS = arrayOf(BAD_CHARACTER_KEY)
        private val EMPTY_KEYS = emptyArray<TextAttributesKey>()
        private val DIRECTIVE_COMMENT_KEYS = arrayOf(DIRECTIVE_COMMENT)
    }
    
    override fun getHighlightingLexer(): Lexer = SdlLexer()
    
    override fun getTokenHighlights(tokenType: IElementType): Array<TextAttributesKey> {
        return when (tokenType) {
            // Keywords
            SdlTokenTypes.TYPE_KEYWORD, SdlTokenTypes.INPUT_KEYWORD, 
            SdlTokenTypes.ENUM_KEYWORD, SdlTokenTypes.CALLS_KEYWORD -> KEYWORD_KEYS
            
            // Scalars
            SdlTokenTypes.SCALAR_TYPE -> SCALAR_KEYS
            
            // HTTP Directives
            SdlTokenTypes.DIRECTIVE_GET, SdlTokenTypes.DIRECTIVE_POST, 
            SdlTokenTypes.DIRECTIVE_PUT, SdlTokenTypes.DIRECTIVE_PATCH, 
            SdlTokenTypes.DIRECTIVE_DELETE -> HTTP_DIRECTIVE_KEYS
            
            // Other Directives
            SdlTokenTypes.DIRECTIVE_BASE, SdlTokenTypes.DIRECTIVE_MODELS,
            SdlTokenTypes.DIRECTIVE_INCLUDE, SdlTokenTypes.DIRECTIVE_OTHER -> DIRECTIVE_KEYS
            
            // Identifiers
            SdlTokenTypes.IDENTIFIER -> IDENTIFIER_KEYS
            
            // Literals
            SdlTokenTypes.STRING -> STRING_KEYS
            SdlTokenTypes.NUMBER -> NUMBER_KEYS

            // Directive comments
            SdlTokenTypes.DIRECTIVE_COMMENT_INCLUDE,
            SdlTokenTypes.DIRECTIVE_COMMENT_BASE,
            SdlTokenTypes.DIRECTIVE_COMMENT_MODELS -> DIRECTIVE_COMMENT_KEYS
            
            // Comments
            SdlTokenTypes.COMMENT -> COMMENT_KEYS
            
            // Punctuation
            SdlTokenTypes.LBRACE, SdlTokenTypes.RBRACE -> BRACES_KEYS
            SdlTokenTypes.LBRACKET, SdlTokenTypes.RBRACKET -> BRACKETS_KEYS
            SdlTokenTypes.LPAREN, SdlTokenTypes.RPAREN -> PARENTHESES_KEYS
            SdlTokenTypes.COLON, SdlTokenTypes.COMMA, 
            SdlTokenTypes.EXCLAIM, SdlTokenTypes.DOT -> OPERATOR_KEYS
            
            // Bad character
            TokenType.BAD_CHARACTER -> BAD_CHARACTER_KEYS
            
            else -> EMPTY_KEYS
        }
    }
}
