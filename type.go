package stroo

import (
	"go/ast"
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
	keeper      map[string]interface{} // template authors keeps data in here, key-value, as they need
	root        *Code                  // reference to the root document - it's usefull
}

func (t *TypeInfo) SetRoot(doc *Code)              { t.root = doc }
func (t *TypeInfo) Root() *Code                    { return t.root }
func (t *TypeInfo) Keeper() map[string]interface{} { return t.keeper }
func (t *TypeInfo) Store(key string, value interface{}) bool {
	if t.keeper == nil {
		t.keeper = make(map[string]interface{})
	}
	_, has := t.keeper[key]
	t.keeper[key] = value
	return has
}

func (t *TypeInfo) Retrieve(key string) interface{} {
	if t.keeper == nil {
		t.keeper = make(map[string]interface{})
	}
	value, _ := t.keeper[key]
	return value
}

func (t *TypeInfo) HasInStore(key string) bool {
	if t.keeper == nil {
		t.keeper = make(map[string]interface{})
	}
	_, has := t.keeper[key]
	return has
}

type TypesSlice []*TypeInfo

func (s TypesSlice) Extract(typeName string) *TypeInfo {
	for _, typeDef := range s {
		if typeDef.Name == typeName {
			return typeDef
		}
	}
	return nil
}
