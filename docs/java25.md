# Java 25+ target

Go+ can elaborate one source module for Go and for a deterministic Java 25
source/JAR pipeline. Go is still the default and complete backend. The Java
backend currently accepts a deliberately checked portable subset: unsupported
Go constructs are source-positioned compiler diagnostics, never silently
translated with different semantics.

The repository's `goplus.toml` makes the Go+ compiler itself the forcing case.
That self-hosting target is not green yet: the compiler uses Go reflection,
filesystem/process APIs, and standard-library packages outside the portable
subset. The working contract is the subset and runtime described below, which
is covered by end-to-end JDK 25 compile/run tests.

## Requirements and commands

Install a complete JDK 25 or newer and either put `javac` and `java` on `PATH`
or set `GOPLUS_JAVA_HOME`. `GOPLUS_JAVA_HOME` takes precedence over
`JAVA_HOME`, asdf Temurin installations, and `PATH`.

```sh
# Emit deterministic Java and runtime sources.
go tool goplus gen --target java ./...

# Verify checked-in generated sources without writing.
go tool goplus gen --target java --check ./...

# Compile with javac --release 25 and create deterministic JARs.
go tool goplus build --target java ./...

# Compile and run Go-style TestXxx functions.
go tool goplus test --target java ./...

# Run the configured app; arguments after -- go to main.
go tool goplus run --target java ./cmd/app -- arg1 arg2

# Build, sign, validate, and upload a library to Maven Central.
go tool goplus publish --target java --automatic ./...
```

`--target` is repeatable for `gen`, `build`, `test`, and `vet`. `run` requires
exactly one target. With no flag, `default_targets` is used; without a
`goplus.toml`, the default remains `["go"]`.

Java compilation is intentionally hermetic with respect to Java dependency
paths: the tool removes ambient `CLASSPATH` and JDK option-injection variables
and supplies only configured paths. It invokes:

```text
javac --release 25 -encoding UTF-8 -proc:none -Xlint:all -Werror
```

## `goplus.toml`

The file is searched upward to the nearest `go.mod`, has a required version,
and rejects unknown tables and keys.

```toml
schema_version = 1
default_targets = ["go"]

[targets.go]

[targets.java]
release = 25
kind = "library" # or "app"
source_dir = "gen/java"
class_dir = ".goplus/build/java/classes"
jar = "dist/project.jar"
runtime_jar = ".goplus/build/java/goplus-runtime-abi1.jar"
package_prefix = "com.example.project"
module_name = "com.example.project"
main_package = "example.com/project/cmd/project"
classpath_files = ["java/classpath.txt"]
modulepath_files = ["java/modulepath.txt"]
bundle = false
strong_module = false

[targets.java.maven]
group_id = "com.example"
artifact_id = "project"
version = "1.0.0"
name = "Example Project"
description = "An example Go+ library for Java 25+"
url = "https://example.com/project"
license_name = "MIT License"
license_url = "https://opensource.org/license/mit"
developer_id = "example"
developer_name = "Example Maintainer"
developer_email = "opensource@example.com"
developer_url = "https://example.com"
scm_url = "https://github.com/example/project"
scm_connection = "scm:git:https://github.com/example/project.git"
scm_developer_connection = "scm:git:ssh://git@github.com/example/project.git"
signing_key = "/secure/path/maven-central-private.asc" # optional
bundle = "dist/central/project-1.0.0-bundle.zip"
```

Output paths must be relative and remain inside the module root. Each path
manifest is UTF-8, one JAR or class directory per line; paths are relative to
the module root unless absolute, blank lines and `#` comments are ignored, and
every entry must exist. Absolute entries are intended for path files generated
by a build system; checked-in manifests should use relative entries.
`package_prefix` defaults to the reversed module domain
(`example.com/acme` becomes `com.example.acme`). `module_name` defaults to that
package prefix.

## Maven Central publication

Maven Central publication is library-only and currently requires `bundle =
true`, so the versioned Go+ runtime ABI is included in the primary artifact and
Java consumers need one dependency. On first publication Go+ creates a 3072-bit
OpenPGP identity from the configured developer name/email, stores it with mode
`0600` under the user configuration directory, and publishes only its public
key to a Central-supported keyserver. `signing_key` or
`GOPLUS_MAVEN_SIGNING_KEY` can select another persistent location.

