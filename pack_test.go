package stroo_test

import (
	"fmt"
	"github.com/badu/stroo/dbg_prn"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"io/ioutil"
	"os"
	"testing"

	. "github.com/badu/stroo"
)

type testCase struct {
	name       string
	input      string
	outputName string
	output     interface{}
}

var cases = []testCase{
	{
		name:       "p",
		input:      `package p; type T0 []int;`,
		outputName: "T0",
		output:     &TypeInfo{Name: "T0", Kind: "int", IsArray: true, Package: "p", PackagePath: "p"},
	}, // 0
	{
		name:       "p",
		input:      `package p; type T1 []*int;`,
		outputName: "T1",
		output:     &TypeInfo{Name: "T1", Kind: "int", IsPointer: true, IsArray: true, Package: "p", PackagePath: "p"},
	}, // 1
	{
		name:       "",
		input:      `package p; type T2 [][]int;`,
		outputName: "T2",
		output:     `*ast.ArrayType found on "T2" (not implemented)`,
	}, // 2
	{
		name:       "",
		input:      `package p; type T3 []map[string]string;`,
		outputName: "T3",
		output:     `*ast.MapType found on "T3" (not implemented)`,
	}, // 3
	{
		name:       "",
		input:      `package p; type S struct { Name string }; type T4 []map[S]string;`,
		outputName: "T4",
		output:     `*ast.MapType found on "T4" (not implemented)`,
	}, // 4
	{
		name:       "",
		input:      `package p; type S struct { Name string }; type T5 []*map[S]string;`,
		outputName: "T5",
		output:     `*ast.StarExpr found on "T5" (not implemented)`,
	}, // 5
	{
		name:       "",
		input:      `package p; type T6 []struct{ Name string };`,
		outputName: "T6",
		output:     `*ast.StructType found on "T6" (not implemented)`,
	}, // 6
	{
		name:       "",
		input:      `package p; type T7 []chan string;`,
		outputName: "T7",
		output:     `*ast.ChanType found on "T7" (not implemented)`,
	}, // 7
	{
		name:       "",
		input:      `package p; type T8 []*chan string;`,
		outputName: "T8",
		output:     `*ast.StarExpr found on "T8" (not implemented)`,
	}, // 8
	{
		name:       "",
		outputName: "T9",
		input:      `package p; type T9 []chan struct{};`,
	}, // 9
	{
		name:       "",
		input:      `package p; type T10 []*chan *struct{};`,
		outputName: "T10",
	}, // 10
	{
		name:       "",
		input:      `package p; type T11 []*struct{ Name string };`,
		outputName: "T11",
		output:     `*ast.StarExpr found on "T11" (not implemented)`,
	}, // 11
	{
		name:       "",
		input:      `package p; type S struct{ Name string }; type T12 []S;`,
		outputName: "T12",
		output: &TypeInfo{
			Name:        "T12",
			Kind:        "S",
			Package:     "p",
			PackagePath: "p",
			IsArray:     true,
			Reference: &TypeInfo{
				Name:        "S",
				Package:     "p",
				PackagePath: "p",
				Fields: Fields{
					&FieldInfo{
						Name:       "Name",
						Kind:       "string",
						IsBasic:    true,
						IsExported: true,
					},
				},
			},
		},
	}, // 12
	{
		name:       "",
		input:      `package p; type T13 []S13; type S13 struct{ Name string; Email string }; `,
		outputName: "T13",
		output: &TypeInfo{
			Name:        "T13",
			Kind:        "S13",
			Package:     "p",
			PackagePath: "p",
			IsArray:     true,
			Reference: &TypeInfo{
				Name:        "S13",
				Package:     "p",
				PackagePath: "p",
				Fields: Fields{
					&FieldInfo{
						Name:       "Name",
						Kind:       "string",
						IsBasic:    true,
						IsExported: true,
					},
					&FieldInfo{
						Name:       "Email",
						Kind:       "string",
						IsBasic:    true,
						IsExported: true,
					},
				},
			},
		},
	}, // 13
	{
		name:       "",
		input:      `package p; type S struct{ Name string }; type T14 []*S;`,
		outputName: "T14",
		output: &TypeInfo{
			Name:        "T14",
			Kind:        "S",
			IsPointer:   true,
			IsArray:     true,
			Package:     "p",
			PackagePath: "p",
			Reference: &TypeInfo{
				Name:        "S",
				Package:     "p",
				PackagePath: "p",
				Fields: Fields{
					&FieldInfo{
						Name:       "Name",
						Kind:       "string",
						IsBasic:    true,
						IsExported: true,
					},
				},
			},
		},
	}, // 14

	{
		name: "p",
		input: `package p 
					type T15 struct{
						EmbeddedS // embedded
						*EmbeddedS2 // embedded pointer
						error // embedded error 
						Name string ` + "`json:\"name\"`" + `
						PtrName *string 
						unexported string 
                    }
					type EmbeddedS struct { 
						Name string ` + "`json:\"s.name\"`" + `
						Email string ` + "`json:\"mail\"`" + `
					}
					type EmbeddedS2 struct { 
						Name string ` + "`json:\"s2.name\"`" + `
					}`,
		outputName: "T15",
		output: &TypeInfo{
			Name:        "T15",
			Kind:        "T15",
			PackagePath: "p",
			Package:     "p",
			Fields: Fields{
				&FieldInfo{
					Name:       "EmbeddedS",
					Kind:       "EmbeddedS",
					IsEmbedded: true,
					IsStruct:   true,
					Reference: &TypeInfo{
						Name:        "EmbeddedS",
						PackagePath: "p",
						Package:     "p",
						Fields: Fields{
							&FieldInfo{
								Name:       "Name",
								Kind:       "string",
								IsBasic:    true,
								IsExported: true,
								Tags: &Tags{
									&Tag{Key: "json", Name: "s.name"},
								},
							},
							&FieldInfo{
								Name:       "Email",
								Kind:       "string",
								IsBasic:    true,
								IsExported: true,
								Tags: &Tags{
									&Tag{Key: "json", Name: "mail"},
								},
							},
						},
					},
				},
				&FieldInfo{
					Name:       "EmbeddedS2",
					Kind:       "EmbeddedS2",
					IsPointer:  true,
					IsEmbedded: true,
					IsStruct:   true,
					Reference: &TypeInfo{
						Name:        "EmbeddedS2",
						PackagePath: "p",
						Package:     "p",
						Fields: Fields{
							&FieldInfo{
								Name:       "Name",
								Kind:       "string",
								IsBasic:    true,
								IsExported: true,
								Tags: &Tags{
									&Tag{Key: "json", Name: "s2.name"},
								},
							},
						},
					},
				},
				&FieldInfo{
					Name:        "error",
					Kind:        "error",
					IsEmbedded:  true,
					IsInterface: true,
				},
				&FieldInfo{
					Name:       "Name",
					Kind:       "string",
					IsBasic:    true,
					IsExported: true,
					Tags: &Tags{
						&Tag{Key: "json", Name: "name"},
					},
				},
				&FieldInfo{
					Name:       "PtrName",
					Kind:       "string",
					IsBasic:    true,
					IsPointer:  true,
					IsExported: true,
				},
				&FieldInfo{
					Name:    "unexported",
					Kind:    "string",
					IsBasic: true,
				},
			},
		},
	}, // 15 - some fields
	{
		name: "p",
		input: `package p 
					type T16 struct{
						S S
						Itemz	Items ` + "`json:\"itmz\"`" + `
						Pricez	Prices ` + "`json:\"prcz\"`" + `
                    }
					type S struct{}
					type Prices []Price 
					type Items []*Item
					type Item struct { 
						Name string ` + "`json:\"name\"`" + `
						Stock float64
					}
					type Price struct { 
						Name string ` + "`json:\"name\"`" + `
						Value float64
					}`,
		outputName: "T16",
		output: &TypeInfo{
			Name:        "T16",
			Kind:        "T16",
			PackagePath: "p",
			Package:     "p",
			Fields: Fields{
				&FieldInfo{
					Name:       "S",
					Kind:       "S",
					IsExported: true,
					IsStruct:   true,
					Reference: &TypeInfo{
						Name:        "S",
						Package:     "p",
						PackagePath: "p",
					},
				},
				&FieldInfo{
					Name:       "Itemz",
					Kind:       "Items",
					IsExported: true,
					Tags: &Tags{
						&Tag{Key: "json", Name: "itmz"},
					},
					IsArray: true,
					Reference: &TypeInfo{
						Name: "Item",
						Fields: Fields{
							&FieldInfo{
								Name:       "Name",
								Kind:       "string",
								IsBasic:    true,
								IsExported: true,
								Tags: &Tags{
									&Tag{Key: "json", Name: "name"},
								},
							},
						},
					},
				},
				&FieldInfo{
					Name:       "Pricez",
					Kind:       "Prices",
					IsExported: true,
					Tags: &Tags{
						&Tag{Key: "json", Name: "prcz"},
					},
					IsArray: true,
					Reference: &TypeInfo{
						Name: "Price",
						Fields: Fields{
							&FieldInfo{
								Name:       "Name",
								Kind:       "string",
								IsBasic:    true,
								IsExported: true,
								Tags: &Tags{
									&Tag{Key: "json", Name: "name"},
								},
							},
						},
					},
				},
			},
		},
	}, // 16 - has array properties
	{
		name: "p",
		input: `package p 
					type T17 struct{
						Items
                    }
					type Items []*Item
					type Item struct { 
						Name string
					}`,
		outputName: "T17",
		output: &TypeInfo{
			Name:        "T17",
			Kind:        "T17",
			PackagePath: "p",
			Package:     "p",
			Fields: Fields{
				&FieldInfo{
					Name:       "Items",
					Kind:       "Items",
					IsArray:    true,
					IsEmbedded: true,
					Reference: &TypeInfo{
						Name: "Item",
						Fields: Fields{
							&FieldInfo{
								Name:       "Name",
								Kind:       "string",
								IsBasic:    true,
								IsExported: true,
							},
						},
					},
				},
			},
		},
	}, // 17 - embed array
	{
		name: "p",
		input: `package p 
					type T18 struct{
						Child *T18` + "`json:\"ptr_child\"`" + `
                    }`,
		outputName: "T18",
		output: &TypeInfo{
			Name:        "T18",
			Kind:        "T18",
			Package:     "p",
			PackagePath: "p",
			Fields: Fields{
				&FieldInfo{
					Name:       "Child",
					Kind:       "T18",
					IsPointer:  true,
					IsExported: true,
					IsStruct:   true,
					Tags: &Tags{
						&Tag{Key: "json", Name: "ptr_child"},
					},
				},
			},
		},
	}, // 18 - circular reference
}

