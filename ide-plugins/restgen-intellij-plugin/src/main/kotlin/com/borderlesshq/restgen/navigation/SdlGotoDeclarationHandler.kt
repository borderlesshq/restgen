package com.borderlesshq.restgen.navigation

import com.borderlesshq.restgen.filetype.SdlFileType
import com.intellij.codeInsight.navigation.actions.GotoDeclarationHandler
import com.intellij.openapi.editor.Editor
import com.intellij.psi.PsiElement
import com.intellij.psi.search.FilenameIndex
import com.intellij.psi.search.GlobalSearchScope

class SdlGotoDeclarationHandler : GotoDeclarationHandler {

    private val includePattern = """@include\s*\(\s*"([^"]+)"\s*\)""".toRegex()

    override fun getGotoDeclarationTargets(
        sourceElement: PsiElement?,
        offset: Int,
        editor: Editor?
    ): Array<PsiElement>? {
        if (sourceElement == null) return null

        // Get the line text to find @include pattern
        val document = editor?.document ?: return null
        val lineNumber = document.getLineNumber(offset)
        val lineStart = document.getLineStartOffset(lineNumber)
        val lineEnd = document.getLineEndOffset(lineNumber)
        val lineText = document.getText(com.intellij.openapi.util.TextRange(lineStart, lineEnd))

        // Check if we're on an @include line
        val match = includePattern.find(lineText) ?: return null
        val fileName = match.groupValues[1]

        // Check if cursor is within the filename
        val fileNameStartInLine = lineText.indexOf('"', match.range.first) + 1
        val fileNameEndInLine = fileNameStartInLine + fileName.length
        val cursorPosInLine = offset - lineStart

        if (cursorPosInLine < fileNameStartInLine || cursorPosInLine > fileNameEndInLine) {
            return null
        }

        // Find and return the target file
        val project = sourceElement.project
        val scope = GlobalSearchScope.projectScope(project)
        val files = FilenameIndex.getFilesByName(project, fileName, scope)
            .filter { it.fileType == SdlFileType.INSTANCE }

        return if (files.isNotEmpty()) files.toTypedArray() else null
    }
}