Portal user-token credentials are discovered in this order: environment,
`maven_settings.xml` in the project or workspace root, then
`~/.m2/settings.xml`. A conventional settings entry is sufficient:

```xml
<settings><servers><server>
  <id>central</id>
  <username>token username</username>
  <password>token password</password>
</server></servers></settings>
```

Environment credentials remain available for CI:

```sh
export MAVEN_CENTRAL_USERNAME='token username'
export MAVEN_CENTRAL_PASSWORD='token password'

# Creates and internally inspects the complete signed bundle without uploading.
goplus publish --target java --bundle-only ./...

# Publishes the public signing key, uploads, validates, and publishes.
goplus publish --target java ./...
```

`--automatic=true` is the default. Set `--automatic=false` to stop successfully
at `VALIDATED` for manual review. The publisher never copies Portal credentials
into the project or bundle. Signature creation time comes from
`SOURCE_DATE_EPOCH`, or otherwise from the current Git commit, and signature
randomization is disabled for reproducibility: the same artifacts, metadata,
key, and epoch produce the same bundle bytes.
The source JAR contains the generated Java plus bundled runtime sources. Until
the emitter carries JavaDoc comments through the complete portable subset, the
required Javadoc JAR contains a transparent documentation pointer to the
project URL rather than fabricated API prose.

Generated project sources land under `source_dir`. Runtime ABI sources land
under `.goplus/build/java/runtime-src`; classes and runtime classes are build
outputs. `goplus-artifacts.json` is the ownership and reproducibility manifest.
Generation removes only stale paths owned by the prior manifest.
Runtime sources under `.goplus` are ephemeral and regenerated for a build;
`gen --check` verifies the project sources and ownership manifest without
requiring ignored runtime sources to be present in a fresh checkout.

The default thin JAR depends on `goplus-runtime-abi1.jar`. `bundle = true`
copies runtime classes into the project JAR. `strong_module = true` emits
`module-info.java`; a bundled strong module incorporates the ABI package into
the project module instead of requiring the separate runtime module. Otherwise
both JARs receive stable
`Automatic-Module-Name` manifest entries.

## Target selection

The front end runs independently for each target. Java sees these build tags:

```text
goplus_java, java25, jvm64
```

Target-specific declarations can therefore use normal build constraints:

```go
//go:build goplus_java

package demo
```

The public `goforge.dev/goplus/compiler` package is filesystem-write-free and
returns versioned artifact sets:

```go
result, err := compiler.Compile(ctx, compiler.Request{
    Dir:      moduleRoot,
    Patterns: []string{"./..."},
    Targets:  []compiler.Target{compiler.TargetGo, compiler.TargetJava},
    Java: compiler.JavaOptions{
        Release: 25,
        Kind: "library",
        SourceDir: "gen/java",
    },
})
```

Embedders own materialization and caching. Java dependency paths supplied to
the API are explicit resolved JAR/class-directory paths.

## Runtime ABI and semantic mapping

The generated source manifest is `goplus.java.artifacts/v1`; the runtime is ABI
1 in Java package `dev.goforge.goplus.runtime`.

| Go/Go+ value | Java representation and contract |
|---|---|
| `bool`, signed fixed-width integers, floats | corresponding JVM primitive; boxed in generic positions |
| `int`, `uint`, `uintptr` | 64-bit because the Java target declares `jvm64` |
| unsigned arithmetic | raw JVM bits plus unsigned compare/divide/remainder/shift helpers |
| `string` | immutable byte-backed `GpString`; length/index/slice use Go bytes, not UTF-16 code units |
| slice/array | `GpSlice<T>` header with shared backing storage, length, capacity, append growth, and typed zero values |
| map | `GpMap<K,V>` with Go zero-on-miss, comma-ok lookup, nil-map assignment panic, delete, and clear |
| named struct | generated copyable Java class; value receivers operate on a copy and pointer receivers mutate identity |
| pointer to scalar | explicit `GpRef<T>` cell; pointer-to-struct uses object identity |
| channel | `GpChan<T>` with buffered/unbuffered FIFO behavior, close, range, and select coordination |
| goroutine | Java virtual thread started by the runtime |
| two/three results | `GpTuple2` / `GpTuple3` records |

