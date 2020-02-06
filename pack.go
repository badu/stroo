package stroo

import (
	"errors"
	"go/ast"
	"go/types"
	"log"
)

type Imports struct {
	Name string
	Path string
}
type PackageInfo struct {
	Name       string
	Path       string
	Types      TypesSlice
	Interfaces TypesSlice
	Functions  Methods
	Vars       Vars
	TypesInfo  *types.Info
	Imports    []*Imports
	PrintDebug bool
}

func (pkg *PackageInfo) LoadImports(fromImports []*types.Package) {
	for _, imprt := range fromImports {
		if pkg.PrintDebug {
			log.Printf("Adding %q %q to imports", imprt.Path(), imprt.Name())
		}
		pkg.Imports = append(pkg.Imports, &Imports{Path: imprt.Path(), Name: imprt.Name()})
	}
}

func fixFieldsInfo(defs *types.Info, forType *TypeInfo) error {
	for idx := range forType.Fields {
		for key, value := range defs.Defs {
			if key.Name == forType.Fields[idx].Kind {
				if forType.Fields[idx].Package == "" {
					log.Printf("%q DOESN'T have a package", forType.Fields[idx].Kind)
				}
				underlyingValue := value.Type().Underlying()
				switch underType := underlyingValue.(type) {
				case *types.Pointer:
					underlyingPointer := underType.Elem().Underlying()
					forType.Fields[idx].IsPointer = true
					switch underPtr := underlyingPointer.(type) {
					case *types.Struct:
						forType.Fields[idx].IsStruct = true
					case *types.Slice:
						if !forType.Fields[idx].IsArray {
							forType.Fields[idx].IsArray = true
						}
					case *types.Interface:
						if !forType.Fields[idx].IsInterface {
							forType.Fields[idx].IsInterface = true
						}
					default:
						log.Printf("Fixer [pointer]: %#v", underPtr)
					}
				case *types.Struct:
					forType.Fields[idx].IsStruct = true
				case *types.Slice:
					if !forType.Fields[idx].IsArray {
						forType.Fields[idx].IsArray = true
					}
				case *types.Interface:
					if !forType.Fields[idx].IsInterface {
						forType.Fields[idx].IsInterface = true
					}
				case *types.Basic:
					log.Printf("Fixer [basic]: %#v", underType)
				case *types.Signature:
					// ignore signatures
					//log.Printf("Fixer [signature]: %#v", underType)
				default:
					log.Printf("Fixer [?]: %#v", underType)
				}
				break // found it, bail out
			}
		}
	}
	return nil
}

func readIdent(ident *ast.Ident, comment *ast.CommentGroup) (*FieldInfo, error) {
	var (
		result FieldInfo
		err    error
	)
	if comment != nil {
		// direct call from Preorder
		result.Comment = comment
	}
	result.Kind = ident.Name
	if IsBasic(ident.Name) {
		result.IsBasic = true
	}
	return &result, err
}

func readPointer(ptr *ast.StarExpr, comment *ast.CommentGroup) (*FieldInfo, error) {
	var (
		result FieldInfo
		err    error
	)
	if comment != nil {
		// direct call from Preorder
		result.Comment = comment
	}
	result.IsPointer = true
	switch typedSpec := ptr.X.(type) {
	case *ast.Ident:
		fieldInfo, err := readIdent(typedSpec, nil)
		if err == nil {
			result.Kind = fieldInfo.Kind
			result.IsBasic = fieldInfo.IsBasic
		}
	case *ast.SelectorExpr:
		fieldInfo, err := readSelector(typedSpec, nil)
		if err == nil {
			result.Kind = fieldInfo.Kind
			result.IsBasic = fieldInfo.IsBasic
			result.IsImported = fieldInfo.IsImported
			if fieldInfo.Package != "" {
				result.Package = fieldInfo.Package
			}
			result.IsImported = fieldInfo.IsImported
		}
	case *ast.ArrayType:
		elInfo, err := readElemType(typedSpec)
		if err == nil {
			result.Kind = elInfo.Kind
			result.IsPointer = elInfo.IsPointer
		}
	case *ast.MapType:
		result.IsMap = true
		result.Kind = "map (temporary)"
	case *ast.ChanType:
		result.IsChan = true
		result.Kind = "chan (temporary)"
	case *ast.StructType:
		result.Kind = "struct (temporary)"
	default:
		//log.Printf("UNHANDLED PTR CASE : %T", typedSpec)
	}
	return &result, err
}

