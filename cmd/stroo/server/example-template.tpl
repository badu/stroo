{{ if .Declare "Stringer" }}{{ end }}{{/* pass kind of methods we're going to generate */}}
{{- .AddToImports "bytes" }}{{/* knowing that we're going to use this packges */}}
{{- .AddToImports "strconv" }}{{/* we're adding them to imports */}}
{{- .AddToImports "fmt" }}
{{- .AddToImports "strings" -}}
package {{.PackageInfo.Name}}
{{ if .Imports }}
import (
	{{range .Imports -}}
	"{{ . }}"
	{{end -}}
)
{{- end -}}
{{ define "BasicType" }}
	{{- if .IsPointer }} if st.{{.Name}} != nil{ {{ end }}
	{{- if .IsBool -}}
        sb.WriteString("{{.Name}}="+strconv.FormatBool({{ if .IsPointer }}*{{ end }}st.{{.Name}})+"\n")
	{{- else if .IsFloat -}}
		sb.WriteString("{{.Name}}="+fmt.Sprintf("%0.f", {{ if .IsPointer }}*{{ end }}st.{{.Name}})+"\n")
	{{- else if .IsString -}}
		sb.WriteString("{{.Name}}="+{{ if .IsPointer }}*{{ end }}st.{{.Name}}+"\n")
	{{- else if .IsUint -}}
		sb.WriteString("{{.Name}}="+strconv.FormatUint(uint64({{ if .IsPointer }}*{{ end }}st.{{.Name}}), 10)+"\n")
	{{- else if .IsInt -}}
		sb.WriteString("{{.Name}}="+strconv.Itoa(int({{ if .IsPointer }}*{{ end }}st.{{.Name}}))+"\n")
	{{- else -}}
		// unhandled basic field typed {{.Kind}}
	{{- end -}}
	{{ if .IsPointer -}} } {{- end }}
{{ end }}
{{ define "StructOrArray" }}
	{{- $D := .Retrieve "D" -}}
	{{ if .IsPointer }} if st.{{.Name}} != nil {  {{ end }}
	{{- if .IsEmbedded -}}
		// embedded `{{.StructOrArrayString}}` of `{{.RealKind}}`
		{{- if not ($D.HasInStore .Kind) -}}
			{{ if $D.GenerateAndStore .Kind }}{{end }}
		{{- end }}
		{{- if .IsStruct }}
		  {{- $outerName := .Name -}}
          {{- range $field := ($D.StructByKey .Name).Fields -}}
              // embedded field named `{{ $field.Name }}` of type `{{ .RealKind }}`
              {{ if $field.IsPointer }} if st.{{$outerName}}.{{$field.Name}} != nil { {{ end }}
              sb.WriteString("{{concat $outerName $field.Name}}:\n"+fmt.Sprintf("%s", st.{{$outerName}}.{{$field.Name}}))
              {{ if .IsImported }}
				  // embedded imported `{{.Package}}.{{.Name}}`
                  {{ $D.AddToImports .Package }}
              {{ end }}
              {{ if $field.IsPointer }} } {{ end }}
          {{- end -}}
		{{ else if .IsArray }}
			// todo : embedded array
			sb.WriteString("{{.Name}}:\n"+fmt.Sprintf("%s", st.{{.Name}}))
		{{ else }}
			// embedded something else {{.}}
		{{ end }}
	{{ else }}
		// {{.StructOrArrayString}} field `{{.Name}}` of type `{{.RealKind}}`
		{{- if not ($D.HasInStore .Kind) -}}
			{{ if $D.GenerateAndStore .Kind }}{{ end }}
		{{- end }}
		sb.WriteString("{{.Name}}:\n"+fmt.Sprintf("%s", st.{{.Name}}))
	{{- end -}}
	{{ if .IsPointer -}} } {{- end }}
{{ end }}
{{ define "ArrayStringer" }}
  {{- $D := .Retrieve "D" -}}
  // Stringer implementation for {{ .Name }} kind : {{.Kind}}  asd
  func (st {{ .Name }}) String() string {
    var sb strings.Builder;
    for _, el := range st {
      sb.WriteString("{{.Kind}}:\n"+fmt.Sprintf("%s", el))
      {{ if not ($D.HasInStore .Kind) }}
          {{ if $D.GenerateAndStore .Kind }}{{ end }}
      {{ end }}
    }
    return sb.String()
  }
{{ end }}
{{ define "StructStringer" }}
	{{- $D := .Retrieve "D" -}}
	// Stringer implementation for {{ .Kind}}
	func (st {{ .Kind }}) String() string {
		var sb strings.Builder;
      	{{- if sort .Fields }}{{ end -}}
		{{ range .Fields -}}
			{{ if .IsExported -}}
				{{ if .IsImported }}
                  // Not processed : `{{.Name}}` imported field from `{{.Package}}`
                  {{ $D.AddToImports .Package }}
				{{ end }}
				{{- if or .IsStruct .IsArray -}}
					{{ if (.Store "D" $D) }} {{ end }}
					{{- template "StructOrArray" . -}}
				{{ else if .IsBasic }}
					{{- template "BasicType" . }}
				{{ end -}}
			{{ end -}}
		{{ end }}
		return sb.String()
	}
{{ end }}
{{ define "Stringer" }}
	{{- with .T }}
		{{- if (.Store "D" $.D) -}} {{- end -}} {{/* pass reference to document */}}
		{{- if .IsArray }}
			{{ template "ArrayStringer" . }}
		{{ else  }}
			{{- template "StructStringer" . }}
		{{ end }}
	{{ end }}
{{ end }}

{{- with .Main -}}
	{{ template "Stringer" . }}
{{ end }}
{{ range .ListStored }}
{{.}}
{{ end }}