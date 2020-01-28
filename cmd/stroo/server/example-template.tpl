{{ if .Declare "String" }}{{ end }}{{/* pass kind of methods we're going to generate */}}
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
{{ define "Pointer" }}
	{{- if .IsPointer }} if st.{{.Prefix}}{{.Name}} != nil{ {{ end }}
{{ end }}
{{ define "PointerClose" }}
	{{ if .IsPointer -}} } {{- end }}
{{ end }}
{{ define "BasicType" }}
	{{- template "Pointer" . -}}
	{{- if .IsBool -}}
        sb.WriteString("{{.Prefix}}{{.Name}}="+strconv.FormatBool({{ if .IsPointer }}*{{ end }}st.{{.Prefix}}{{.Name}})+"\n")
	{{- else if .IsFloat -}}
		sb.WriteString("{{.Prefix}}{{.Name}}="+fmt.Sprintf("%0.f", {{ if .IsPointer }}*{{ end }}st.{{.Prefix}}{{.Name}})+"\n")
	{{- else if .IsString -}}
sb.WriteString("{{.Prefix}}{{.Name}}="+{{ if .IsPointer }}*{{ end }}st.{{.Prefix}}{{.Name}}+"\n")
	{{- else if .IsUint -}}
		sb.WriteString("{{.Prefix}}{{.Name}}="+strconv.FormatUint(uint64({{ if .IsPointer }}*{{ end }}st.{{.Prefix}}{{.Name}}), 10)+"\n")
	{{- else if .IsInt -}}
		sb.WriteString("{{.Prefix}}{{.Name}}="+strconv.Itoa(int({{ if .IsPointer }}*{{ end }}st.{{.Prefix}}{{.Name}}))+"\n")
	{{- else -}}
		// implement me : basic field typed {{.Kind}}
	{{- end -}}
	{{- template "PointerClose" . -}}
{{ end }}
{{ define "Recurse" }}
	{{- if (.Root.HasNotGenerated .Kind) -}}
		{{- if .Root.RecurseGenerate .Kind -}}{{- end -}}
	{{- end -}}
{{ end }}
{{ define "Embedded" }}
	// embedded `{{.StructOrArrayString}}` of `{{.RealKind}}` with prefix `{{.Prefix}}`
	{{ template "Recurse" . -}}
	{{- if .IsStruct }}		
		{{- $outerName := concat (concat .Name ".") .Prefix -}}
        {{- range $field := (.Root.StructByKey .Name).Fields -}}
			// embedded field named `{{ $field.Name }}` of type `{{ .RealKind }}`
            {{ $err := $field.SetPrefix $outerName }}
			{{- template "Pointer" $field -}}
			{{- if or $field.IsStruct $field.IsArray -}}
				{{- template "StructOrArray" $field -}}
			{{ else if $field.IsBasic }}
				{{- template "BasicType" $field -}}
			{{ else if $field.IsEmbedded }}
				// current prefix = `{{ $outerName }}`
				{{ template "Embedded" $field }}
			{{ else }}
				// implement me : embedded {{ $field.Kind }}
			{{ end }}			
            {{- template "PointerClose" . -}}
			{{ $err = $field.SetPrefix "" }} {{/*reset prefix, so we won't print others fields prefixes*/}}
		{{- end -}}
	{{ else if .IsArray }}
		// todo : embedded array
		sb.WriteString("{{.Name}}:\n"+fmt.Sprintf("%s", st.{{.Name}}))
	{{ else }}
		// implement me : embedded SOMETHING else {{.}}
	{{ end }}
{{ end }}
{{ define "StructOrArray" }}
	{{- template "Pointer" . -}}
	{{- if .IsEmbedded -}}
		{{- template "Embedded" . -}}
	{{ else }}
		// {{.StructOrArrayString}} field `{{.Name}}` of type `{{.RealKind}}`
		{{- template "Recurse" . }}
		sb.WriteString("{{.Name}}:\n"+fmt.Sprintf("%s", st.{{.Name}}))
	{{- end -}}
	{{- template "PointerClose" . -}}
{{ end }}
{{ define "ArrayStringer" }}
  // Stringer implementation for {{ .Name }} kind : {{.Kind}}
  func (st {{ .Name }}) String() string {
    var sb strings.Builder
    for _, el := range st {
      sb.WriteString("{{.Kind}}:\n"+fmt.Sprintf("%s", el))
      {{- template "Recurse" . -}}
    }
    return sb.String()
  }
{{ end }}
{{ define "StructStringer" }}
	// Stringer implementation for {{ .Kind}}
	func (st {{ .Kind }}) String() string {
		var sb strings.Builder
      	{{ if sort .Fields }}{{ end -}}
		{{ range .Fields -}}
			{{ if .IsExported -}}
				{{- if or .IsStruct .IsArray -}}
					{{- template "StructOrArray" . -}}
				{{ else if .IsBasic }}
					{{- template "BasicType" . -}}
				{{ else -}}
      				/** implement me {{ .Root.Implements .Package .Kind }}! **/
      			{{ end }}
			{{ end -}}
		{{ end }}
		return sb.String()
	}
{{ end }}
{{ define "String" }}
	{{- with . }}
		{{- if .IsArray }}{{/* cannot be anything else than Array or Struct */}}
			{{ template "ArrayStringer" . }}
		{{ else  }}
			{{- template "StructStringer" . }}
		{{ end }}
	{{ end }}
{{ end }}
{{- with .Main -}}
	{{ template "String" . }}
{{ end }}
{{ range .ListStored }}
{{.}}
{{ end }}