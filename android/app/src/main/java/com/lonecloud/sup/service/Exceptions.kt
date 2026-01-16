package com.lonecloud.sup.service

class NotAuthorizedException(message: String, val user: Any? = null) : Exception(message)

fun Throwable.hasCause(causeClass: Class<out Throwable>): Boolean {
    var current: Throwable? = this
    while (current != null) {
        if (causeClass.isInstance(current)) return true
        current = current.cause
    }
    return false
}
