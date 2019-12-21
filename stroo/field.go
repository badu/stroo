package stroo

import (
	"go/ast"
	"log"
	"strings"
)

type FieldInfo struct {
	Name           string
	TypeName       string
	IsBasic        bool
	IsPointer      bool
	IsStruct       bool
	IsArray        bool
	IsMap          bool
	IsChan         bool
	IsExported     bool
	IsEmbedded     bool
	Reference      *TypeInfo  // if it's a struct, we have struct info here
	ArrayReference *FieldInfo // if it's array, we have the field info here
	Tags           *Tags
	Comment        *ast.CommentGroup
	Package        string
	PackagePath    string
	MethodList     Methods
	ReferenceName  string // after nullified Reference or ArrayReference, we keep the name here to get from cache
}

var (
	VisitedStructs = make(map[string]struct{})
	VisitedFields  = make(map[string]string)
)

// cannot implement Stringer due to tests
func (f *FieldInfo) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
		tno++
	}
	sb.WriteString(tabs + "&FieldInfo{\n")
	if f.Name != "" {
		sb.WriteString(tabs + "Name:\"" + f.Name + "\",\n")
	}
	if f.TypeName != "" {
		sb.WriteString(tabs + "TypeName:\"" + f.TypeName + "\",\n")
	}
	if f.IsBasic {
		sb.WriteString(tabs + "IsBasic:true,\n")
	}
	if f.IsPointer {
		sb.WriteString(tabs + "IsPointer:true,\n")
	}
	if f.IsMap {
		sb.WriteString(tabs + "IsMap:true,\n")
	}
	if f.IsChan {
		sb.WriteString(tabs + "IsChan:true,\n")
	}
	if f.IsExported {
		sb.WriteString(tabs + "IsExported:true,\n")
	}
	if f.IsEmbedded {
		sb.WriteString(tabs + "IsEmbedded:true,\n")
	}
	if f.Tags != nil {
		if tno > 0 {
			f.Tags.Debug(sb, tno)
		} else {
			f.Tags.Debug(sb)
		}
	}
	if f.Package != "" {
		sb.WriteString(tabs + "Package:\"" + f.Package + "\",\n")
	}
	if f.PackagePath != "" {
		sb.WriteString(tabs + "PackagePath:\"" + f.PackagePath + "\",\n")
	}

	if f.IsStruct {
		sb.WriteString(tabs + "IsStruct:true,\n")
		if !f.IsBasic {
			if f.Reference != nil {
				f.ReferenceName = f.Reference.Name
				if _, visited := VisitedStructs[f.ReferenceName]; !visited { // already visited (avoid self reference and infinite loop)
					VisitedStructs[f.Reference.Name] = struct{}{} // first, mark as visited, so we won't enter here again because of fields
					var cachedSb strings.Builder
					tno++
					if tno > 0 {
						f.Reference.Debug(&cachedSb, tno)
					} else {
						f.Reference.Debug(&cachedSb)
					}
					VisitedFields[f.ReferenceName] = cachedSb.String() // store in cache, in case of revisit
					f.Reference = nil
				}
			} else {
				if VisitedFields[f.ReferenceName] != "" {
					sb.WriteString(tabs + "Reference:" + VisitedFields[f.ReferenceName])
				} else {
					//panic("reference is nil for stuct field " + f.Name + " having reference named " + f.ReferenceName)
				}
			}
		}
	}
	if f.IsArray {
		sb.WriteString(tabs + "IsArray:true,\n")
		if !f.IsBasic {
			if f.Reference != nil {
				f.ReferenceName = f.Reference.Name
				if _, visited := VisitedStructs[f.ReferenceName]; !visited { // already visited (avoid self reference and infinite loop)
					VisitedStructs[f.ReferenceName] = struct{}{} // first, mark as visited, so we won't enter here again because of fields
					var cachedSb strings.Builder
					tno++
					if tno > 0 {
						f.Reference.Debug(&cachedSb, tno)
					} else {
						f.Reference.Debug(&cachedSb)
					}
					VisitedFields[f.ReferenceName] = cachedSb.String() // store in cache, in case of revisit
					f.Reference = nil
				}
				if VisitedFields[f.ReferenceName] != "" {
					sb.WriteString(tabs + "Reference:" + VisitedFields[f.ReferenceName])
				}
			} else {
				//panic("reference is nil for array field " + f.Name + " having reference named " + f.ReferenceName)
			}
		}
	}

	if tno > 0 {
		f.MethodList.Debug(sb, tno)
	} else {
		f.MethodList.Debug(sb)
	}
	sb.WriteString(tabs + "},\n")
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

type Fields []*FieldInfo

// cannot implement Stringer due to tests
func (f Fields) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
	}
	sb.WriteString(tabs + "Fields:Fields{\n")
	tno++
	for _, field := range f {
		if tno > 0 {
			field.Debug(sb, tno)
		} else {
			field.Debug(sb)
		}
	}
	sb.WriteString(tabs + "},\n")
}

// implementation of Sorter interface, so we can sort fields
func (f Fields) Len() int { return len(f) }
func (f Fields) Less(i, j int) bool {
	if f[i].Name != "" && f[j].Name != "" {
		return f[i].Name < f[j].Name
	}
	// for embedded fields
	return f[i].TypeName < f[j].TypeName
}
func (f Fields) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