func TestAllDefinitions(t *testing.T) {
	for idx, cCase := range cases {
		t.Logf("%d. Running test\n", idx)
		result := PackageInfo{Name: "test"} //, PrintDebug: true}
		fileSet := token.NewFileSet()
		astNodes, err := parser.ParseFile(fileSet, cCase.name, cCase.input, parser.DeclarationErrors|parser.AllErrors)
		if err != nil {
			t.Fatalf("%d. Fatal error : %v", idx, err)
		}

		info := types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
			Defs:  make(map[*ast.Ident]types.Object),
			Uses:  make(map[*ast.Ident]types.Object),
		}
		var conf types.Config
		_, err = conf.Check("p", fileSet, []*ast.File{astNodes}, &info)
		if err != nil {
			t.Fatal(err)
		}
		result.TypesInfo = &info
		/**
		tx := 0
		for key, value := range result.TypesInfo.Defs {
			t.Logf("%d.%d %#v === %#v", idx, tx, key, value)
			tx++
		}
		**/
		var infoErr error
		for _, node := range astNodes.Decls {
			switch nodeType := node.(type) {
			case *ast.GenDecl:
				for _, spec := range nodeType.Specs {
					astSpec := spec.(*ast.TypeSpec)
					switch astSpec.Type.(type) {
					case *ast.ArrayType:
						infoErr = result.ReadArrayInfo(astSpec, nodeType.Doc)
						if infoErr != nil && cCase.output != nil {
							if infoErr.Error() != cCase.output {
								t.Errorf("%d.errors not equal:\nexpected output:\n`%v`\nreceived:\n`%v`", idx, cCase.output, infoErr)
							}
						}
					case *ast.StructType:
						infoErr = result.ReadStructInfo(astSpec, nodeType.Doc)
						if infoErr != nil && cCase.output != nil {
							if infoErr.Error() != cCase.output {
								t.Errorf("%d.errors not equal:\nexpected output:\n`%v`\nreceived:\n`%v`", idx, cCase.output, infoErr)
							}
						}
					}
				}
			}
		}
		if infoErr != nil {
			t.Logf("%d. skipped : info err not nil : %v", idx, infoErr)
			continue
		}

		if cCase.output != nil {
			if cCase.outputName == "" {
				t.Skipf("%d. output is no-name", idx)
			}
			expected := ""
			var typed *TypeInfo
			ok := false
			if typed, ok = cCase.output.(*TypeInfo); ok {
				expected = dbg_prn.SPrint(typed)
			} else {
				t.Logf("%d. Expecting error", idx)
				e, ok := cCase.output.(string)
				if !ok {
					t.Fatalf("cCase.output should be string or *TypeInfo! it's : %T", cCase.output)
				}
				expected = dbg_prn.SPrint(e)
			}
			received := ""
			def := result.Types.Extract(cCase.outputName)
			if def == nil {
				t.Fatalf("Definition not found looking for %q", cCase.outputName)
			}
			received = dbg_prn.SPrint(def)
			// because circular references, we cannot use reflect.DeepEqual
			if received != expected {
				//			changelog, _ := diff.Diff(typed, def)
				//			t.Logf("%#v", changelog)
				t.Fatalf("%d. output error: GOT:\n%s\nWANT:\n%s", idx, received, expected)
			} else {
				t.Logf("Result of %d:\n%s\nSHOULD BE:\n\n%s\n", idx, received, expected)
			}
		} else {
			t.Logf("%d. skipped - expecting nothing", idx)
		}
	}
}

