# stroo
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fbadu%2Fstroo.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fbadu%2Fstroo?ref=badge_shield)


Ever got tired of writing a `Stringer` implementation over and over again? Wanted to implement `Marshaler` and `Unmarshaler` for `json` package?

Indeed, there are all sort of tools to generate code, but all of them are forcing you to take their own road. How about working with templates instead?
Thus you could carry your template around with your project, customizing it and having no worries that the way it is being used would change.

This tool traverses Go AST using the static checker of the [Go Tools](golang.org/x/tools). After traversal, it produces information regarding structs, functions and interfaces that are being declared in the inspected package. That information is exposed to a template that you, the developer, define. In the end, the generated code - using that template - will be written into a file of your choice. 

## How

Having the following declaration:

```go
package model
//go:generate stroo -type=SomeJsonPayload -output=model_json_gen.go -template=./../../templates/json_marshal.tmpl
//go:generate stroo -type=SomeJsonPayload -output=model_json_gen.go -template=./../../templates/json_unmarshal.tmpl
type SomeJsonPayload struct{
	Name string `json:"name"`
}
```

stroo will use the template (relative path in the example `json_marshal.tmpl` and `json_unmarshal.tmpl`) to generate the files indicated as output, in the same package with the struct declaration.

## Install

As usual, install like any other Go tool.

## Playground

Yes, there is a playground to help you build templates. By default, the first `type` definition from the source is passed to your template and you should have at least one.

### Notes

Developers can store and retrieve information inside a template : templates can store and retrieve key-values by using `{{ .Store <key> <value> }}` and retrieve them with `{{ .Retrieve <key> }}` where <key> is a `string` and <value> is `interface{}`.

This repository contains code taken (and modified) from [internal go tools](golang.org/x/tools/go/analysis/internal/checker) because the package is internal and cannot be imported. 

Thank you good authors!

### Wiki

[Here](https://github.com/badu/stroo/wiki) is the wiki.

## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fbadu%2Fstroo.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fbadu%2Fstroo?ref=badge_large)