package com.borderlesshq.restgen.language

import com.intellij.lang.Language

object SdlLanguage : Language("SDL") {
    override fun getDisplayName(): String = "RestGen SDL"
    
    override fun isCaseSensitive(): Boolean = true
}