func TestOneDefinition(t *testing.T) {
	idx := 18
	cCase := cases[idx]
	t.Logf("%d. Running test\n", idx)
	result := PackageInfo{Name: "test"} //, PrintDebug: true}
	fileSet := token.NewFileSet()
	astNodes, err := parser.ParseFile(fileSet, cCase.name, cCase.input, parser.DeclarationErrors|parser.AllErrors)
	if err != nil {
		t.Fatalf("%d. Fatal error : %v", idx, err)
	}

	info := types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	var conf types.Config
	_, err = conf.Check("p", fileSet, []*ast.File{astNodes}, &info)
	if err != nil {
		t.Fatal(err)
	}
	result.TypesInfo = &info
	/**
	tx := 0
	for key, value := range result.TypesInfo.Defs {
		t.Logf("%d.%d %#v === %#v", idx, tx, key, value)
		tx++
	}
	**/
	var infoErr error
	for _, node := range astNodes.Decls {
		switch nodeType := node.(type) {
		case *ast.GenDecl:
			for _, spec := range nodeType.Specs {
				astSpec := spec.(*ast.TypeSpec)
				switch astSpec.Type.(type) {
				case *ast.ArrayType:
					infoErr = result.ReadArrayInfo(astSpec, nodeType.Doc)
					if infoErr != nil && cCase.output != nil {
						if infoErr.Error() != cCase.output {
							t.Errorf("%d.errors not equal:\nexpected output:\n`%v`\nreceived:\n`%v`", idx, cCase.output, infoErr)
						}
					}
				case *ast.StructType:
					infoErr = result.ReadStructInfo(astSpec, nodeType.Doc)
					if infoErr != nil && cCase.output != nil {
						if infoErr.Error() != cCase.output {
							t.Errorf("%d.errors not equal:\nexpected output:\n`%v`\nreceived:\n`%v`", idx, cCase.output, infoErr)
						}
					}
				}
			}
		}
	}
	if infoErr != nil {
		t.Skipf("%d. skipped : info err not nil : %v", idx, infoErr)
	}

	if cCase.output != nil {
		if cCase.outputName == "" {
			t.Skipf("%d. output is no-name", idx)
		}
		expected := ""
		var typed *TypeInfo
		ok := false
		if typed, ok = cCase.output.(*TypeInfo); ok {
			expected = dbg_prn.SPrint(typed)
		} else {
			t.Logf("%d. Expecting error", idx)
			e, ok := cCase.output.(string)
			if !ok {
				t.Fatalf("cCase.output should be string or *TypeInfo! it's : %T", cCase.output)
			}
			expected = dbg_prn.SPrint(e)
		}
		received := ""
		def := result.Types.Extract(cCase.outputName)
		if def == nil {
			t.Fatalf("Definition not found looking for %q", cCase.outputName)
		}
		received = dbg_prn.SPrint(def)
		// because circular references, we cannot use reflect.DeepEqual
		if received != expected {
			//			changelog, _ := diff.Diff(typed, def)
			//			t.Logf("%#v", changelog)
			t.Fatalf("%d. output error: GOT:\n%s\nWANT:\n%s", idx, received, expected)
		} else {
			t.Logf("Result of %d:\n%s\nSHOULD BE:\n\n%s\n", idx, received, expected)
		}
	} else {
		t.Logf("%d. skipped - expecting nothing", idx)
	}

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
	command := NewCommand(DefaultAnalyzer())
	if err := command.Analyse(thePackages[0]); err != nil {
		t.Fatalf("error analyzing package : %v", err)
	}
	t.Logf("Result : %#v", command.Result)
}
