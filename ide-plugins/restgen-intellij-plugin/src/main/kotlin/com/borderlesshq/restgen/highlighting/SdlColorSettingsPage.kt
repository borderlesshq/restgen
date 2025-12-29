package com.borderlesshq.restgen.highlighting

import com.borderlesshq.restgen.filetype.SdlIcons
import com.intellij.openapi.editor.colors.TextAttributesKey
import com.intellij.openapi.fileTypes.SyntaxHighlighter
import com.intellij.openapi.options.colors.AttributesDescriptor
import com.intellij.openapi.options.colors.ColorDescriptor
import com.intellij.openapi.options.colors.ColorSettingsPage
import javax.swing.Icon

class SdlColorSettingsPage : ColorSettingsPage {

    override fun getIcon(): Icon = SdlIcons.FILE

    override fun getHighlighter(): SyntaxHighlighter = SdlSyntaxHighlighter()

    override fun getDemoText(): String = """
# @base("/v1/contacts")
# @models("github.com/yourorg/yourapp/models")
# @include("common.sdl")

enum ContactStatus {
    active
    inactive
    pending
}

type Contact {
    id: ID!
    name: String!
    email: String
    status: ContactStatus!
    createdAt: Time!
}

input CreateContactInput {
    name: String!
    email: String!
}

type Calls {
    // Get a contact by ID
    getContact(id: ID!): Contact @get("/{id}")
    
    // Create a new contact
    createContact(input: CreateContactInput!): Contact! @post("/")
    
    // List all contacts with pagination
    listContacts(page: Int, limit: Int): [Contact!]! @get("/")
}
    """.trimIndent()

    override fun getAdditionalHighlightingTagToDescriptorMap(): MutableMap<String, TextAttributesKey>? = null

    override fun getAttributeDescriptors(): Array<AttributesDescriptor> = DESCRIPTORS

    override fun getColorDescriptors(): Array<ColorDescriptor> = ColorDescriptor.EMPTY_ARRAY

    override fun getDisplayName(): String = "RestGen SDL"
}

private val DESCRIPTORS = arrayOf(
    AttributesDescriptor("Keyword", SdlSyntaxHighlighter.KEYWORD),
    AttributesDescriptor("Scalar type", SdlSyntaxHighlighter.SCALAR),
    AttributesDescriptor("Directive", SdlSyntaxHighlighter.DIRECTIVE),
    AttributesDescriptor("HTTP directive", SdlSyntaxHighlighter.HTTP_DIRECTIVE),
    AttributesDescriptor("Identifier", SdlSyntaxHighlighter.IDENTIFIER_KEY),
    AttributesDescriptor("String", SdlSyntaxHighlighter.STRING_KEY),
    AttributesDescriptor("Number", SdlSyntaxHighlighter.NUMBER_KEY),
    AttributesDescriptor("Comment", SdlSyntaxHighlighter.COMMENT_KEY),
    AttributesDescriptor("Braces", SdlSyntaxHighlighter.BRACES),
    AttributesDescriptor("Brackets", SdlSyntaxHighlighter.BRACKETS),
    AttributesDescriptor("Parentheses", SdlSyntaxHighlighter.PARENTHESES),
    AttributesDescriptor("Operator", SdlSyntaxHighlighter.OPERATOR),
    AttributesDescriptor("Bad character", SdlSyntaxHighlighter.BAD_CHARACTER_KEY),
    AttributesDescriptor("Directive comment", SdlSyntaxHighlighter.DIRECTIVE_COMMENT),
    AttributesDescriptor("Include link", SdlSyntaxHighlighter.INCLUDE_LINK),
    AttributesDescriptor("Unresolved include", SdlSyntaxHighlighter.INCLUDE_LINK_UNRESOLVED)
)