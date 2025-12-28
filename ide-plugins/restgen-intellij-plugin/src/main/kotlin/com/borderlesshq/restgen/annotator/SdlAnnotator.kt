package com.borderlesshq.restgen.annotator

import com.borderlesshq.restgen.filetype.SdlFileType
import com.borderlesshq.restgen.highlighting.SdlSyntaxHighlighter
import com.intellij.lang.annotation.AnnotationHolder
import com.intellij.lang.annotation.Annotator
import com.intellij.lang.annotation.HighlightSeverity
import com.intellij.openapi.editor.colors.TextAttributesKey
import com.intellij.openapi.project.Project
import com.intellij.openapi.util.TextRange
import com.intellij.psi.PsiElement
import com.intellij.psi.search.FilenameIndex
import com.intellij.psi.search.GlobalSearchScope

class SdlAnnotator : Annotator {

    // Pattern to match @include("filename.sdl") in comments
    private val includePattern = """#\s*@include\s*\(\s*"([^"]+)"\s*\)""".toRegex()

    // Pattern to match @base("path") in comments
    private val basePattern = """#\s*@base\s*\(\s*"([^"]+)"\s*\)""".toRegex()

    // Pattern to match @models("package") in comments
    private val modelsPattern = """#\s*@models\s*\(\s*"([^"]+)"\s*\)""".toRegex()

    override fun annotate(element: PsiElement, holder: AnnotationHolder) {
        val text = element.text
        val startOffset = element.textRange.startOffset

        // Highlight @include directives and make them navigable
        includePattern.findAll(text).forEach { match ->
            val fullMatch = match.range
            val fileName = match.groupValues[1]

            // Highlight the whole directive
            highlightRange(
                holder,
                TextRange(startOffset + fullMatch.first, startOffset + fullMatch.last + 1),
                SdlSyntaxHighlighter.DIRECTIVE_COMMENT
            )

            // Find the filename part and make it a link
            val fileNameStart = match.value.indexOf('"') + 1
            val fileNameEnd = fileNameStart + fileName.length
            val fileRange = TextRange(
                startOffset + fullMatch.first + fileNameStart,
                startOffset + fullMatch.first + fileNameEnd
            )

            // Check if file exists and create navigation
            val targetFile = findSdlFile(element.project, fileName)
            if (targetFile != null) {
                holder.newAnnotation(HighlightSeverity.INFORMATION, "Navigate to $fileName")
                    .range(fileRange)
                    .textAttributes(SdlSyntaxHighlighter.INCLUDE_LINK)
                    .create()
            } else {
                holder.newAnnotation(HighlightSeverity.WARNING, "File not found: $fileName")
                    .range(fileRange)
                    .textAttributes(SdlSyntaxHighlighter.INCLUDE_LINK_UNRESOLVED)
                    .create()
            }
        }

        // Highlight @base directives
        basePattern.findAll(text).forEach { match ->
            val fullMatch = match.range
            highlightRange(
                holder,
                TextRange(startOffset + fullMatch.first, startOffset + fullMatch.last + 1),
                SdlSyntaxHighlighter.DIRECTIVE_COMMENT
            )
        }

        // Highlight @models directives
        modelsPattern.findAll(text).forEach { match ->
            val fullMatch = match.range
            highlightRange(
                holder,
                TextRange(startOffset + fullMatch.first, startOffset + fullMatch.last + 1),
                SdlSyntaxHighlighter.DIRECTIVE_COMMENT
            )
        }
    }

    private fun highlightRange(
        holder: AnnotationHolder,
        range: TextRange,
        textAttributes: TextAttributesKey
    ) {
        holder.newSilentAnnotation(HighlightSeverity.INFORMATION)
            .range(range)
            .textAttributes(textAttributes)
            .create()
    }

    private fun findSdlFile(project: Project, fileName: String): PsiElement? {
        val scope = GlobalSearchScope.projectScope(project)
        val files = FilenameIndex.getFilesByName(project, fileName, scope)
        return files.firstOrNull { it.fileType == SdlFileType.INSTANCE }
    }
}