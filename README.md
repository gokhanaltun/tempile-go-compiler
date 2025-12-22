# Tempile Go Compiler

`tempile-go-compiler` is a Go package that compiles Tempile templates into Go source code.
It is intended for **internal use by the Tempile CLI** or other tools, **not directly by end-users**.

## Features

* Parses Tempile template files and generates readable Go code.
* Handles HTML elements, expressions, loops, and conditional logic.
* Supports merging static content for efficient output.
* Properly handles void HTML elements (e.g., `<br>`, `<img>`).
* Outputs Go code that writes to `io.Writer`, using `html.EscapeString` for safe rendering.

## Installation

As a library:

```bash
go get github.com/gokhanaltun/tempile-go-compiler
```

> Usually imported and used by the Tempile CLI internally.

## Usage

```go
import tempilegocompiler "github.com/gokhanaltun/tempile-go-compiler"

templateSource := `<div>{{ data.Name }}</div>` // example template string

options := &tempilegocompiler.CompileOptions{
    PackageName:  "mypackage",
    TemplateName: "MyTemplate",
    FileName:     "example.tempile",
    SrcPath:      "./templates",
    Imports:      []string{"fmt"},
}

compiled, err := tempilegocompiler.Compile(templateSource, options)
if err != nil {
    panic(err)
}

// `compiled` now contains valid Go code that can be written to a file
```

### Notes

* The compiler **does not perform type checking** of template expressions.
  All type assertions (e.g., `data["items"].([]map[string]string)`) are evaluated at runtime.
* HTML output and expressions are automatically escaped using `html.EscapeString` for safety.
* Designed to simplify template-to-Go generation, leaving runtime logic and type safety to the developer.
* This package is expected to be imported and wrapped by the [CLI tool](https://github.com/gokhanaltun/tempile) for end-user interaction.
