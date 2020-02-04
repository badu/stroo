package stroo

type VarInfo struct {
	Name string
	Type *TypeInfo
	Kind string
}

type Vars []VarInfo
