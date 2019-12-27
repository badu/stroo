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
	TypesInfo  *types.Info
	PrintDebug bool
}

func (pkg *PackageInfo) GetStructByKey(key string) *TypeInfo {
	structInfo, ok := pkg.StructDefs[key]
	if ok {
		return structInfo
	}
	if pkg.PrintDebug {
		log.Printf("GetStructByKey - %q NOT found", key)
	}
	return nil
}

func (pkg *PackageInfo) PostProcess() error {
	// solve receivers to their structs
	var result Methods
	for _, fn := range pkg.Functions {
		if fn.ReceiverType == "" {
			// todo test
			if pkg.PrintDebug {
				log.Printf("receiverType is empty : %#v", fn)
			}
			continue
		}

		// look into structs
		structInfo, found := pkg.StructDefs[fn.ReceiverType]
		if found {
			structInfo.MethodList = append(structInfo.MethodList, fn)
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
			if field.IsStruct && !field.IsImported {
				if refInfo, found := pkg.StructDefs[field.Kind]; found {
					field.Reference = refInfo
					field.IsArray = false
					continue
				}
				log.Fatalf("Struct not found %q:%q [%q:%q] -> data :\n %#v", field.Name, field.Kind, field.Package, field.PackagePath, field)
			}
		}
	}
	return nil
}

// reads array type declaration
// currently, we accept the following definitions :
// type T []V - where V is a basic type or a typed struct
// type T []*V - where V is a basic type or a typed struct
func (pkg *PackageInfo) ReadArrayInfo(astSpec *ast.TypeSpec, comment *ast.CommentGroup) error {
	info := TypeInfo{
		Name:    astSpec.Name.Name,
		Comment: comment,
		IsArray: true,
	}
	if pkg.PrintDebug {
		log.Printf("==Traversing array %q==", astSpec.Name.Name)
	}
	found := false
	for key, value := range pkg.TypesInfo.Defs {
		if key.Name == astSpec.Name.Name {
			info.Kind = value.Name()
			switch value.Type().Underlying().(type) {
			case *types.Slice:
				info.IsArray = true
			case *types.Pointer:
				info.IsPointer = true
			default:
				log.Fatalf("Array Lookup unknown %q -> exported %t; type %#v; package= %v; id=%q", value.Name(), value.Exported(), value.Type().Underlying(), value.Pkg(), value.Id())
			}
			found = true
			break
		}
	}
	if !found {
		log.Fatalf("%q not found while processing array %#v", astSpec.Name.Name, info)
	}

	//if err := pkg.ReadArray(astSpec.Type.(*ast.ArrayType), &info, comment); err != nil {
	//	return err
	//}
	if value, has := pkg.StructDefs[info.Name]; has {
		return fmt.Errorf("error : array %q already defined\n%#v", info.Name, value)
	}
	pkg.StructDefs[info.Name] = &info
	return nil
}

/**
There are eight kinds of objects in the Go type checker :
Object = *Func         // function, concrete method, or abstract method
       | *Var          // variable, parameter, result, or struct field
       | *Const        // constant
       | *TypeName     // type name
       | *Label        // statement label
       | *PkgName      // package name, e.g. json after import "encoding/json"
       | *Builtin      // predeclared function such as append or len
       | *Nil          // predeclared nil

type Object interface {
	Name() string   // package-local object name
	Exported() bool // reports whether the name starts with a capital letter
	Type() Type     // object type
	Pos() token.Pos // position of object identifier in declaration

	Parent() *Scope // scope in which this object is declared
	Pkg() *Package  // nil for objects in the Universe scope and labels
	Id() string     // object id (see Ids section below)
}

func (*Func) Scope() *Scope
func (*Var) Anonymous() bool
func (*Var) IsField() bool
func (*Const) Val() constant.Value
func (*TypeName) IsAlias() bool
func (*PkgName) Imported() *Package

Type = *Basic
     | *Pointer
     | *Array
     | *Slice
     | *Map
     | *Chan
     | *Struct
     | *Tuple
     | *Signature
     | *Named
     | *Interface
*/
func (pkg *PackageInfo) Lookup(named string, info *FieldInfo) types.Object {
	for key, value := range pkg.TypesInfo.Defs {
		if key.Name == named {
			info.Kind = value.Name()
			switch value.Type().Underlying().(type) {
			case *types.Pointer:
				info.IsPointer = true
			case *types.Struct:
				info.IsStruct = true
			case *types.Slice:
				info.IsArray = true
			case *types.Interface:
				info.IsInterface = true
			default:
				log.Fatalf("Lookup unknown %q -> exported %t; type %#v; package= %v; id=%q", value.Name(), value.Exported(), value.Type().Underlying(), value.Pkg(), value.Id())
			}
			return value
		}
	}
	return nil
}

func (pkg *PackageInfo) ReadIdent(ident *ast.Ident, info *FieldInfo, comment *ast.CommentGroup) {
	if info == nil {
		// TODO : comment + test
		info = &FieldInfo{Name: pkg.TypesInfo.Defs[ident].Name(), Comment: comment}
		if pkg.PrintDebug {
			log.Printf("Self Adding %q", info.Name)
		}
		panic("not implemented")
	}

	switch ident.Name {
	case "bool", "int", "int8", "int16", "int32", "rune", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "complex64", "complex128", "string":
		info.IsBasic = true
		info.Kind = ident.Name
	default:
		obj := pkg.Lookup(ident.Name, info)
		if obj == nil {
			//log.Fatalf("%q not found while processing ident %#v", ident.Name, info)
		}
	}

	if pkg.PrintDebug {
		if !info.IsBasic {
			log.Printf("[%q] kind = %q pointer = %t basic = %t struct = %t array = %t package %q", info.Name, info.Kind, info.IsPointer, info.IsBasic, info.IsStruct, info.IsArray, info.Package)
		}
	}

}

