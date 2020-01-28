package stroo

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"log"
)

var (
	ErrNotImplemented = "%T found on %q (not implemented)"
)

type Imports struct {
	Name string
	Path string
}
type PackageInfo struct {
	Name       string
	Vars       Vars
	Functions  Methods
	Types      TypesSlice
	Interfaces Interfaces
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
	obj, found := pkg.TypesInfo.Defs[astSpec.Name]
	if found {
		info.Package = obj.Pkg().Name()
		info.PackagePath = obj.Pkg().Path()
	} else {
		return fmt.Errorf("%q not found in TypesInfo.Defs", astSpec.Name)
	}
	// TODO : take comments from both astSpec.Comment.Text(), comment.Text()
	switch elType := astSpec.Type.(*ast.ArrayType).Elt.(type) {
	case *ast.Ident:
		info.Kind = elType.Name
	case *ast.StarExpr:
		info.IsPointer = true
		switch ptrType := elType.X.(type) {
		case *ast.Ident:
			info.Kind = ptrType.Name
		case *ast.SelectorExpr:
			if ident, ok := ptrType.X.(*ast.Ident); ok {
				info.Kind = ident.Name + "." + ptrType.Sel.Name
			}
		default:
			return fmt.Errorf(ErrNotImplemented, elType, info.Name)
		}
	case *ast.SelectorExpr:
		// fields from other packages
		if ident, ok := elType.X.(*ast.Ident); ok {
			info.Kind = ident.Name + "." + elType.Sel.Name
			//TODO : if dotindex := strings.LastIndex(nameStr, "."); dotindex >= 0 {
		}
	default:
		return fmt.Errorf(ErrNotImplemented, elType, info.Name)
	}
	if pkg.PrintDebug {
		log.Printf("ReadArrayInfo Traversing array %q of type %q==", info.Name, info.Kind)
	}
	pkg.Types = append(pkg.Types, &info)
	return nil
}

func (pkg *PackageInfo) FieldLookupAndFill(named string, info *FieldInfo) (types.Object, error) {
	for key, value := range pkg.TypesInfo.Defs {
		if key.Name == named {
			info.Kind = value.Name()
			switch underType := value.Type().Underlying().(type) {
			case *types.Pointer:
				info.IsPointer = true
				switch underPtr := underType.Elem().Underlying().(type) {
				default:
					return nil, fmt.Errorf("pointer lookup : don't know what to do with : %#v", underPtr)
				case *types.Struct:
					info.IsStruct = true
				case *types.Slice:
					info.IsArray = true
				case *types.Interface:
					info.IsInterface = true
				}
			case *types.Struct:
				info.IsStruct = true
			case *types.Slice:
				info.IsArray = true
			case *types.Interface:
				info.IsInterface = true
			default:
				return nil, fmt.Errorf("lookup unknown %q -> exported %t; type %#v; package= %v; id=%q", value.Name(), value.Exported(), value.Type().Underlying(), value.Pkg(), value.Id())
			}
			return value, nil
		}
	}
	return nil, fmt.Errorf("lookup for %q failed", named)
}

func (pkg *PackageInfo) DirectIdent(ident *ast.Ident, comment *ast.CommentGroup) error {
	newInfo := TypeInfo{Comment: comment}
	found := false
	switch ident.Name {
	case "bool", "int", "int8", "int16", "int32", "rune", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64", "complex64", "complex128", "string":
		newInfo.Kind = ident.Name
		newInfo.Name = ident.Name
		found = true
	default:
		for key, value := range pkg.TypesInfo.Defs {
			if key.Name == ident.Name {
				found = true
				newInfo.Name = value.Name()
				switch underType := value.Type().Underlying().(type) {
				case *types.Pointer:
					newInfo.IsPointer = true
					switch underPtr := underType.Elem().Underlying().(type) {
					default:
						return fmt.Errorf("pointer lookup : don't know what to do with : %#v", underPtr)
					case *types.Slice:
						newInfo.IsArray = true
					}
				case *types.Slice:
					newInfo.IsArray = true
				default:
					return fmt.Errorf("lookup unknown %q -> exported %t; type %#v; package= %v; id=%q", value.Name(), value.Exported(), value.Type().Underlying(), value.Pkg(), value.Id())
				}
				if pkg.PrintDebug {
					log.Printf("Self adding : %#v", newInfo)
				}
				break
			}
		}
	}

	if found {
		pkg.Types = append(pkg.Types, &newInfo)
	}
	if pkg.PrintDebug {
		log.Printf("%q not found in TypesInfo.Defs", ident.Name)
	}
	return nil
}

func (pkg *PackageInfo) DirectSelector(sel *ast.SelectorExpr, comment *ast.CommentGroup) error {
	if ident, ok := sel.X.(*ast.Ident); ok {
		newInfo := TypeInfo{Comment: comment}
		newInfo.Package = ident.Name
		newInfo.Kind = sel.Sel.Name
		newInfo.Name = sel.Sel.Name
		pkg.Types = append(pkg.Types, &newInfo)
	}
	return nil
}