func readSelector(sel *ast.SelectorExpr, comment *ast.CommentGroup) (*FieldInfo, error) {
	var (
		result FieldInfo
		err    error
	)
	if comment != nil {
		// direct call from Preorder
		result.Comment = comment
	}
	result.IsImported = true
	result.Kind = sel.Sel.Name
	if sel.Sel != nil {
		idt, ok := sel.X.(*ast.Ident)
		if ok {
			fieldInfo, err := readIdent(idt, nil)
			if err == nil {
				result.Kind = fieldInfo.Kind + "." + result.Kind
				result.IsBasic = fieldInfo.IsBasic
			}
		} else {
			log.Printf("BAD SELECTOR [NOT IDENT] : %#v", sel)
		}
	} else {
		log.Printf("BAD SELECTOR [NIL SEL]: %#v", sel)
	}
	return &result, err
}

func readField(pkg *types.Package, field *ast.Field, comment *ast.CommentGroup) ([]FieldInfo, error) {
	var (
		oneResult  FieldInfo
		err        error
		fieldNames []string
	)

	oneResult.Comment = comment

	oneResult.Package = pkg.Name()
	oneResult.PackagePath = pkg.Path()
	switch len(field.Names) {
	case 0:
		// embedded field
		oneResult.IsEmbedded = true
	case 1:
		oneResult.IsExported = field.Names[0].IsExported()
		oneResult.Name = field.Names[0].Name
	default:
		// processedField declaration is like `Name, Phone string`
		oneResult.IsExported = field.Names[0].IsExported()
		oneResult.Name = field.Names[0].Name
		for idx, fieldName := range field.Names {
			if idx > 0 {
				fieldNames = append(fieldNames, fieldName.Name)
			}
		}
	}

	switch typedSpec := field.Type.(type) {
	case *ast.Ident:
		fieldInfo, err := readIdent(typedSpec, nil)
		if err == nil {
			oneResult.Kind = fieldInfo.Kind
			oneResult.IsBasic = fieldInfo.IsBasic
		}
	case *ast.StarExpr:
		fieldInfo, err := readPointer(typedSpec, nil)
		if err == nil {
			oneResult.IsPointer = fieldInfo.IsPointer
			oneResult.Kind = fieldInfo.Kind
			oneResult.IsBasic = fieldInfo.IsBasic
			oneResult.IsImported = fieldInfo.IsImported
		}
	case *ast.SelectorExpr:
		fieldInfo, err := readSelector(typedSpec, nil)
		if err == nil {
			oneResult.IsImported = fieldInfo.IsImported
			oneResult.Kind = fieldInfo.Kind
			if fieldInfo.Package != "" {
				oneResult.Package = fieldInfo.Package
			}
			oneResult.IsBasic = fieldInfo.IsBasic
			oneResult.IsImported = fieldInfo.IsImported
		}
	case *ast.MapType:
		oneResult.IsMap = true
		oneResult.Kind = "map (temporary)"
		//log.Printf("%q is map", oneResult.Name)
	case *ast.ChanType:
		oneResult.IsChan = true
		oneResult.Kind = "chan (temporary)"
		// TODO : read chan kind
		//log.Printf("%q is chan", oneResult.Name)
	case *ast.ArrayType:
		oneResult.IsArray = true
		elemTypeInfo, err := readElemType(typedSpec)
		if err == nil {
			oneResult.IsBasic = IsBasic(elemTypeInfo.Kind)
			oneResult.IsPointer = elemTypeInfo.IsPointer
			oneResult.Kind = elemTypeInfo.Kind
		}
		//log.Printf("%q is array : %#v", oneResult.Name, typedSpec)
	case *ast.FuncType:
		oneResult.IsFunc = true
		//log.Printf("%q %s is func", oneResult.Name, oneResult.Kind)
		params, returns, err := readFunc(typedSpec)
		if err == nil {
			oneResult.FuncData = &FunctionInfo{Params: params, Returns: returns}
		}

	case *ast.InterfaceType:
		oneResult.IsInterface = true
		oneResult.Kind = "interface"
	default:
		//log.Printf("UNHANDLED FIELD CASE : %T", typedSpec)
	}

	if field.Tag != nil {
		oneResult.Tags, err = ParseTags(field.Tag.Value)
	}

	result := make([]FieldInfo, 0)
	result = append(result, oneResult)
	if len(fieldNames) > 0 {
		for _, name := range fieldNames {
			result = append(result, oneResult.Clone(name))
		}
	}

	return result, err
}

