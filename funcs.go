package stroo

import "go/ast"

type FunctionInfo struct {
	Package          string
	PackagePath      string
	Name             string
	ReceiverName     string // empty for normal functions
	ReceiverType     string
	IsMethodReceiver bool
	IsExported       bool
	Params           []VarInfo
	Returns          []VarInfo
	comment          *ast.CommentGroup
}

type Methods []FunctionInfo