Go string ↔ `java.lang.String` conversion is explicit UTF-8 at a Java boundary.
A nullable Java string cannot bind directly to a Go string. Map iteration is
not promised to reproduce a particular Go runtime order.

## Java interop

### Direct type imports and construction

Java-only source may import one type or a package namespace. Constructors use
the explicit `new(JavaType, args...)` Go+ form:

```go
//go:build goplus_java

package demo

import ArrayList "java:type/java.util.ArrayList"

func create() ArrayList[string] {
    return new(ArrayList[string], 16)
}
```

Use `java:package/java.util` when several types need a `util.Type` namespace.
Direct imports currently provide Java type references and constructors. They do
not yet synthesize arbitrary overloaded member calls or Java subtype
assignability in the Go+ type checker.

### Exact method bindings

A bodyless, Java-tagged package function can bind a constructor, static method,
or virtual method with an exact JVM descriptor:

```go
//go:build goplus_java

package demo

//goplus:java kind=static owner=java.lang.Math member=max descriptor="(II)I" null=nonnull throws=none
func JavaMax(a, b int32) int32

//goplus:java kind=virtual owner=java.lang.String member=concat descriptor="(Ljava/lang/String;)Ljava/lang/String;" null=nonnull throws=none string=java
func JavaConcat(receiver, suffix string) string
```

Fields are:

- `kind`: `static`, `virtual`, or `constructor`;
- `owner`: binary Java class name;
- `member`: method name (omitted for a constructor);
- `descriptor`: exact JVM method descriptor;
- `null`: `nonnull` or `nullable`;
- `throws`: `none`, `panic`, or `result`;
- `string`: `go` or `java`, with `java` enabling explicit UTF-8 conversion.

A constructor descriptor ends in `V`, as required by the JVM, while its Go
binding returns the imported Java type (usually a pointer-shaped Go view).
With `throws=result`, the Go binding has two results and the second must be the
built-in `error`; Java exceptions are exposed through that error without
crossing the boundary as an unchecked host exception.

Bindings are validated before emission against JDK `ct.sym` (the same public
API signatures used by `javac --release`), available JMODs, and configured
classpath/module-path entries. Multi-release JARs select the highest entry no
newer than the configured release. Classfiles newer than the configured Java
release are rejected rather than guessed at (Java 25 is classfile major 69).

## Current compatibility ledger

Implemented and exercised on JDK 25:

- package functions, globals, constants, `init`, named structs, package-local basic
  interfaces, methods, generics, variadics, and two/three-value returns;
- arithmetic and comparisons, byte-correct strings, struct/slice/map/array
  literals, indexing and two-index slicing;
- assignments, `if`, classic and range `for`, expression switch, `break`,
  `continue`, `return`, `defer`, `go`, channel send/receive/close/range, and
  `select` including `default`;
- `len`, `cap`, `append`, `make`, `new`, `copy`, `delete`, `clear`, `close`,
  `panic`, `print`, `println`, `min`, and `max`;
- `TestXxx(*testing.T)` discovery, nested `Run`, logging, fatal failures, and
  process exit status;
- small adapters for `fmt.Print/Println`, selected `strings` operations, and
  `strconv.Itoa`;
- deterministic source/JAR bytes, thin or bundled runtime, automatic modules,
  and strong JPMS modules.

Diagnosed as unsupported today:

- the full Go standard library, reflection, cgo, and native Go package
  implementations that have no Java-target adapter;
- type switches, `recover`, three-index slices, array ellipsis, keyed
  slice/array literals, anonymous struct types, and address-taking of escaping
  local variables;
- function values with more than three parameters and results outside the
  current two/three-value tuple ABI;
- deferred mutation of named return values and the full edge-case interaction
  among nested panics, defers, and recovery;
- a general overload-resolution engine for direct Java imports;
- automatic `requires` discovery for external dependencies in a strong JPMS
  project module;
- self-hosting the Go+ compiler, native-image production, and automatic
  Maven/Gradle dependency resolution (publication itself is supported).

This ledger is intentionally explicit: a successful Go build does not imply a
successful Java build. Run `goplus gen --target java --check` and the JDK build
in CI for every module that publishes Java artifacts.
