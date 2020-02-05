package stroo_test

import (
	"fmt"
	. "github.com/badu/stroo"
	"github.com/badu/stroo/halp"
	"go/ast"
	"go/types"
	"golang.org/x/tools/go/packages"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
)

type testCase struct {
	name       string
	outputName string
	output     interface{}
}

const (
	testPackage     = "testdata"
	testPackagePath = "github.com/badu/stroo/testdata"
)

var cases = []testCase{
	{
		name:       "slice of int",
		outputName: "T0",
		output:     &TypeInfo{Package: testPackage, PackagePath: testPackagePath, Name: "T0", Kind: "int", IsArray: true},
	}, // 0
	{
		name:       "p1",
		outputName: "T1",
		output:     &TypeInfo{Package: testPackage, PackagePath: testPackagePath, Name: "T1", Kind: "int", IsPointer: true, IsArray: true},
	}, // 1
	{
		name:       "p2",
		outputName: "T2",
		output:     &TypeInfo{Package: testPackage, PackagePath: testPackagePath, Name: "T2", Kind: "int", IsArray: true},
	}, // 2.
	{
		name:       "p2_1",
		outputName: "T2_1",
		output:     &TypeInfo{Package: testPackage, PackagePath: testPackagePath, Name: "T2_1", Kind: "int", IsArray: true}, //, IsPointer: true},
	}, // 2.1.
	{
		name:       "p3",
		outputName: "T3",
		output:     &TypeInfo{Package: testPackage, PackagePath: testPackagePath, Name: "T3", Kind: "map (temporary)", IsArray: true},
	}, // 3. `
	{
		name:       "p4",
		outputName: "T4",
		output:     &TypeInfo{Package: testPackage, PackagePath: testPackagePath, Name: "T4", Kind: "map (temporary)", IsArray: true},
	}, // 4. `
	{
		name:       "p5",
		outputName: "T5",
		output:     &TypeInfo{Package: testPackage, PackagePath: testPackagePath, Name: "T5", Kind: "map (temporary)", IsArray: true, IsPointer: true},
	}, // 5. `
	{
		name:       "p6",
		outputName: "T6",
		output:     &TypeInfo{Package: testPackage, PackagePath: testPackagePath, Name: "T6", Kind: "struct (temporary)", IsArray: true},
	}, // 6. `
	{
		name:       "p7",
		outputName: "T7",
		output:     &TypeInfo{Package: testPackage, PackagePath: testPackagePath, Name: "T7", Kind: "chan (temporary)", IsArray: true},
	}, // 7. `
	{
		name:       "p8",
		outputName: "T8",
		output:     &TypeInfo{Package: testPackage, PackagePath: testPackagePath, Name: "T8", Kind: "chan (temporary)", IsArray: true, IsPointer: true},
	}, // 8. `
	{
		name:       "p9",
		outputName: "T9",
		output:     &TypeInfo{Package: testPackage, PackagePath: testPackagePath, Name: "T9", Kind: "chan (temporary)", IsArray: true},
	}, // 9. `
	{
		name:       "p10",
		outputName: "T10",
		output:     &TypeInfo{Package: testPackage, PackagePath: testPackagePath, Name: "T10", Kind: "chan (temporary)", IsArray: true, IsPointer: true},
	}, // 10. `
	{
		name:       "p11",
		outputName: "T11",
		output:     &TypeInfo{Package: testPackage, PackagePath: testPackagePath, Name: "T11", Kind: "struct (temporary)", IsArray: true, IsPointer: true},
	}, // 11. `
	{
		name:       "p12",
		outputName: "T12",
		output: &TypeInfo{
			Name:        "T12",
			Kind:        "S3",
			Package:     testPackage,
			PackagePath: testPackagePath,
			IsArray:     true,
		},
	}, // 12
	{
		name:       "p13",
		outputName: "T13",
		output: &TypeInfo{
			Name:        "T13",
			Kind:        "S13",
			Package:     testPackage,
			PackagePath: testPackagePath,
			IsArray:     true,
		},
	}, // 13
	{
		name:       "p14",
		outputName: "T14",
		output: &TypeInfo{
			Name:        "T14",
			Kind:        "S4",
			IsPointer:   true,
			IsArray:     true,
			Package:     testPackage,
			PackagePath: testPackagePath,
		},
	}, // 14

	{
		name:       "p15",
		outputName: "T15",
		output: &TypeInfo{
			Package:     testPackage,
			PackagePath: testPackagePath,
			Name:        "T15",
			Kind:        "T15",
			Fields: Fields{
				FieldInfo{
					PackagePath: testPackagePath,
					Package:     testPackage,
					Kind:        "EmbeddedS",
					IsEmbedded:  true,
					IsStruct:    true,
					Comment: &ast.CommentGroup{
						List: []*ast.Comment{
							{
								Slash: 524931,
								Text:  "// embedded",
							},
						},
					},
				},
				FieldInfo{
					PackagePath: testPackagePath,
					Package:     testPackage,
					Kind:        "EmbeddedS2",
					IsPointer:   true,
					IsEmbedded:  true,
					IsStruct:    true,
					Comment: &ast.CommentGroup{
						List: []*ast.Comment{
							{
								Slash: 524963,
								Text:  "// embedded pointer",
							},
						},
					},
				},
				FieldInfo{
					PackagePath: testPackagePath,
					Package:     testPackage,
					Kind:        "error",
					IsEmbedded:  true,
					IsInterface: true,
					Comment: &ast.CommentGroup{
						List: []*ast.Comment{
							{
								Slash: 525003,
								Text:  "// embedded error",
							},
						},
					},
				},
				FieldInfo{
					PackagePath: testPackagePath,
					Package:     testPackage,
					Name:        "Name",
					Kind:        "string",
					IsBasic:     true,
					IsExported:  true,
					Tags: Tags{
						&Tag{Key: "json", Name: "name"},
					},
				},
				FieldInfo{
					PackagePath: testPackagePath,
					Package:     testPackage,
					Name:        "PtrName",
					Kind:        "string",
					IsBasic:     true,
					IsPointer:   true,
					IsExported:  true,
				},
				FieldInfo{
					PackagePath: testPackagePath,
					Package:     testPackage,
					Name:        "unexported",
					Kind:        "string",
					IsBasic:     true,
				},
			},
		},
	}, // 15 - some fields
	{
		name:       "p16",
		outputName: "T16",
		output: &TypeInfo{
			Name:        "T16",
			Kind:        "T16",
			PackagePath: testPackagePath,
			Package:     testPackage,
			Fields: Fields{
				FieldInfo{
					PackagePath: testPackagePath,
					Package:     testPackage,
					Name:        "S",
					Kind:        "S5",
					IsExported:  true,
					IsStruct:    true,
				},
				FieldInfo{
					PackagePath: testPackagePath,
					Package:     testPackage,
					Name:        "Itemz",
					Kind:        "Items",
					IsExported:  true,
					Tags: Tags{
						&Tag{Key: "json", Name: "itmz"},
					},
					IsArray: true,
				},
				FieldInfo{
					PackagePath: testPackagePath,
					Package:     testPackage,
					Name:        "Pricez",
					Kind:        "Prices",
					IsExported:  true,
					Tags: Tags{
						&Tag{Key: "json", Name: "prcz"},
					},
					IsArray: true,
				},
			},
		},
	}, // 16 - has array properties
	{
		name:       "p17",
		outputName: "T17",
		output: &TypeInfo{
			Package:     testPackage,
			PackagePath: testPackagePath,
			Name:        "T17",
			Kind:        "T17",
			Fields: Fields{
				FieldInfo{
					Package:     testPackage,
					PackagePath: testPackagePath,
					Kind:        "Items",
					IsArray:     true,
					IsEmbedded:  true,
				},
			},
		},
	}, // 17 - embed array
	{
		name:       "p18",
		outputName: "T18",
		output: &TypeInfo{
			Name:        "T18",
			Kind:        "T18",
			Package:     testPackage,
			PackagePath: testPackagePath,
			Fields: Fields{
				FieldInfo{
					Package:     testPackage,
					PackagePath: testPackagePath,
					Name:        "Child",
					Kind:        "T18",
					IsPointer:   true,
					IsExported:  true,
					IsStruct:    true,
					Tags: Tags{
						&Tag{Key: "json", Name: "ptr_child"},
					},
				},
			},
		},
	}, // 18 - circular reference
	{
		name:       "p19",
		outputName: "ExternalSliceOfPointers",
		output: &TypeInfo{
			Package:     testPackage,
			PackagePath: testPackagePath,
			Name:        "ExternalSliceOfPointers",
			Kind:        "time.Time",
			IsPointer:   true,
			IsArray:     true,
			IsImported:  true,
		},
	}, // 19 - type ExternalSliceOfPointers []*time.Time
	{
		name:       "p20",
		outputName: "ExternalPtrAlias",
		output: &TypeInfo{
			Package:     testPackage,
			PackagePath: testPackagePath,
			Name:        "ExternalPtrAlias",
			Kind:        "time.Ticker",
			IsPointer:   true,
			IsImported:  true,
			IsAlias:     true,
		},
	}, // 20 - type ExternalPtrAlias *time.Ticker
	{
		name:       "p21",
		outputName: "ExternalAlias",
		output: &TypeInfo{
			Package:     testPackage,
			PackagePath: testPackagePath,
			Name:        "ExternalAlias",
			Kind:        "time.Ticker",
			IsImported:  true,
			IsAlias:     true,
		},
	}, // 21 - type ExternalAlias time.Ticker
	{
		name:       "p22",
		outputName: "BasicPtrAlias",
		output: &TypeInfo{
			Package:     testPackage,
			PackagePath: testPackagePath,
			Name:        "BasicPtrAlias",
			Kind:        "string",
			IsPointer:   true,
			IsAlias:     true,
		},
	}, // 22 - type BasicPtrAlias *string
	{
		name:       "p23",
		outputName: "BasicAlias",
		output: &TypeInfo{
			Package:     testPackage,
			PackagePath: testPackagePath,
			Name:        "BasicAlias",
			Kind:        "string",
			IsAlias:     true,
		},
	}, // 23 - type BasicAlias string
}

