{{- .AddToImports "bytes" -}}
{{- .AddToImports "strconv" -}}
{{- .AddToImports "fmt" -}}
{{- .AddToImports "strings"}}
package {{.PackageInfo.Name}}
{{ if .Imports }}
import (
	{{range .Imports -}}
	"{{ . }}"
	{{end -}}
)
{{end}}
{{ define "Stringer" }}
	{{ with .T -}}
		{{- if .IsArray -}}
			// Stringer implementation for {{ .Name }}
			func (st {{ .Name }}) String() string {
				var sb strings.Builder
				for _, el := range st {
					sb.WriteString("{{.Kind}}:\n"+fmt.Sprintf("%s", el))
					{{- if not ($.D.HasInStore .Kind) }}
						{{- if $.D.GenerateAndStore .Kind -}}{{end -}}
					{{end -}}
				}
				return sb.String()
			}
		{{- else }}
			// Stringer implementation for {{ .Kind}}
			func (st {{ .Kind }}) String() string {
				var sb strings.Builder
				{{- $package := .Package }}
				{{- if sort .Fields -}}{{- end -}}
				{{- range .Fields }}
					{{- if .IsExported }}

						{{- if .IsImported -}}
							// IMPORTED `{{.Package}}.{{.Name}}` `{{.PackagePath}}` and {{.IsBasic}} {{.IsStruct}} {{.IsArray}}
                        	{{ $.D.AddToImports .PackagePath }}
                        {{- end -}}

						{{- $fieldName := .Name -}}
						{{- $hasIncluded := $.D.HasInStore .Kind -}}

						{{- if or .IsStruct .IsArray}}
							{{- if .IsPointer }}
								if st.{{$fieldName}} != nil {
							{{end}}

							{{- if .IsEmbedded }}
								// embedded {{ if .IsStruct}}struct{{else}}array{{end }} of {{ if .IsPointer }} `*{{ .Kind }}` {{ else }}  `{{ .Kind }}` {{end -}}
								{{- if not $hasIncluded }}
									{{- if $.D.GenerateAndStore .Kind -}}{{end -}}
								{{end -}}

								{{- if .IsStruct }}
									{{- range $field := ($.D.StructByKey $fieldName).Fields }}
										// embedded field : {{ $field.Name }} - pointer : {{ .IsPointer }}
										{{- if $field.IsPointer }}
											if st.{{$fieldName}}.{{$field.Name}} != nil {
										{{end}}
										sb.WriteString("{{$field.Name}}:\n"+fmt.Sprintf("%s", st.{{$fieldName}}.{{$field.Name}}))
										{{- if .IsImported -}}
											// EMBEDDED IMPORTED `{{.Package}}.{{.Name}}` `{{.PackagePath}}` and {{.IsBasic}} {{.IsStruct}} {{.IsArray}}
											{{ $.D.AddToImports .PackagePath }}
										{{- end -}}
										{{- if $field.IsPointer }}
											}
										{{end}}
									{{ end -}}
								{{ else }}
									// todo : array
									sb.WriteString("{{$fieldName}}:\n"+fmt.Sprintf("%s", st.{{$fieldName}}))
								{{ end -}}

							{{- else }}
								// {{ if .IsStruct}}struct{{else}}array{{end }} of {{ if .IsPointer }} `*{{ .Kind }}` {{ else }}  `{{ .Kind }}` {{end}}
								{{- if not $hasIncluded }}
									{{- if $.D.GenerateAndStore .Kind -}}{{end -}}
								{{end}}
								sb.WriteString("{{$fieldName}}:\n"+fmt.Sprintf("%s", st.{{$fieldName}}))
							{{end -}}

							{{- if .IsPointer }}
								}
							{{end}}
						{{- else if .IsBasic -}}

							{{- if .IsPointer -}}
								if st.{{$fieldName}} != nil{
							{{end -}}

							{{- if .IsBool }}
								sb.WriteString("{{$fieldName}}="+strconv.FormatBool({{- if .IsPointer -}}*{{- end -}}st.{{$fieldName}})+"\n")
							{{- else if .IsFloat }}
								sb.WriteString("{{$fieldName}}="+fmt.Sprintf("%0.f", {{- if .IsPointer -}}*{{- end -}}st.{{$fieldName}})+"\n")
							{{- else if .IsString }}
								sb.WriteString("{{$fieldName}}="+{{- if .IsPointer -}}*{{- end -}}st.{{$fieldName}}+"\n")
							{{- else if .IsUint }}
								sb.WriteString("{{$fieldName}}="+strconv.FormatUint(uint64({{- if .IsPointer -}}*{{- end -}}st.{{$fieldName}}), 10)+"\n")
							{{- else if .IsInt }}
								sb.WriteString("{{$fieldName}}="+strconv.Itoa(int({{- if .IsPointer -}}*{{- end -}}st.{{$fieldName}}))+"\n")
							{{- else }}
								// unhandled basic field typed {{.Kind}}
							{{end -}}

							{{- if .IsPointer -}}
								}
							{{- end }}
						{{ end -}}
					{{ end -}}
				{{ end }}
				return sb.String()
			}
		{{- end -}}
	{{- end -}}
{{- end -}}

{{- if .Declare "Stringer" }}{{end -}}
{{- template "Stringer" .Main -}}

{{ range .ListStored }}
{{.}}
{{end }}