func (pkg *PackageInfo) ReadSelector(sel *ast.SelectorExpr, info *FieldInfo, comment *ast.CommentGroup) {
	info.IsImported = true
	if ident, ok := sel.X.(*ast.Ident); ok {
		info.Package = ident.Name
		info.Kind = sel.Sel.Name
		if pkg.PrintDebug {
			log.Printf("SelectorExpr : %q.%q", info.Package, info.Kind)
		}
		pkg.ReadIdent(ident, info, comment)
	} else {
		log.Fatalf("*ast.SelectorExpr.X is not *ast.Ident")
	}
}

func (pkg *PackageInfo) ReadPointer(ptr *ast.StarExpr, info *FieldInfo, comment *ast.CommentGroup) error {
	if pkg.PrintDebug {
		log.Printf("%q is pointer", info.Name)
	}
	info.IsPointer = true
	switch ptrType := ptr.X.(type) {
	case *ast.Ident:
		pkg.ReadIdent(ptrType, info, comment)
	case *ast.SelectorExpr:
		// fields from other packages
		pkg.ReadSelector(ptrType, info, comment)
	default:
		return fmt.Errorf(ErrNotImplemented, ptr, info.Name)
	}
	return nil
}

// TODO : not used
func (pkg *PackageInfo) ReadArray(arr *ast.ArrayType, info *FieldInfo, comment *ast.CommentGroup) error {
	info.IsArray = true
	switch elType := arr.Elt.(type) {
	case *ast.Ident:
		pkg.ReadIdent(elType, info, comment)
	case *ast.StarExpr:
		if err := pkg.ReadPointer(elType, info, comment); err != nil {
			return err
		}
	default:
		return fmt.Errorf(ErrNotImplemented, elType, info.Name)
	}
	if pkg.PrintDebug {
		log.Printf("[%q] ReadArray : array = %t", info.Name, info.IsArray)
	}
	return nil
}

// get struct information from structure type declaration
func (pkg *PackageInfo) ReadStructInfo(astSpec *ast.TypeSpec, comment *ast.CommentGroup) error {
	info := TypeInfo{
		Name:    astSpec.Name.Name,
		Kind:    astSpec.Name.Name,
		Comment: comment,
	}
	if pkg.PrintDebug {
		log.Printf("==Traversing struct %q==", astSpec.Name.Name)
	}
	obj, found := pkg.TypesInfo.Defs[astSpec.Name]
	if found {
		info.Package = obj.Pkg().Name()
		info.PackagePath = obj.Pkg().Path()
	} else {
		log.Printf("%q not found in TypesInfo.Defs", astSpec.Name)
	}
	for _, field := range astSpec.Type.(*ast.StructType).Fields.List {
		if len(field.Names) == 0 {
			// embedded field
			embeddedField := FieldInfo{IsEmbedded: true}
			switch fieldType := field.Type.(type) {
			case *ast.StarExpr:
				if err := pkg.ReadPointer(fieldType, &embeddedField, nil); err != nil {
					return err
				}
			case *ast.Ident:
				pkg.ReadIdent(fieldType, &embeddedField, nil)
			default:
				// not allowed
				return fmt.Errorf(ErrNotImplemented, fieldType, info.Name)
			}
			info.Fields = append(info.Fields, &embeddedField)
			continue
		}
		newField := FieldInfo{Name: field.Names[0].Name, IsExported: field.Names[0].IsExported(), Comment: field.Comment}
		if pkg.PrintDebug {
			log.Printf("inspecting %q %T", newField.Name, field.Type)
		}
		switch fieldType := field.Type.(type) {
		case *ast.StarExpr:
			if err := pkg.ReadPointer(fieldType, &newField, nil); err != nil {
				return err
			}
		case *ast.Ident:
			pkg.ReadIdent(fieldType, &newField, nil)
		case *ast.MapType:
			newField.IsMap = true
		case *ast.ChanType:
			newField.IsChan = true
		case *ast.ArrayType:
			newField.IsArray = true
		case *ast.SelectorExpr:
			// fields from other packages
			pkg.ReadSelector(fieldType, &newField, nil)
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
func (pkg *PackageInfo) ReadFunctionInfo(spec *ast.FuncDecl) {

	info := FunctionInfo{
		Name:         spec.Name.Name,
		ReceiverName: stringer(getReceiver(spec)),
		ReceiverType: stringer(getReceiverType(spec)),
		IsExported:   spec.Name.IsExported(),
	}
	obj := pkg.TypesInfo.Defs[spec.Name]
	if obj != nil {
		typ := obj.(*types.Func)
		info.Package = typ.Pkg().Name()
		info.PackagePath = typ.Pkg().Path()
		info.ReturnType = typ.Type().String()
	} else if pkg.PrintDebug {
		log.Printf("ReadFunctionInfo : obj is nil")
	}
	pkg.Functions = append(pkg.Functions, &info)
}

// get interface information from interface type declaration
func (pkg *PackageInfo) ReadInterfaceInfo(spec ast.Spec, comment *ast.CommentGroup) {
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
	obj := pkg.TypesInfo.Defs[astSpec.Name]
	if obj != nil {
		info.Package = obj.Pkg().Name()
		info.PackagePath = obj.Pkg().Path()
	}
	pkg.Interfaces = append(pkg.Interfaces, &info)
}

func (pkg *PackageInfo) ReadVariablesInfo(spec ast.Spec, valueSpec *ast.ValueSpec) {
	obj := pkg.TypesInfo.Defs[spec.(*ast.ValueSpec).Names[0]]
	if obj == nil {
		return
	}
	for _, varName := range valueSpec.Names {
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
