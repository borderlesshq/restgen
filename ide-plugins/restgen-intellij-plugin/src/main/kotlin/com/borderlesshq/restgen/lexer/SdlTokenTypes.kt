package com.borderlesshq.restgen.lexer

import com.borderlesshq.restgen.language.SdlLanguage
import com.intellij.psi.tree.IElementType

object SdlTokenTypes {
    // Keywords
    @JvmField val TYPE_KEYWORD = IElementType("TYPE_KEYWORD", SdlLanguage)
    @JvmField val INPUT_KEYWORD = IElementType("INPUT_KEYWORD", SdlLanguage)
    @JvmField val ENUM_KEYWORD = IElementType("ENUM_KEYWORD", SdlLanguage)
    @JvmField val CALLS_KEYWORD = IElementType("CALLS_KEYWORD", SdlLanguage)
    
    // Scalar types
    @JvmField val SCALAR_TYPE = IElementType("SCALAR_TYPE", SdlLanguage)
    
    // HTTP Directives
    @JvmField val DIRECTIVE_GET = IElementType("DIRECTIVE_GET", SdlLanguage)
    @JvmField val DIRECTIVE_POST = IElementType("DIRECTIVE_POST", SdlLanguage)
    @JvmField val DIRECTIVE_PUT = IElementType("DIRECTIVE_PUT", SdlLanguage)
    @JvmField val DIRECTIVE_PATCH = IElementType("DIRECTIVE_PATCH", SdlLanguage)
    @JvmField val DIRECTIVE_DELETE = IElementType("DIRECTIVE_DELETE", SdlLanguage)
    
    // Config Directives
    @JvmField val DIRECTIVE_BASE = IElementType("DIRECTIVE_BASE", SdlLanguage)
    @JvmField val DIRECTIVE_MODELS = IElementType("DIRECTIVE_MODELS", SdlLanguage)
    @JvmField val DIRECTIVE_INCLUDE = IElementType("DIRECTIVE_INCLUDE", SdlLanguage)
    @JvmField val DIRECTIVE_OTHER = IElementType("DIRECTIVE_OTHER", SdlLanguage)
    
    // Punctuation
    @JvmField val LBRACE = IElementType("LBRACE", SdlLanguage)
    @JvmField val RBRACE = IElementType("RBRACE", SdlLanguage)
    @JvmField val LPAREN = IElementType("LPAREN", SdlLanguage)
    @JvmField val RPAREN = IElementType("RPAREN", SdlLanguage)
    @JvmField val LBRACKET = IElementType("LBRACKET", SdlLanguage)
    @JvmField val RBRACKET = IElementType("RBRACKET", SdlLanguage)
    @JvmField val COLON = IElementType("COLON", SdlLanguage)
    @JvmField val COMMA = IElementType("COMMA", SdlLanguage)
    @JvmField val EXCLAIM = IElementType("EXCLAIM", SdlLanguage)
    @JvmField val DOT = IElementType("DOT", SdlLanguage)
    
    // Literals
    @JvmField val STRING = IElementType("STRING", SdlLanguage)
    @JvmField val NUMBER = IElementType("NUMBER", SdlLanguage)
    
    // Identifier
    @JvmField val IDENTIFIER = IElementType("IDENTIFIER", SdlLanguage)
    
    // Comment
    @JvmField val COMMENT = IElementType("COMMENT", SdlLanguage)

    // Directive comments (# @include, # @base, # @models)
    @JvmField val DIRECTIVE_COMMENT_INCLUDE = IElementType("DIRECTIVE_COMMENT_INCLUDE", SdlLanguage)
    @JvmField val DIRECTIVE_COMMENT_BASE = IElementType("DIRECTIVE_COMMENT_BASE", SdlLanguage)
    @JvmField val DIRECTIVE_COMMENT_MODELS = IElementType("DIRECTIVE_COMMENT_MODELS", SdlLanguage)
}