func TestLoadExamplePackage(t *testing.T) {
	loadedPackage, err := LoadPackage(testPackagePath)
	if err != nil {
		t.Fatalf("error : %v", err)
	}
	codeBuilder := DefaultAnalyzer()
	command := NewCommand(codeBuilder)
	if err := command.Analyse(codeBuilder, loadedPackage); err != nil {
		t.Fatalf("error : %v", err)
	}
	for idx := 0; idx < len(cases); idx++ {
		currentType := cases[idx].outputName
		resultType := command.Result.Types.Extract(currentType)
		if resultType == nil {
			var knownTypes []string
			for _, sType := range command.Result.Types {
				knownTypes = append(knownTypes, sType.Kind)
			}
			t.Fatalf("error : %q not found in types\nknown types:\n%s", currentType, strings.Join(knownTypes, "\n"))
		}
		if compared := halp.Equal(resultType, cases[idx].output); compared != nil {
			t.Logf("%d. %#v", idx, compared)
			t.Fatalf("expected :\n%s\nactual :\n%s\n", halp.SPrint(cases[idx].output), halp.SPrint(resultType))
		}
	}

	t.Logf("ran %d tests and finished.", len(cases))
}

func TestLoadWithExternal(t *testing.T) {
	wd, _ := os.Getwd()
	file1, err := ioutil.ReadFile(wd + "/testdata/pkg/model_a/easy.go")
	if err != nil {
		t.Fatalf("error loading source file : %v", err)
	}
	file2, err := ioutil.ReadFile(wd + "/testdata/pkg/model_b/model.go")
	if err != nil {
		t.Fatalf("error loading second source file : %v", err)
	}

	tmpProj, err := CreateTempProj([]TemporaryPackage{
		{
			Name:  "model_a",
			Files: map[string]interface{}{"model.go": string(file1)},
		},
		{
			Name:  "model_b",
			Files: map[string]interface{}{"model.go": string(file2)},
		},
	})
	if err != nil {
		t.Fatalf("Error : %v", err)
	}
	defer tmpProj.Cleanup()

	tmpProj.Config.Mode = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedImports

	// load package
	thePackages, err := packages.Load(tmpProj.Config, fmt.Sprintf("file=%s", tmpProj.File("model_a", "model.go")))
	if err != nil {
		t.Fatalf("error loading : %v", err)
	}
	// helper
	contains := func(imports map[string]*packages.Package, wantImport string) bool {
		for imp := range imports {
			if imp == wantImport {
				return true
			}
		}
		return false
	}

	for _, pkg := range thePackages {
		if !contains(pkg.Imports, "time") {
			t.Errorf("expected %s import in %s", "time", pkg.ID)
		}
		if !contains(pkg.Imports, "github.com/badu/stroo/testdata/pkg/model_b") {
			t.Errorf("expected %s import in %s", "github.com/badu/stroo/testdata/pkg/model_b", pkg.ID)
		}
	}
}

