package stroo

import (
	"go/ast"
)

type InterfaceInfo struct {
	Name        string
	Methods     []string
	Package     string
	PackagePath string
	Comment     *ast.CommentGroup
}

type Interfaces []*InterfaceInfo