func (pkg *PackageInfo) DirectPointer(ptr *ast.StarExpr, comment *ast.CommentGroup) error {
	newInfo := TypeInfo{Comment: comment}
	newInfo.IsPointer = true
	switch ptrType := ptr.X.(type) {
	case *ast.Ident:
		found := false
		switch ptrType.Name {
		case "bool", "int", "int8", "int16", "int32", "rune", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64", "complex64", "complex128", "string":
			newInfo.Kind = ptrType.Name
			newInfo.Name = ptrType.Name
			found = true
		default:
			for key, value := range pkg.TypesInfo.Defs {
				if key.Name == ptrType.Name {
					found = true
					newInfo.Name = value.Name()
					switch underType := value.Type().Underlying().(type) {
					case *types.Pointer:
						newInfo.IsPointer = true
						switch underPtr := underType.Elem().Underlying().(type) {
						default:
							return fmt.Errorf("pointer lookup : don't know what to do with : %#v", underPtr)
						case *types.Slice:
							newInfo.IsArray = true
						}
					case *types.Slice:
						newInfo.IsArray = true
					default:
						return fmt.Errorf("lookup unknown %q -> exported %t; type %#v; package= %v; id=%q", value.Name(), value.Exported(), value.Type().Underlying(), value.Pkg(), value.Id())
					}
					if pkg.PrintDebug {
						log.Printf("Self adding : %#v", newInfo)
					}
					break
				}
			}
		}
		if found {
			pkg.Types = append(pkg.Types, &newInfo)
		}
	case *ast.SelectorExpr:
		// fields from other packages
		if ident, ok := ptrType.X.(*ast.Ident); ok {
			newInfo.Package = ident.Name
			newInfo.Kind = ptrType.Sel.Name
			newInfo.Name = ptrType.Sel.Name
			pkg.Types = append(pkg.Types, &newInfo)
		}
	default:
		log.Printf("unknown pointer type %T", ptrType)
	}
	return nil
}

func (pkg *PackageInfo) ReadIdent(ident *ast.Ident, info *FieldInfo, comment *ast.CommentGroup) error {
	switch ident.Name {
	case "bool", "int", "int8", "int16", "int32", "rune", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64", "complex64", "complex128", "string":
		info.IsBasic = true
		info.Kind = ident.Name
	case "chan":
		return fmt.Errorf("chan not implemented")
	default:
		if info.IsImported {
			return nil
		}
		info.Kind = ident.Name
		obj, err := pkg.FieldLookupAndFill(ident.Name, info)
		if obj == nil {
			log.Printf("[error] in %q while processing ident : %v", info.Name, err)
			return fmt.Errorf("%q not found while processing ident %#v", ident.Name, info)
		} else if err != nil {
			log.Printf("[error] in %q while processing ident not found : %v", info.Name, err)
			return fmt.Errorf("%q processing ident with error : %v", ident.Name, err)
		}
	}

	if pkg.PrintDebug {
		if !info.IsBasic {
			log.Printf("[%q] kind = %q pointer = %t basic = %t struct = %t array = %t package %q", info.Name, info.Kind, info.IsPointer, info.IsBasic, info.IsStruct, info.IsArray, info.Package)
		}
	}
	return nil
}

func (pkg *PackageInfo) ReadSelector(sel *ast.SelectorExpr, info *FieldInfo, comment *ast.CommentGroup) error {
	info.IsImported = true
	if ident, ok := sel.X.(*ast.Ident); ok {
		info.Package = ident.Name
		info.Kind = sel.Sel.Name
		if pkg.PrintDebug {
			log.Printf("SelectorExpr : %q.%q", info.Package, info.Kind)
		}
		if err := pkg.ReadIdent(ident, info, comment); err != nil {
			log.Printf("[error] in reading selector %q while reading ident : %v", info.Name, err)
			return fmt.Errorf("error reading selector : %v", err)
		}
	} else {
		log.Printf("Possible error in %q : *ast.SelectorExpr.X is not *ast.Ident : %#v", info.Name, sel)
	}
	return nil
}

