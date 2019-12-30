package stroo

import (
	"go/ast"
	"log"
	"strings"
)

type FieldInfo struct {
	Name                  string
	Kind                  string
	IsBasic               bool
	IsPointer             bool
	IsStruct              bool
	IsArray               bool
	IsMap                 bool
	IsChan                bool
	IsExported            bool
	IsEmbedded            bool
	IsImported            bool
	IsInterface           bool
	Reference             *TypeInfo // if it's a struct, we have struct info here
	Tags                  *Tags
	Package               string
	PackagePath           string
	Comment               *ast.CommentGroup
	willNotVisitReference bool
}

type Debugger struct {
	strings.Builder
	name          int
	tabs          int
	visitedFields map[string]string
	types         *TypesSlice
}

func (d *Debugger) WriteString(s string) {
	d.Builder.WriteString(s)
}

func NewDebugger(types *TypesSlice, args ...int) *Debugger {
	tabs := 0
	name := 0
	if len(args) > 0 {
		tabs = args[0]
		if len(args) > 1 {
			name = args[1]
		}
	}
	return &Debugger{
		types:         types,
		tabs:          tabs,
		name:          name,
		visitedFields: make(map[string]string),
	}
}

// cannot implement Stringer due to tests
func (f *FieldInfo) Debug(sdb *Debugger) {
	tabs := strings.Repeat("\t", sdb.tabs)
	sdb.WriteString(tabs + "&FieldInfo{\n")
	if f.Name != "" {
		sdb.WriteString(tabs + "Name:\"" + f.Name + "\",\n")
	}
	if f.Kind != "" {
		sdb.WriteString(tabs + "Kind:\"" + f.Kind + "\",\n")
	}
	if f.IsBasic {
		sdb.WriteString(tabs + "IsBasic:true,\n")
	}
	if f.IsPointer {
		sdb.WriteString(tabs + "IsPointer:true,\n")
	}
	if f.IsMap {
		sdb.WriteString(tabs + "IsMap:true,\n")
	}
	if f.IsChan {
		sdb.WriteString(tabs + "IsChan:true,\n")
	}
	if f.IsExported {
		sdb.WriteString(tabs + "IsExported:true,\n")
	}
	if f.IsEmbedded {
		sdb.WriteString(tabs + "IsEmbedded:true,\n")
	}
	if f.IsInterface {
		sdb.WriteString(tabs + "IsInterface:true,\n")
	}
	if f.Package != "" {
		sdb.WriteString(tabs + "Package:\"" + f.Package + "\",\n")
	}
	if f.PackagePath != "" {
		sdb.WriteString(tabs + "PackagePath:\"" + f.PackagePath + "\",\n")
	}
	if f.IsArray {
		sdb.WriteString(tabs + "IsArray:true,\n")
	}
	if f.IsStruct {
		sdb.WriteString(tabs + "IsStruct:true,\n")
	}
	if f.IsArray || f.IsStruct && !(f.IsImported || f.IsInterface) {
		switch f.Kind {
		case "bool", "int", "int8", "int16", "int32", "rune", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "complex64", "complex128", "string":
			// it's ok - basic
		default:
			if !f.willNotVisitReference {
				fieldData, visited := sdb.visitedFields[f.Kind]
				if !visited { // already visited (avoid self reference and infinite loop)
					f.Reference = sdb.types.Extract(f.Kind)
					if f.Reference == nil {
						panic(f.Kind + " not found in types")
					}
					cachedSb := NewDebugger(sdb.types, sdb.tabs, sdb.name)
					f.Reference.Debug(cachedSb)
					sdb.visitedFields[f.Kind] = cachedSb.String() // store in cache, in case of revisit
					fieldData = sdb.visitedFields[f.Kind]
				}
				sdb.WriteString(tabs + "Reference:" + fieldData) // already seen - use cached
			}
		}
	}

	if f.Tags != nil {
		f.Tags.Debug(sdb)
	}
	sdb.WriteString(tabs + "},\n")
}

func (f *FieldInfo) TagsByKey(name string) []string {
	if f.Tags == nil {
		return nil
	}
	tag, err := f.Tags.Get(name)
	if err != nil {
		log.Fatalf("error: %v", err)
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

// cannot implement Stringer due to tests
func (f Fields) Debug(sdb *Debugger) {
	tabs := strings.Repeat("\t", sdb.tabs)
	sdb.WriteString(tabs + "Fields:Fields{\n")
	sdb.tabs++
	for _, field := range f {
		field.Debug(sdb)
	}
	sdb.tabs--
	sdb.WriteString(tabs + "},\n")
}

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
