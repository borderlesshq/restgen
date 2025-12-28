package com.borderlesshq.restgen.reference

import com.intellij.openapi.util.TextRange
import com.intellij.psi.*
import com.intellij.psi.search.FilenameIndex
import com.intellij.psi.search.GlobalSearchScope

class SdlIncludeReference(
    element: PsiElement,
    private val fileName: String,
    textRange: TextRange
) : PsiReferenceBase<PsiElement>(element, textRange), PsiPolyVariantReference {

    override fun resolve(): PsiElement? {
        val results = multiResolve(false)
        return if (results.size == 1) results[0].element else null
    }

    override fun multiResolve(incompleteCode: Boolean): Array<ResolveResult> {
        val project = element.project
        val scope = GlobalSearchScope.projectScope(project)

        // Search for files with the given name
        val files = FilenameIndex.getFilesByName(project, fileName, scope)

        return files.map { PsiElementResolveResult(it) }.toTypedArray()
    }

    override fun getVariants(): Array<Any> = emptyArray()
}