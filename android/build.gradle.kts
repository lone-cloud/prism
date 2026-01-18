plugins {
    id("com.android.application") version "8.9.1" apply false
    id("org.jetbrains.kotlin.android") version "2.1.0" apply false
}

subprojects {
    configurations.all {
        resolutionStrategy {
            activateDependencyLocking()
            
            // Fix Room 2.8.x kotlinx-serialization mismatch
            // https://issuetracker.google.com/issues/447154195
            eachDependency {
                if (requested.group == "org.jetbrains.kotlinx" &&
                    requested.name.startsWith("kotlinx-serialization-")
                ) {
                    useVersion("1.8.0")
                }
            }
        }
    }
    
    dependencyLocking {
        lockAllConfigurations()
    }
}