func TestLoadWithCommand(t *testing.T) {
	wd, _ := os.Getwd()
	file1, err := ioutil.ReadFile(wd + "/testdata/pkg/model_a/easy.go")
	if err != nil {
		t.Fatalf("error loading source file : %v", err)
	}
	file2, err := ioutil.ReadFile(wd + "/testdata/pkg/model_b/model.go")
	if err != nil {
		t.Fatalf("error loading second source file : %v", err)
	}

	tmpProj, err := CreateTempProj([]TemporaryPackage{
		{
			Name:  "model_a",
			Files: map[string]interface{}{"model.go": string(file1)},
		},
		{
			Name:  "model_b",
			Files: map[string]interface{}{"model.go": string(file2)},
		},
	})
	if err != nil {
		t.Fatalf("Error : %v", err)
	}
	defer tmpProj.Cleanup()

	tmpProj.Config.Mode = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedImports

	// load package
	thePackages, err := packages.Load(tmpProj.Config, fmt.Sprintf("file=%s", tmpProj.File("model_a", "model.go")))
	if err != nil {
		t.Fatalf("error loading : %v", err)
	}
	codeBuilder := DefaultAnalyzer()
	command := NewCommand(codeBuilder)
	if err := command.Analyse(codeBuilder, thePackages[0]); err != nil {
		t.Fatalf("error analyzing package : %v", err)
	}
	t.Logf("Result : %s", halp.SPrint(command.Result))
}

