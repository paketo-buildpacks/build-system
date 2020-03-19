# `paketo-buildpacks/build-system`
The Paketo Build System Buildpack is a Cloud Native Buildpack that enables the building of JVM applications from source.

This buildpack is designed to work in collaboration with other buildpacks that provide JDKs.

## Behavior
This buildpack will participate if any of the following conditions are met

* `<APPLICATION_ROOT>/build.gradle` exists
* `<APPLICATION_ROOT>/build.gradle.kts` exists
* `<APPLICATION_ROOT>/pom.xml` exists

The buildpack will do the following for Gradle projects:

* Requests that a JDK be installed
* Links the `~/.gradle` to a layer for caching
* If `<APPLICATION_ROOT>/gradlew` exists
  * Runs `<APPLICATION_ROOT>/gradlew --no-daemon -x test build` to build the application
* If `<APPLICATION_ROOT>/gradlew` does not exist
  * Contributes Gradle to a layer with all commands on `$PATH`
  * Runs `<GRADLE_ROOT>/gradle -x test build` to build the application
* Removes the source code in `<APPLICATION_ROOT>`
* Expands `<APPLICATION_ROOT>/build/libs/*.[jw]ar` to `<APPLICATION_ROOT>`

The buildpack will do the following for Maven projects:

* Requests that a JDK be installed
* Links the `~/.m2` to a layer for caching
* If `<APPLICATION_ROOT>/mvnw` exists
  * Runs `<APPLICATION_ROOT>/mvnw -Dmaven.test.skip=true package` to build the application
* If `<APPLICATION_ROOT>/mvnw` does not exist
  * Contributes Maven to a layer with all commands on `$PATH`
  * Runs `<MAVEN_ROOT>/mvn -Dmaven.test.skip=true package` to build the application
* Removes the source code in `<APPLICATION_ROOT>`
* Expands `<APPLICATION_ROOT>/target/*.[jw]ar` to `<APPLICATION_ROOT>`

## Configuration
| Environment Variable | Description
| -------------------- | -----------
| `$BP_BUILD_ARGUMENTS` | Configure the arguments to pass to build system.  Defaults to `--no-daemon -x test build` for Gradle and `-Dmaven.test.skip=true package` for Maven.
| `$BP_BUILT_MODULE` | Configure the module to find application artifact in.  Defaults to the root module (empty).
| `$BP_BUILT_ARTIFACT` | Configure the built application artifact explicitly.  Supersedes `$BP_BUILT_MODULE`  Defaults to `build/libs/*.[jw]ar` for Gradle and `target/*.[jw]ar` for Maven.

## License
This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0
