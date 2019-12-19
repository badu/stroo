package stroo

import (
	"fmt"
	"go/ast"
	"go/types"
	"log"
	"strings"
)

var (
	ErrNotImplemented = "%T found on %q (not implemented)"
)

type PackageInfo struct {
	Name       string
	Vars       Vars
	Functions  Methods
	Types      TypesSlice
	Interfaces Interfaces
	StructDefs map[string]*TypeInfo
	ArrayDefs  map[string]*FieldInfo
}

func (pkg *PackageInfo) GetStructByKey(key string) *TypeInfo {
	structInfo, ok := pkg.StructDefs[key]
	if ok {
		return structInfo
	}
	return nil
}

func (pkg *PackageInfo) PostProcess() error {
	// solve receivers to their structs
	var result Methods
	for _, fn := range pkg.Functions {
		if fn.ReceiverType == "" {
			// todo test
			log.Fatalf("receiverType is empty : %#v", fn)
			continue
		}

		// look into structs
		structInfo, found := pkg.StructDefs[fn.ReceiverType]
		if found {
			structInfo.MethodList = append(structInfo.MethodList, fn)
			continue
		}

		// look in arrays
		arrayInfo, found := pkg.ArrayDefs[fn.ReceiverType]
		if found {
			arrayInfo.MethodList = append(arrayInfo.MethodList, fn)
			continue
		}

		// normal function
		// todo test
		// log.Printf("function : %#v", fn)
		result = append(result, fn)
	}
	pkg.Functions = result

	// fix array references
	for _, arrValue := range pkg.ArrayDefs {
		structInfo, found := pkg.StructDefs[arrValue.TypeName]
		if found {
			arrValue.Reference = structInfo
			continue
		}
		log.Fatalf("Not found %q array data :\n %#v", arrValue.TypeName, arrValue)
	}

	// create references
	for _, typeDef := range pkg.Types {
		for _, field := range typeDef.Fields {
			if field.IsStruct {
				// look into structs
				structInfo, found := pkg.StructDefs[field.TypeName]
				if found {
					field.Reference = structInfo
					continue
				}

				// look in arrays
				arrayInfo, found := pkg.ArrayDefs[field.TypeName]
				if found {
					field.FromArray(arrayInfo)
					continue
				}

				log.Fatalf("Not found %q isArray: %t -> array data :\n %#v", field.TypeName, field.IsArray, field)
			}

		}
	}
	return nil
}

// reads array type declaration
// currently, we accept the following definitions :
// type T []V - where V is a basic type or a typed struct
// type T []*V - where V is a basic type or a typed struct
func (pkg *PackageInfo) ReadArrayInfo(spec ast.Spec, obj types.Object, comment *ast.CommentGroup) error {
	astSpec := spec.(*ast.TypeSpec)
	info := FieldInfo{
		Name:    astSpec.Name.Name,
		Comment: comment,
		IsArray: true,
	}
	if obj != nil {
		info.Package = obj.Pkg().Name()
		info.PackagePath = obj.Pkg().Path()
	}
	switch elType := astSpec.Type.(*ast.ArrayType).Elt.(type) {
	case *ast.Ident:
		info.TypeName = elType.Name
		if elType.Obj == nil {
			info.IsBasic = true
		}
	case *ast.StarExpr:
		info.IsPointer = true
		switch ptrType := elType.X.(type) {
		case *ast.Ident:
			if ptrType.Obj == nil {
				info.IsBasic = true
			}
			info.TypeName = ptrType.Name
		default:
			return fmt.Errorf(ErrNotImplemented, elType, info.Name)
		}
	default:
		return fmt.Errorf(ErrNotImplemented, elType, info.Name)
	}
	pkg.ArrayDefs[info.Name] = &info
	return nil
}

