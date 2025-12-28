package com.borderlesshq.restgen.filetype

import com.borderlesshq.restgen.language.SdlLanguage
import com.intellij.openapi.fileTypes.LanguageFileType
import javax.swing.Icon

class SdlFileType private constructor() : LanguageFileType(SdlLanguage) {
    
    companion object {
        @JvmField
        val INSTANCE = SdlFileType()
    }
    
    override fun getName(): String = "SDL"
    
    override fun getDescription(): String = "RestGen Schema Definition Language"
    
    override fun getDefaultExtension(): String = "sdl"
    
    override fun getIcon(): Icon = SdlIcons.FILE
}
