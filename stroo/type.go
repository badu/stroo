package stroo

import (
	"go/ast"
	"strings"
)

type TypeInfo struct {
	Package     string
	PackagePath string
	Name        string
	Kind        string
	IsArray     bool // if it's not array, it's struct
	IsPointer   bool // if array, it's pointer
	Fields      Fields
	MethodList  Methods
	Comment     *ast.CommentGroup
	Reference   *TypeInfo
}

// cannot implement Stringer due to tests
func (s *TypeInfo) Debug(sdb *Debugger) {
	tabs := strings.Repeat("\t", sdb.tabs)
	sdb.WriteString(tabs + "&TypeInfo{\n")
	sdb.tabs++
	tabs = strings.Repeat("\t", sdb.tabs)
	sdb.WriteString(tabs + "Name:\"" + s.Name + "\",\n")
	sdb.WriteString(tabs + "Kind:\"" + s.Kind + "\",\n")
	if s.Package != "" {
		sdb.WriteString(tabs + "Package:\"" + s.Package + "\",\n")
	}
	if s.PackagePath != "" {
		sdb.WriteString(tabs + "PackagePath:\"" + s.PackagePath + "\",\n")
	}
	if s.IsArray {
		sdb.WriteString(tabs + "IsArray:true,\n")
	}
	if s.IsPointer {
		sdb.WriteString(tabs + "IsPointer:true,\n")
	}

	if s.IsArray {
		switch s.Kind {
		case "bool", "int", "int8", "int16", "int32", "rune", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "complex64", "complex128", "string":
			// it's ok - basic
		default:
			fieldData, visited := sdb.visitedFields[s.Kind]
			if !visited { // already visited (avoid self reference and infinite loop)
				s.Reference = sdb.types.Extract(s.Kind)
				if s.Reference == nil {
					panic(s.Kind + " not found in types")
				}
				cachedSb := NewDebugger(sdb.types, sdb.tabs, sdb.name)
				s.Reference.Debug(cachedSb)
				sdb.visitedFields[s.Kind] = cachedSb.String() // store in cache, in case of revisit
				fieldData = sdb.visitedFields[s.Kind]
			}
			sdb.WriteString(tabs + "Reference:" + fieldData) // already seen - use cached
		}
	}

	if len(s.Fields) > 0 {
		if !s.Fields.HasFieldKind(s.Kind) {
			s.Fields.Debug(sdb) // the usual route, no circular references
		} else {
			// we have self reference
			for _, field := range s.Fields {
				if field.Kind != s.Kind {
					field.Debug(sdb)
				} else {
					field.willNotVisitReference = true
				}
			}
			s.Fields.Debug(sdb)
		}
	}
	sdb.WriteString(tabs + "},\n")
	sdb.tabs--
}

type TypesSlice []*TypeInfo

// cannot implement Stringer due to tests
func (s TypesSlice) Debug(sdb *Debugger) {
	tabs := strings.Repeat("\t", sdb.tabs)
	sdb.WriteString(tabs + "TypesSlice:TypesSlice{\n")
	sdb.tabs++
	for _, el := range s {
		el.Debug(sdb)
	}
	sdb.tabs--
	sdb.WriteString(tabs + "},\n")
}

func (s TypesSlice) Extract(typeName string) *TypeInfo {
	for _, typeDef := range s {
		if typeDef.Name == typeName {
			return typeDef
		}
	}
	return nil
}
