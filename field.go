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
	Tags        *Tags
	Package     string
	PackagePath string
	Comment     *ast.CommentGroup
	keeper      map[string]interface{} // template authors keeps data in here, key-value, as they need
	root        *Code
}

func (f *FieldInfo) SetRoot(doc *Code) { f.root = doc }
func (f *FieldInfo) Root() *Code       { return f.root }
func (f *FieldInfo) StructOrArrayString() string {
	if f.IsArray {
		return "array"
	}
	if f.IsStruct {
		return "struct"
	}
	return "nor struct nor array"
}

// in case we need to print `*Something` instead of `Something`
func (f *FieldInfo) RealKind() string {
	if f.IsPointer {
		return "*" + f.Kind
	}
	return f.Kind
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

func (f *FieldInfo) Keeper() map[string]interface{} { return f.keeper }
func (f *FieldInfo) Store(key string, value interface{}) bool {
	if f.keeper == nil {
		f.keeper = make(map[string]interface{})
	}
	_, has := f.keeper[key]
	f.keeper[key] = value
	return has
}

func (f *FieldInfo) Retrieve(key string) interface{} {
	if f.keeper == nil {
		f.keeper = make(map[string]interface{})
	}
	value, _ := f.keeper[key]
	return value
}

func (f *FieldInfo) HasInStore(key string) bool {
	if f.keeper == nil {
		f.keeper = make(map[string]interface{})
	}
	_, has := f.keeper[key]
	return has
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
