package com.borderlesshq.restgen.reference

import com.borderlesshq.restgen.lexer.SdlTokenTypes
import com.intellij.openapi.util.TextRange
import com.intellij.patterns.PlatformPatterns
import com.intellij.psi.*
import com.intellij.util.ProcessingContext

class SdlReferenceContributor : PsiReferenceContributor() {

    override fun registerReferenceProviders(registrar: PsiReferenceRegistrar) {
        registrar.registerReferenceProvider(
            PlatformPatterns.psiElement(),
            object : PsiReferenceProvider() {
                override fun getReferencesByElement(
                    element: PsiElement,
                    context: ProcessingContext
                ): Array<PsiReference> {
                    val node = element.node ?: return PsiReference.EMPTY_ARRAY

                    // Check if this is a directive comment with @include
                    if (node.elementType == SdlTokenTypes.DIRECTIVE_COMMENT_INCLUDE) {
                        val text = element.text

                        // Extract filename from @include("filename.sdl")
                        val regex = """@include\s*\(\s*"([^"]+)"\s*\)""".toRegex()
                        val match = regex.find(text)

                        if (match != null) {
                            val fileName = match.groupValues[1]
                            val startOffset = match.range.first + text.indexOf('"') + 1
                            val endOffset = startOffset + fileName.length

                            return arrayOf(
                                SdlIncludeReference(
                                    element,
                                    fileName,
                                    TextRange(startOffset, endOffset)
                                )
                            )
                        }
                    }

                    return PsiReference.EMPTY_ARRAY
                }
            }
        )
    }
}