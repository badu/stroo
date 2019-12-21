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
	FieldsDefs map[string]*FieldInfo
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
		arrayInfo, found := pkg.FieldsDefs[fn.ReceiverType]
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
	// fix structs that are actually arrays + set the references
	for _, typeDef := range pkg.Types {
		for _, field := range typeDef.Fields {
			if field.IsStruct {
				if refInfo, found := pkg.StructDefs[field.TypeName]; found {
					field.Reference = refInfo
					field.IsArray = false
					continue
				}
				if arrayInfo, found := pkg.FieldsDefs[field.TypeName]; found {
					// skip basic fields
					if arrayInfo.IsBasic {
						field.IsStruct = false
						field.IsArray = true
						continue
					}
					if refInfo, arrayFound := pkg.StructDefs[arrayInfo.TypeName]; arrayFound {
						field.IsStruct = false
						field.IsArray = true
						arrayInfo.Reference = refInfo
						field.Reference = refInfo        // the real reference to the struct
						field.ArrayReference = arrayInfo // set it in case it's needed
						continue
					}
					log.Fatalf("Struct not found %q:%q -> data :\n %#v", field.Name, arrayInfo.TypeName, arrayInfo)

				}
				log.Fatalf("Struct not found %q:%q -> data :\n %#v", field.Name, field.TypeName, field)
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
	if err := pkg.ReadArray(astSpec.Type.(*ast.ArrayType), &info); err != nil {
		return err
	}
	if _, has := pkg.FieldsDefs[info.Name]; has {
		return fmt.Errorf("error : array %q already defined", info.Name)
	}
	pkg.FieldsDefs[info.Name] = &info
	return nil
}

func (pkg *PackageInfo) ReadIdent(ident *ast.Ident, info *FieldInfo) {
	info.TypeName = ident.Name
	if ident.Obj == nil {
		info.IsBasic = true
	} else {
		info.IsStruct = true
	}
}

func (pkg *PackageInfo) ReadPointer(ptr *ast.StarExpr, info *FieldInfo) error {
	info.IsPointer = true
	switch ptrType := ptr.X.(type) {
	case *ast.Ident:
		pkg.ReadIdent(ptrType, info)
	default:
		return fmt.Errorf(ErrNotImplemented, ptr, info.Name)
	}
	return nil
}

func (pkg *PackageInfo) ReadArray(arr *ast.ArrayType, info *FieldInfo) error {
	info.IsArray = true
	switch elType := arr.Elt.(type) {
	case *ast.Ident:
		pkg.ReadIdent(elType, info)
	case *ast.StarExpr:
		if err := pkg.ReadPointer(elType, info); err != nil {
			return err
		}
	default:
		return fmt.Errorf(ErrNotImplemented, elType, info.Name)
	}
	return nil
}

// get struct information from structure type declaration
func (pkg *PackageInfo) ReadStructInfo(spec ast.Spec, obj types.Object, comment *ast.CommentGroup) error {
	astSpec := spec.(*ast.TypeSpec)
	info := TypeInfo{
		Name:     astSpec.Name.Name,
		TypeName: astSpec.Name.Name,
		Comment:  comment,
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
				if err := pkg.ReadPointer(fieldType, &embeddedField); err != nil {
					return err
				}
			case *ast.Ident:
				pkg.ReadIdent(fieldType, &embeddedField)
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
			if err := pkg.ReadPointer(fieldType, &newField); err != nil {
				return err
			}
		case *ast.Ident:
			pkg.ReadIdent(fieldType, &newField)
		case *ast.MapType:
			newField.IsMap = true
		case *ast.ChanType:
			newField.IsChan = true
		case *ast.ArrayType:
			if err := pkg.ReadArray(fieldType, &newField); err != nil {
				return err
			}
			if _, has := pkg.FieldsDefs[newField.Name]; has {
				return fmt.Errorf("error : array %q already defined", newField.Name)
			}
			pkg.FieldsDefs[newField.Name] = &newField
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
