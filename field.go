package stroo

import (
	"go/ast"
	"log"
)

type FieldInfo struct {
	Name        string
	Kind        string
	IsBasic     bool
	IsPointer   bool
	IsStruct    bool
	IsArray     bool
	IsMap       bool
	IsChan      bool
	IsExported  bool
	IsEmbedded  bool
	IsImported  bool
	IsInterface bool
	Reference   *TypeInfo // if it's a struct, we have struct info here
	Tags        *Tags
	Package     string
	PackagePath string
	Comment     *ast.CommentGroup
}

func (f *FieldInfo) TagsByKey(name string) []string {
	if f.Tags == nil {
		return nil
	}
	tag, err := f.Tags.Get(name)
	if err != nil {
		log.Printf("error: %v", err)
		return nil
	}
	return tag.Options
}

func (f *FieldInfo) IsBool() bool {
	if f.Kind == "bool" {
		return true
	}
	return false
}

func (f *FieldInfo) IsString() bool {
	if f.Kind == "string" {
		return true
	}
	return false
}

func (f *FieldInfo) IsFloat() bool {
	if f.Kind == "float32" || f.Kind == "float64" {
		return true
	}
	return false
}

func (f *FieldInfo) IsUint() bool {
	if f.Kind == "uint" || f.Kind == "uint8" || f.Kind == "uint16" || f.Kind == "uint32" || f.Kind == "uint64" {
		return true
	}
	return false
}

func (f *FieldInfo) IsInt() bool {
	if f.Kind == "int" || f.Kind == "int8" || f.Kind == "int16" || f.Kind == "int32" || f.Kind == "int64" {
		return true
	}
	return false
}

type Fields []*FieldInfo

func (f Fields) HasFieldKind(kind string) bool {
	for _, field := range f {
		if field.Kind == kind {
			return true
		}
	}
	return false
}

// implementation of Sorter interface, so we can sort fields
func (f Fields) Len() int { return len(f) }
func (f Fields) Less(i, j int) bool {
	if f[i].Name != "" && f[j].Name != "" {
		return f[i].Name < f[j].Name
	}
	// for embedded fields
	return f[i].Kind < f[j].Kind
}
func (f Fields) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
