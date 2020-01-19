// This is the default template to help you build with Stroo
// Modify it as you wish
{{- range .Packages -}}
// Interfaces
{{ range .Interfaces }}
type {{.Name}} interface {
	{{- range .Methods }}
	{{ .Name }}Func func({{ .Args | ArgList }}) {{ .ReturnArgs | ArgListTypes }}
	{{- end }}
}
{{ end }}

// Structs
{{ range .Structs }}
type {{ .Name }} struct {
	{{- range .Fields }}
        {{ .Name }} {{ .Type.Name }}
	{{- end }}
}
{{ end }}
{{- end }}