func TestLoadStdPackage(t *testing.T) {
	const (
		currentType = "RawMessage"
	)
	loadedPackage, err := LoadPackage("encoding/json")
	if err != nil {
		t.Fatalf("error : %v", err)
	}
	codeBuilder := DefaultAnalyzer()
	command := NewCommand(codeBuilder)
	if err := command.Analyse(codeBuilder, loadedPackage); err != nil {
		t.Fatalf("error : %v", err)
	}

	resultType := command.Result.Types.Extract(currentType)
	if resultType == nil {
		var knownTypes []string
		for _, sType := range command.Result.Types {
			knownTypes = append(knownTypes, sType.Kind)
		}
		t.Fatalf("error : %q not found in types\nknown types:\n%s", currentType, strings.Join(knownTypes, "\n"))
	}

	t.Logf("listing %q :\n %s", currentType, halp.SPrint(resultType))
	for _, field := range resultType.Fields {
		t.Logf("field :\n %#v\n", field)
	}
	for _, fn := range resultType.MethodList {
		t.Logf("method :\n %#v\n", fn)
	}
}

func TestTypesInfoDefs(t *testing.T) {
	const packageName = "encoding/json"
	loadedPackage, err := LoadPackage(packageName)
	if err != nil {
		t.Fatalf("error : %v", err)
	}
	codeBuilder := DefaultAnalyzer()
	command := NewCommand(codeBuilder)
	if err := command.Analyse(codeBuilder, loadedPackage); err != nil {
		t.Fatalf("error : %v", err)
	}
	packageParts := strings.Split(packageName, "/")
	currentPackage := packageParts[len(packageParts)-1]
	t.Logf("Dumping knowhow:")
	alreadyDumped := make(map[string]struct{})
	for key, value := range command.Result.TypesInfo.Types {
		if !value.IsType() {
			//t.Logf("value not type : %s", dbg_prn.SPrint(value))
			continue
		}

		keyStr := ""
		switch kType := key.(type) {
		case *ast.Ident:
			keyStr = "[ident] `" + kType.Name + "`"
		default:
			continue
			//case *ast.StructType:
			//	keyStr = fmt.Sprintf("[struct] %d fields : %s", kType.Fields.NumFields(), dbg_prn.SPrint(key)) //+ dbg_prn.SPrint(kType.Fields)
			//case *ast.SelectorExpr:
			//	keyStr = "[selector] " + dbg_prn.SPrint(kType.Sel)
			//default:
			//	keyStr = fmt.Sprintf("[%T] %s", kType, dbg_prn.SPrint(kType))
		}

		switch structDef := value.Type.Underlying().(type) {
		case *types.Struct:
			pushContinue := false
			for i := 0; i < structDef.NumFields(); i++ {
				field := structDef.Field(i)
				if field.Pkg().Name() != currentPackage {
					//t.Logf("field is not in package : %s",  field.Pkg().Name())
					// this is not in current package - might be useful, but taking one package at a time
					pushContinue = true
					break
				}
			}
			if pushContinue {
				continue
			}
			if _, has := alreadyDumped[keyStr]; has {
				continue
			}
			alreadyDumped[keyStr] = struct{}{}
			t.Logf("Struct : %s", keyStr)
			for i := 0; i < structDef.NumFields(); i++ {
				field := structDef.Field(i)
				if !field.IsField() {
					t.Logf("field is not field : %s", halp.SPrint(field))
					continue
				}
				if field.Pkg().Name() != currentPackage {
					//t.Logf("field is not in package : %s",  field.Pkg().Name())
					// this is not in current package - might be useful, but taking one package at a time
					continue
				}
				embed := ""
				if field.Embedded() {
					embed = "embedded"
				}
				anon := ""
				if field.Anonymous() {
					anon = "anonymous"
				}
				private := ""
				if field.Exported() {
					private = "public"
				}
				fieldKind := ""
				switch uType := field.Type().Underlying().(type) {
				case *types.Slice:
					// slice of something
					fieldKind = "[]" + uType.Elem().String()
				case *types.Array:
					// slice of something
					fieldKind = "[" + uType.Elem().String() + "]" + uType.Elem().String()
				case *types.Basic:
					// basic field (int, string, etc)
					fieldKind = uType.Name()
				case *types.Pointer:
					fieldKind = "*" + uType.String()
				case *types.Signature:
					// function field
					fieldKind = "func()"
				case *types.Struct:
					fieldKind = "struct{} " + strconv.Itoa(uType.NumFields()) + " fields"
				case *types.Named:
					fieldKind = "->" + halp.SPrint(uType.Underlying())
				case *types.Interface:
					switch iType := field.Type().(type) {
					case *types.Named:
						fieldKind = "interface `" + iType.String() + "` [" + strconv.Itoa(iType.NumMethods()) + " methods]"
					default:
						fieldKind = "interface ??? " + iType.String() + " [" + halp.SPrint(iType) + "]"
					}
				case *types.Map:
					fieldKind = "map [" + uType.Key().String() + "]" + uType.Elem().String()
				default:
					fieldKind = "{{" + halp.SPrint(uType) + "}}"
				}
				t.Logf("\t%s %s %s field in package %q named %q of kind %s", embed, anon, private, field.Pkg().Name(), field.Name(), fieldKind)
			}
		}
	}
}