func readType(pkg *types.Package, astSpec *ast.TypeSpec, comment *ast.CommentGroup) (*TypeInfo, error) {
	var (
		result TypeInfo
		err    error
	)
	result.comment = comment
	if astSpec.Name == nil {
		log.Printf("possible error : astSpec.Name is nil on %#v", astSpec)
		return nil, errors.New("type name is nil")
	}
	result.Package = pkg.Name()
	result.PackagePath = pkg.Path()
	result.Kind = astSpec.Name.Name
	switch typedSpec := astSpec.Type.(type) {
	case *ast.StructType:
		result.Name = result.Kind
		//log.Printf("Traversing struct kind : %q", result.Kind)
		for _, field := range typedSpec.Fields.List {
			fieldsInfo, err := readField(pkg, field, field.Comment)
			if err == nil {
				result.Fields = append(result.Fields, fieldsInfo...)
			}
		}
	case *ast.ArrayType:
		//log.Printf("Traversing array : %q", result.Kind)
		result.Name = astSpec.Name.Name
		result.IsArray = true
		elInfo, err := readElemType(typedSpec)
		if err == nil {
			result.Kind = elInfo.Kind
			result.IsPointer = elInfo.IsPointer
			result.IsImported = elInfo.IsImported
		}
	case *ast.InterfaceType:
		for _, method := range typedSpec.Methods.List {
			fieldInfo, err := readField(pkg, method, nil)
			if err == nil {
				result.Fields = append(result.Fields, fieldInfo...)
			}
		}
	case *ast.FuncType:
		result.Name = result.Kind
		paramsInfo, resultsInfo, err := readFunc(typedSpec)
		if err == nil {
			result.MethodList = append(result.MethodList, FunctionInfo{
				Package:     pkg.Name(),
				PackagePath: pkg.Path(),
				Name:        "func",
				Params:      paramsInfo,
				Returns:     resultsInfo,
				comment:     comment,
				//IsExported:  ???,
			})
		}
	default:
		//log.Printf("UNHANDLED TYPE CASE : %T", typedSpec)
	}
	return &result, err
}

func readElemType(arr *ast.ArrayType) (*TypeInfo, error) {
	var (
		result TypeInfo
		err    error
	)
	switch elType := arr.Elt.(type) {
	case *ast.Ident:
		fieldInfo, err := readIdent(elType, nil)
		if err == nil {
			result.Kind = fieldInfo.Kind
		}
	case *ast.StarExpr:
		fieldInfo, err := readPointer(elType, nil)
		if err == nil {
			result.Kind = fieldInfo.Kind
			result.IsPointer = fieldInfo.IsPointer
			result.IsImported = fieldInfo.IsImported
		}
	case *ast.SelectorExpr:
		fieldInfo, err := readSelector(elType, nil)
		if err == nil {
			result.Kind = fieldInfo.Kind
			result.IsPointer = fieldInfo.IsPointer   // TODO : check
			result.IsImported = fieldInfo.IsImported // TODO : check
			if fieldInfo.Package != "" {
				result.Package = fieldInfo.Package // TODO : check
			}
			if result.PackagePath != "" {
				result.PackagePath = fieldInfo.PackagePath // TODO : check
			}
		}
	case *ast.ArrayType:
		elInfo, err := readElemType(elType)
		if err == nil {
			result.Kind = elInfo.Kind
			result.IsPointer = elInfo.IsPointer
		}
	case *ast.MapType:
		result.Kind = "map (temporary)"
	case *ast.StructType:
		result.Kind = "struct (temporary)"
	case *ast.ChanType:
		result.Kind = "chan (temporary)"
	default:
		//log.Printf("UNHANDLED ELEM CASE : %T", elType)
	}
	return &result, err
}

