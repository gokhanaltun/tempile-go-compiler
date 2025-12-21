# Tempile Go Compiler

`tempile-go-compiler` is a Go package that compiles Tempile templates into Go source code.
It is intended to be used internally by the Tempile CLI or other tools, **not directly by end-users**.

## Features

* Parses Tempile template files and generates readable Go code.
* Handles HTML elements, expressions, loops, and conditional logic.
* Supports merging static content for efficient output.
* Handles void HTML elements properly.
* Outputs Go code that writes to `io.Writer`, using `html.EscapeString` for safety.

## Installation

As a library:

```bash
go get github.com/gokhanaltun/tempile-go-compiler
```

> Usually imported and used by Tempile CLI internally.

## Usage

```go
import (
    "github.com/gokhanaltun/tempile-go-compiler"
)

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

* The compiler **does not perform type checking** of your template expressions.
  All type assertions (e.g., `data["items"].([]map[string]string)`) are evaluated at runtime.
* It's designed to simplify template to Go generation, leaving runtime logic and type safety to the developer.
* [CLI tool](https://github.com/gokhanaltun/tempile) is expected to import and wrap this compiler for end-user interaction.
