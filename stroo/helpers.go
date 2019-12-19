package stroo

import "go/ast"

func getReceiver(fnDecl *ast.FuncDecl) interface{} {
	if fnDecl.Recv == nil {
		return nil
	}
	if len(fnDecl.Recv.List[0].Names) == 0 {
		return nil
	}
	return fnDecl.Recv.List[0].Names[0].Name
}

func getReceiverType(f *ast.FuncDecl) interface{} {
	if f.Recv == nil {
		return nil
	}
	if len(f.Recv.List[0].Names) == 0 {
		return nil
	}
	switch f.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return f.Recv.List[0].Type.(*ast.Ident).Name
	case *ast.StarExpr:
		return f.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
	}
	return nil
}

func stringer(str interface{}) string {
	if str == nil {
		return ""
	}
	return str.(string)
}