func readFunc(spec *ast.FuncType) ([]VarInfo, []VarInfo, error) {
	var (
		err error
	)
	var params []VarInfo
	if spec.Params != nil {
		for _, p := range spec.Params.List {
			param := buildVarFromExpr(p)
			if param.Kind == "" && param.Name == "" {
				log.Printf("noname/notype found while inspecting params of %#v", spec.Params.List)
				continue
			}
			params = append(params, param)
		}
	}
	var returns []VarInfo
	if spec.Results != nil {
		for _, r := range spec.Results.List {
			param := buildVarFromExpr(r)
			if param.Kind == "" && param.Name == "" {
				log.Printf("noname/notype found while inspecting returns of %#v", spec.Results.List)
				continue
			}
			returns = append(returns, param)
		}
	}
	return params, returns, err
}

// get function information from the function object
func readFuncDecl(spec *ast.FuncDecl) (*FunctionInfo, error) {
	if spec.Name == nil {
		return nil, errors.New("spec name is nil while reading function")
	}
	info := FunctionInfo{
		Name:       spec.Name.Name,
		IsExported: spec.Name.IsExported(),
		comment:    spec.Doc,
	}
	if spec.Recv != nil {
		if len(spec.Recv.List) > 0 {
			stList := spec.Recv.List[0]
			if len(stList.Names) > 0 {
				info.ReceiverName = stList.Names[0].Name
			}
			switch recType := stList.Type.(type) {
			case *ast.Ident:
				info.ReceiverType = recType.Name
			case *ast.StarExpr:
				info.IsMethodReceiver = true
				info.ReceiverType = recType.X.(*ast.Ident).Name
			default:
				//log.Printf("don't know what to do with receiver : %T %#v", recType, recType)
			}
		}
	}
	params, returns, err := readFunc(spec.Type)
	if err == nil {
		info.Params = params
		info.Returns = returns
	}
	return &info, nil
}

func buildVarFromExpr(field *ast.Field) VarInfo {
	var param VarInfo
	if len(field.Names) == 1 {
		param.Name = field.Names[0].Name //  it has a name
	}
	switch paramType := field.Type.(type) {
	case *ast.Ident:
		fieldInfo, err := readIdent(paramType, nil)
		if err == nil {
			param.Kind = fieldInfo.Kind
		}
	case *ast.StarExpr:
		fieldInfo, err := readPointer(paramType, nil)
		if err == nil {
			param.Kind = fieldInfo.Kind
		}
	case *ast.SelectorExpr:
		fieldInfo, err := readSelector(paramType, nil)
		if err == nil {
			param.Kind = fieldInfo.Kind // TODO : check
		}
	case *ast.ArrayType:
		typeInfo, err := readElemType(paramType)
		if err == nil {
			param.Kind = "[]" + typeInfo.Kind
		}
	case *ast.InterfaceType:
		param.Kind = "interface"
	case *ast.FuncType:
		param.Kind = "function"
	case *ast.MapType:
		param.Kind = "map"
	default:
		log.Printf("UNHANDLED PARAM/RETURN CASE : %T", paramType)
	}
	return param
}

func readValue(defObj types.Object, valueSpec *ast.ValueSpec) ([]VarInfo, error) {
	var result []VarInfo
	for _, varName := range valueSpec.Names {
		var name string
		switch defObj.(type) {
		case *types.Var:
			name = defObj.(*types.Var).Type().String()
		case *types.Const:
			name = defObj.(*types.Const).Type().String()
		}
		result = append(result, VarInfo{
			Name: varName.Name,
			Kind: name,
		})
	}
	return result, nil
}
