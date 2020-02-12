package stroo

import (
	"fmt"
	"go/ast"
	"go/types"
	"log"
)

type TypeInfo struct {
	Package     string            // current or imported package name
	PackagePath string            // current or imported package path
	Name        string            // for `type` in case of struct or func type declaration Name == Kind; for `field` the name of the field
	Kind        string            // for `type` usually the way we Extract; for `field` the kind of the field (for extracting `type`)
	Tags        Tags              // tags, for both
	Prefix      string            // `field` info property, for storing embedding names
	Fields      TypesSlice        // for `type` fields ; for `field` is always nil
	MethodList  Methods           // for `type` methods ; for `field` first element contains `func data` if it's marked as `IsFunc`
	IsArray     bool              // for `type` if it's array it's not a struct, it's struct; for `field` if it's an array
	IsPointer   bool              // for `type` if array, it's pointer; for `field` if it's a pointer
	IsImported  bool              // for `type` if kind it's an imported one; for `field` if it's external to current package
	IsAlias     bool              // for `type` if it's alias; for `field` it's always false
	IsFunc      bool              // for `type` if it's a function type definition; for `field`
	IsStruct    bool              // `field` info property
	IsMap       bool              // `field` info property
	IsChan      bool              // `field` info property
	IsExported  bool              // `field` info property
	IsEmbedded  bool              // `field` info property
	IsInterface bool              // `field` info property
	Comment     *ast.CommentGroup // comment found in AST
}

func NewAliasFromField(pkg *types.Package, field *TypeInfo, name string) TypeInfo {
	return TypeInfo{
		Package:     pkg.Name(),
		PackagePath: pkg.Path(),
		Name:        name,
		Kind:        field.Kind,
		IsArray:     field.IsArray,
		IsPointer:   field.IsPointer,
		IsImported:  field.IsImported,
		IsAlias:     true,
	}
}

func (t *TypeInfo) IsBasic() bool {
	return IsBasic(t.Kind)
}

func (t *TypeInfo) Clone(newName string) TypeInfo {
	result := TypeInfo{
		Name:        newName,
		Kind:        t.Kind,
		IsPointer:   t.IsPointer,
		IsStruct:    t.IsStruct,
		IsArray:     t.IsArray,
		IsMap:       t.IsMap,
		IsChan:      t.IsChan,
		IsExported:  t.IsExported,
		IsEmbedded:  t.IsEmbedded,
		IsImported:  t.IsImported,
		IsInterface: t.IsInterface,
		IsFunc:      t.IsFunc,
		Tags:        t.Tags,
		Package:     t.Package,
		PackagePath: t.PackagePath,
		Comment:     t.Comment,
		Prefix:      t.Prefix,
	}
	copy(result.MethodList, t.MethodList)
	return result
}

func (t *TypeInfo) FuncData() (*FunctionInfo, error) {
	switch len(t.MethodList) {
	case 0:
		return nil, nil
	case 1:
		return &t.MethodList[0], nil
	default:
		return nil, fmt.Errorf("%d methods found on field %s %s", len(t.MethodList), t.Name, t.Kind)
	}
}

func (t *TypeInfo) SetPrefix(prefix string) error {
	t.Prefix = prefix
	return nil
}

func (t *TypeInfo) StructOrArrayString() string {
	if t.IsArray {
		return "array"
	}
	if t.IsStruct {
		return "struct"
	}
	return "nor struct nor array"
}

func (t *TypeInfo) PackageAndKind() string {
	return t.Package + "." + t.Kind
}

// in case we need to print `*Something` instead of `Something`
func (t *TypeInfo) RealKind() string {
	if t.IsPointer {
		return "*" + t.Kind
	}
	return t.Kind
}

func (t *TypeInfo) TagsByKey(name string) []string {
	if t.Tags == nil {
		return nil
	}
	tag, err := t.Tags.Get(name)
	if err != nil {
		log.Printf("error: %v", err)
		return nil
	}
	return tag.Options
}

func (t *TypeInfo) IsBool() bool {
	return t.Kind == "bool"
}

func (t *TypeInfo) IsString() bool {
	return t.Kind == "string"
}

func (t *TypeInfo) IsUnsafe() bool {
	return t.Kind == "unsafe.Pointer"
}

func (t *TypeInfo) IsRune() bool {
	return t.Kind == "rune"
}

func (t *TypeInfo) IsFloat() bool {
	return IsFloat(t.Kind)
}

func (t *TypeInfo) IsUint() bool {
	return IsUint(t.Kind)
}

func (t *TypeInfo) IsInt() bool {
	return IsInt(t.Kind)
}

func (t *TypeInfo) IsComplex() bool {
	return IsComplex(t.Kind)
}

type TypesSlice []TypeInfo

// implementation of Sorter interface, so we can sort fields
func (s TypesSlice) Len() int { return len(s) }
func (s TypesSlice) Less(i, j int) bool {
	if s[i].Name != "" && s[j].Name != "" {
		return s[i].Name < s[j].Name
	}
	// for embedded kinds (fields)
	return s[i].Kind < s[j].Kind
}
func (s TypesSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s TypesSlice) HasFieldKind(kind string) bool {
	for _, field := range s {
		if field.Kind == kind {
			return true
		}
	}
	return false
}

// this should be called only from Code's StructByKey method
func (s TypesSlice) Extract(typeName string) *TypeInfo {
	for _, typeDef := range s {
		// first try by "Kind"
		if typeDef.Kind == typeName {
			return &typeDef
		}
		// next try by "Name"
		if typeDef.Name == typeName {
			return &typeDef
		}
	}
	return nil
}

type VarInfo struct {
	Name string
	Type *TypeInfo
	Kind string
}

type Vars []VarInfo

// implementation of Sorter interface, so we can sort
func (s Vars) Len() int { return len(s) }
func (s Vars) Less(i, j int) bool {
	if s[i].Name != "" && s[j].Name != "" {
		return s[i].Name < s[j].Name
	}
	// for embedded kinds
	return s[i].Kind < s[j].Kind
}
func (s Vars) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type FunctionInfo struct {
	Package          string
	PackagePath      string
	Name             string
	Kind             string // currently unused
	ReceiverName     string // empty for normal functions
	ReceiverType     string
	IsMethodReceiver bool
	IsExported       bool
	Params           []VarInfo
	Returns          []VarInfo
	comment          *ast.CommentGroup
}

type Methods []FunctionInfo

// implementation of Sorter interface, so we can sort
func (s Methods) Len() int { return len(s) }
func (s Methods) Less(i, j int) bool {
	if s[i].Name != "" && s[j].Name != "" {
		return s[i].Name < s[j].Name
	}
	// for embedded kinds
	return s[i].Kind < s[j].Kind
}
func (s Methods) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
