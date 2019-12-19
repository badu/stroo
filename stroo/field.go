package stroo

import (
	"go/ast"
	"log"
	"strings"
)

type FieldInfo struct {
	Name       string
	TypeName   string
	IsBasic    bool
	IsPointer  bool
	IsStruct   bool
	IsSlice    bool
	IsArray    bool
	IsMap      bool
	IsChan     bool
	IsExported bool
	IsEmbedded bool
	Reference  *TypeInfo

	Tags        *Tags
	Comment     *ast.CommentGroup
	Package     string
	PackagePath string
	MethodList  Methods
}

// cannot implement Stringer due to tests
func (f FieldInfo) Debug(sb *strings.Builder, args ...int) {
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
	if f.IsStruct {
		sb.WriteString(tabs + "IsStruct:true,\n")
	}
	if f.IsSlice {
		sb.WriteString(tabs + "IsSlice:true,\n")
	}
	if f.IsArray {
		sb.WriteString(tabs + "IsArray:true,\n")
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
	if f.Reference != nil {
		sb.WriteString(tabs + "Reference:")
		if tno > 0 {
			f.Reference.Debug(sb, tno)
		} else {
			f.Reference.Debug(sb)
		}
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
	if tno > 0 {
		f.MethodList.Debug(sb, tno)
	} else {
		f.MethodList.Debug(sb)
	}
	sb.WriteString(tabs + "},\n")
}

// overwrites field information with data from array info
func (f *FieldInfo) FromArray(arrayReference *FieldInfo) {
	f.IsBasic = arrayReference.IsBasic
	f.IsStruct = false // force false, because it's array
	f.IsArray = true
	f.Reference = arrayReference.Reference
	f.Reference.ReferenceIsPointer = arrayReference.IsPointer // signal `type T []*V`
	f.Tags = arrayReference.Tags
	f.Comment = arrayReference.Comment
	f.MethodList = arrayReference.MethodList
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
		tno++
	}
	if len(f) > 0 {
		sb.WriteString(tabs + "Fields:Fields{\n")
	}
	for _, field := range f {
		if tno > 0 {
			field.Debug(sb, tno)
		} else {
			field.Debug(sb)
		}
	}
	if len(f) > 0 {
		sb.WriteString(tabs + "},\n")
	}
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