func (pkg *PackageInfo) ReadPointer(ptr *ast.StarExpr, info *FieldInfo, comment *ast.CommentGroup) error {
	if pkg.PrintDebug && info.Name != "" {
		log.Printf("%q is pointer", info.Name)
	}
	info.IsPointer = true
	switch ptrType := ptr.X.(type) {
	case *ast.Ident:
		if err := pkg.ReadIdent(ptrType, info, comment); err != nil {
			log.Printf("[error] in reading pointer %q while reading ident : %v", info.Name, err)
			return fmt.Errorf("error reading selector : %v", err)
		}
	case *ast.SelectorExpr:
		// fields from other packages
		if err := pkg.ReadSelector(ptrType, info, comment); err != nil {
			log.Printf("[error] in reading pointer %q while reading selector : %v", info.Name, err)
			return fmt.Errorf("error reading selector : %v", err)
		}
	default:
		log.Printf("unknown pointer %q of type %T", info.Kind, ptrType)
		return fmt.Errorf(ErrNotImplemented, ptr, info.Name)
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
		log.Printf("%q: Traversing struct ", astSpec.Name.Name)
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
			// TODO : note that we need a checker `types.Check` to establish that the field is exported. Forcing exported default
			embeddedField := FieldInfo{IsEmbedded: true, IsExported: true}
			switch fieldType := field.Type.(type) {
			case *ast.StarExpr:
				if err := pkg.ReadPointer(fieldType, &embeddedField, nil); err != nil {
					log.Printf("[error] in %q while reading embed pointer : %v", info.Name, err)
					return err
				}
			case *ast.Ident:
				if err := pkg.ReadIdent(fieldType, &embeddedField, nil); err != nil {
					log.Printf("[error] in %q while reading embed ident : %v", info.Name, err)
					return fmt.Errorf("error reading ident : %v", err)
				}
			case *ast.SelectorExpr:
				// fields from other packages
				if err := pkg.ReadSelector(fieldType, &embeddedField, nil); err != nil {
					log.Printf("[error] in %q while reading embed selector : %v", info.Name, err)
					return fmt.Errorf("error reading selector : %v", err)
				}
			default:
				log.Printf("[error] in %q unknown embedded field %q of type %T", info.Name, embeddedField.Kind, fieldType)
				// not allowed
				return fmt.Errorf(ErrNotImplemented, fieldType, info.Name)
			}
			embeddedField.Name = embeddedField.Kind
			if pkg.PrintDebug {
				log.Printf("%q: embedded field %q", info.Name, embeddedField.Name)
			}
			info.Fields = append(info.Fields, &embeddedField)
			continue
		}
		newField := FieldInfo{Name: field.Names[0].Name, IsExported: field.Names[0].IsExported(), Comment: field.Comment}
		if pkg.PrintDebug {
			log.Printf("%q: inspecting %q %T", info.Name, newField.Name, field.Type)
		}
		switch fieldType := field.Type.(type) {
		case *ast.StarExpr:
			if err := pkg.ReadPointer(fieldType, &newField, nil); err != nil {
				log.Printf("[error] in %q while reading pointer : %v", info.Name, err)
				return err
			}
		case *ast.Ident:
			if err := pkg.ReadIdent(fieldType, &newField, nil); err != nil {
				log.Printf("[error] in %q while reading ident : %v", info.Name, err)
				return fmt.Errorf("error reading selector : %v", err)
			}
		case *ast.MapType:
			newField.IsMap = true
		case *ast.ChanType:
			newField.IsChan = true
		case *ast.ArrayType:
			newField.IsArray = true
		case *ast.SelectorExpr:
			// fields from other packages
			if err := pkg.ReadSelector(fieldType, &newField, nil); err != nil {
				log.Printf("[error] in %q while reading selector : %v", info.Name, err)
				return fmt.Errorf("error reading selector : %v", err)
			}
		case *ast.FuncType:
			log.Printf("[TODO] read func %q on field %q of type %T : %#v", info.Name, newField.Kind, fieldType, fieldType)
		case *ast.InterfaceType:
			log.Printf("[TODO] read interface %q on field %q of type %T : %#v", info.Name, newField.Kind, fieldType, fieldType)
			// ignore ??
		default:
			log.Printf("[error] in %q unknown field %q of type %T", info.Name, newField.Kind, fieldType)
			return fmt.Errorf(ErrNotImplemented, fieldType, info.Name)
		}

		if field.Tag != nil {
			var err error
			newField.Tags, err = ParseTags(field.Tag.Value)
			if err != nil {
				log.Printf("[error] in %q while parsing tags : %v", info.Name, err)
				return fmt.Errorf("error parsing tags : %v of field named %q input = %s", err.Error(), newField.Name, field.Tag.Value)
			}
		}
		if pkg.PrintDebug {
			log.Printf("%q: adding field %q", newField.Name, info.Name)
		}
		info.Fields = append(info.Fields, &newField)
	}
	if pkg.PrintDebug {
		log.Printf("%q: adding to Types", info.Name)
	}
	pkg.Types = append(pkg.Types, &info)
	return nil
}

// get function information from the function object
func (pkg *PackageInfo) ReadFunctionInfo(spec *ast.FuncDecl) error {
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
		return errors.New("ReadFunctionInfo : obj is nil")
	}
	pkg.Functions = append(pkg.Functions, &info)
	return nil
}

// get interface information from interface type declaration
func (pkg *PackageInfo) ReadInterfaceInfo(spec ast.Spec, comment *ast.CommentGroup) error {
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
	return nil
}

func (pkg *PackageInfo) ReadVariablesInfo(spec ast.Spec, valueSpec *ast.ValueSpec) error {
	obj := pkg.TypesInfo.Defs[spec.(*ast.ValueSpec).Names[0]]
	if obj == nil {
		log.Printf("%q was not found ", spec.(*ast.ValueSpec).Names[0])
		return nil
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
	return nil
}
