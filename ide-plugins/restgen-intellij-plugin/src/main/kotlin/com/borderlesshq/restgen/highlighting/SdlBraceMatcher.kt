package com.borderlesshq.restgen.highlighting

import com.borderlesshq.restgen.lexer.SdlTokenTypes
import com.intellij.lang.BracePair
import com.intellij.lang.PairedBraceMatcher
import com.intellij.psi.PsiFile
import com.intellij.psi.tree.IElementType

class SdlBraceMatcher : PairedBraceMatcher {

    override fun getPairs(): Array<BracePair> = PAIRS
    
    override fun isPairedBracesAllowedBeforeType(lbraceType: IElementType, contextType: IElementType?): Boolean = true
    
    override fun getCodeConstructStart(file: PsiFile?, openingBraceOffset: Int): Int = openingBraceOffset
}

private val PAIRS = arrayOf(
    BracePair(SdlTokenTypes.LBRACE, SdlTokenTypes.RBRACE, true),
    BracePair(SdlTokenTypes.LBRACKET, SdlTokenTypes.RBRACKET, false),
    BracePair(SdlTokenTypes.LPAREN, SdlTokenTypes.RPAREN, false)
)