// get struct information from structure type declaration
func (pkg *PackageInfo) ReadStructInfo(spec ast.Spec, obj types.Object, comment *ast.CommentGroup) error {
	astSpec := spec.(*ast.TypeSpec)
	info := TypeInfo{
		Name:    astSpec.Name.Name,
		Comment: comment,
	}
	if obj != nil {
		info.Package = obj.Pkg().Name()
		info.PackagePath = obj.Pkg().Path()
	}
	for _, field := range astSpec.Type.(*ast.StructType).Fields.List {
		if len(field.Names) == 0 {
			// embedded field
			embeddedField := FieldInfo{IsEmbedded: true}
			switch fieldType := field.Type.(type) {
			case *ast.StarExpr:
				embeddedField.IsPointer = true
				switch ptrType := fieldType.X.(type) {
				case *ast.Ident:
					if ptrType.Obj == nil {
						embeddedField.IsBasic = true
					} else {
						embeddedField.IsStruct = true
					}
					embeddedField.TypeName = ptrType.Name
				default:
					// not allowed
				}
			case *ast.Ident:
				if fieldType.Obj == nil {
					embeddedField.IsBasic = true
				} else {
					embeddedField.IsStruct = true
				}
				embeddedField.TypeName = fieldType.Name
			default:
				// not allowed
				return fmt.Errorf(ErrNotImplemented, fieldType, info.Name)
			}
			info.Fields = append(info.Fields, &embeddedField)
			continue
		}
		newField := FieldInfo{Name: field.Names[0].Name, IsExported: field.Names[0].IsExported(), Comment: field.Comment}
		if false {
			log.Printf("[%q] Name %q- exported : %t", info.Name, field.Names[0].Name, field.Names[0].IsExported())
			log.Printf("[%q] Type %#v", info.Name, field.Type)
			if field.Tag != nil {
				log.Printf("[%q] Tag %#v", info.Name, field.Tag.Value)
			}
			if field.Doc != nil {
				log.Printf("Doc %#v", field.Doc)
			}
			if field.Comment != nil {
				log.Printf("Comment %#v", field.Comment)
			}
		}
		switch fieldType := field.Type.(type) {
		case *ast.StarExpr:
			newField.IsPointer = true
			switch ptrType := fieldType.X.(type) {
			case *ast.Ident:
				newField.TypeName = ptrType.Name
				if ptrType.Obj == nil {
					newField.IsBasic = true
				} else {
					newField.IsStruct = true
				}
			default:
				return fmt.Errorf(ErrNotImplemented, ptrType, info.Name)
			}
		case *ast.Ident:
			newField.TypeName = fieldType.Name
			if fieldType.Obj == nil {
				newField.IsBasic = true
			} else {
				newField.IsStruct = true
			}
		case *ast.MapType:
			newField.IsMap = true
		case *ast.ChanType:
			newField.IsChan = true
		default:
			return fmt.Errorf(ErrNotImplemented, fieldType, info.Name)
		}

		if field.Tag != nil {
			var err error
			newField.Tags, err = ParseTags(field.Tag.Value)
			if err != nil {
				return fmt.Errorf("error parsing tags : %v of field named %q input = %s", err.Error(), newField.Name, field.Tag.Value)
			}
		}
		info.Fields = append(info.Fields, &newField)
	}
	if _, has := pkg.StructDefs[info.Name]; has {
		return fmt.Errorf("%q was already declared. shouldn't happen", info.Name)
	}
	pkg.StructDefs[info.Name] = &info
	pkg.Types = append(pkg.Types, &info)
	return nil
}

// get function information from the function object
func (pkg *PackageInfo) ReadFunctionInfo(spec *ast.FuncDecl, obj types.Object) {
	info := FunctionInfo{
		Name:         spec.Name.Name,
		ReceiverName: stringer(getReceiver(spec)),
		ReceiverType: stringer(getReceiverType(spec)),
		IsExported:   spec.Name.IsExported(),
	}
	if obj != nil {
		typ := obj.(*types.Func)
		info.Package = typ.Pkg().Name()
		info.PackagePath = typ.Pkg().Path()
		info.ReturnType = typ.Type().String()
	}
	pkg.Functions = append(pkg.Functions, &info)
}

// get interface information from interface type declaration
func (pkg *PackageInfo) ReadInterfaceInfo(spec ast.Spec, obj types.Object, comment *ast.CommentGroup) {
	//var list []InterfaceInfo
	astSpec := spec.(*ast.TypeSpec)
	var methods []string
	for _, m := range astSpec.Type.(*ast.InterfaceType).Methods.List {
		if len(m.Names) == 0 {
			continue
		}
		methods = append(methods, m.Names[0].Name)
	}
	info := InterfaceInfo{
		Name:    astSpec.Name.Name,
		Methods: methods,
		Comment: comment,
	}
	if obj != nil {
		info.Package = obj.Pkg().Name()
		info.PackagePath = obj.Pkg().Path()
	}
	pkg.Interfaces = append(pkg.Interfaces, &info)
}

func (pkg *PackageInfo) ReadVariablesInfo(spec *ast.ValueSpec, obj types.Object) {
	if obj == nil {
		return
	}
	for _, varName := range spec.Names {
		var name string
		switch obj.(type) {
		case *types.Var:
			name = obj.(*types.Var).Type().String()
		case *types.Const:
			name = obj.(*types.Const).Type().String()
		}
		info := VarInfo{
			Name: varName.Name,
			Type: name,
		}
		pkg.Vars = append(pkg.Vars, info)
	}
}

// cannot implement Stringer due to tests
func (pkg PackageInfo) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
		tno++
	}
	sb.WriteString(tabs + "Package : \"" + pkg.Name + "\"\n")
	if tno > 0 {
		pkg.Interfaces.Debug(sb, tno)
		pkg.Types.Debug(sb, tno)
		pkg.Functions.Debug(sb, tno)
		pkg.Vars.Debug(sb, tno)
	} else {
		pkg.Interfaces.Debug(sb)
		pkg.Types.Debug(sb)
		pkg.Functions.Debug(sb)
		pkg.Vars.Debug(sb)
	}
}
