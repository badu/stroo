package stroo_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	. "github.com/badu/stroo/stroo"
)

type testCase struct {
	name       string
	input      string
	outputName string
	output     interface{}
}

func TestArrayDefinition(t *testing.T) {
	cases := []testCase{
		{
			name:       "p",
			input:      `package p; type T0 []int;`,
			outputName: "T0",
			output:     &TypeInfo{Name: "T0", Kind: "int", IsArray: true},
		}, // 0
		{
			name:       "p",
			input:      `package p; type T1 []*int;`,
			outputName: "T1",
			output:     &TypeInfo{Name: "T1", Kind: "int", IsPointer: true, IsArray: true},
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
				Name:    "T12",
				Kind:    "S",
				IsArray: true,
				Reference: &TypeInfo{
					Name: "S",
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
			input:      `package p; type S struct{ Name string }; type T13 []*S;`,
			outputName: "T13",
			output: &TypeInfo{
				Name:      "T13",
				Kind:      "S",
				IsPointer: true,
				IsArray:   true,
				Reference: &TypeInfo{
					Name: "S",
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
		}, // 13
	}

	for idx, cCase := range cases {
		result := PackageInfo{Name: "test", StructDefs: make(map[string]*TypeInfo)}
		VisitedStructs = make(map[string]struct{}) // reset
		VisitedFields = make(map[string]string)    // reset
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
			continue
		}

		if cCase.output != nil {
			if cCase.outputName == "" {
				t.Skipf("%d. output is no-name", idx)
			}
			def := result.StructDefs[cCase.outputName]

			var expected, received strings.Builder
			if typed, ok := cCase.output.(*TypeInfo); ok {
				typed.Debug(&expected, 0)
			} else {
				e, ok := cCase.output.(string)
				if !ok {
					t.Fatalf("cCase.output should be string or *TypeInfo! it's : %T", cCase.output)
				}
				expected.WriteString(e)
			}
			def.Debug(&received, 0)
			// because circular references, we cannot use reflect.DeepEqual
			if received.String() != expected.String() {
				t.Logf("%d. output error: GOT:\n%s\nWANT:\n%s", idx, received.String(), expected.String())
			} else {
				//t.Logf("test #%d:\n%s\n%s", idx, cCase.input, received.String())
			}
		}
	}
}

func TestStructDefinition(t *testing.T) {
	cases := []testCase{
		{
			name: "p",
			input: `package p 
					type T1 struct{
						S // embedded
						*S2 // embedded pointer
						error // embedded error 
						Name string ` + "`json:\"name\"`" + `
						PtrName *string 
						unexported string 
                    }
					type S struct { 
						Name string ` + "`json:\"s.name\"`" + `
					}
					type S2 struct { 
						Name string ` + "`json:\"s2.name\"`" + `
					}`,
			outputName: "T1",
			output: &TypeInfo{
				Name: "T1",
				Fields: Fields{
					&FieldInfo{
						Kind:       "S",
						IsEmbedded: true,
						IsStruct:   true,
						Reference: &TypeInfo{
							Name: "S",
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
							},
						},
					},
					&FieldInfo{
						Kind:       "S2",
						IsPointer:  true,
						IsEmbedded: true,
						IsStruct:   true,
						Reference: &TypeInfo{
							Name: "S2",
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
						Kind:       "error",
						IsBasic:    true,
						IsEmbedded: true,
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
		}, // 0 - some fields
		{
			name: "p",
			input: `package p 
					type T2 struct{
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
			outputName: "T2",
			output: &TypeInfo{
				Name: "T2",
				Fields: Fields{
					&FieldInfo{
						Name:       "S",
						Kind:       "S",
						IsExported: true,
						IsStruct:   true,
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
		}, // 1 - has array properties
		{
			name: "p",
			input: `package p 
					type T3 struct{
						Items
                    }
					type Items []*Item
					type Item struct { 
						Name string
					}`,
			outputName: "T3",
			output: &TypeInfo{
				Name: "T3",
				Fields: Fields{
					&FieldInfo{
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
		}, // 2 - embed array
		{
			name: "p",
			input: `package p 
					type TR struct{
						Child TR ` + "`json:\"child\"`" + `
						PtrChild *TR` + "`json:\"ptr_child\"`" + `
                    }`,
			outputName: "TR",
			output: &TypeInfo{
				Name: "TR",
				Fields: Fields{
					&FieldInfo{
						Name:       "Child",
						Kind:       "TR",
						IsExported: true,
						Tags: &Tags{
							&Tag{Key: "json", Name: "child"},
						},
						IsStruct: true,
					},
					&FieldInfo{
						Name:       "PtrChild",
						Kind:       "TR",
						IsPointer:  true,
						IsExported: true,
						Tags: &Tags{
							&Tag{Key: "json", Name: "ptr_child"},
						},
						IsStruct: true,
					},
				},
			},
		}, // 3 - circular reference
		{
			name: "p",
			input: `package p
					type T4 struct{
						S S ` + "`json:\"s\"`" + `
					}
					type S struct{
						T T4 ` + "`json:\"t\"`" + `
					}`,
			outputName: "T4",
			output: &TypeInfo{
				Name: "T4",
				Fields: Fields{
					&FieldInfo{
						Name:       "S",
						Kind:       "S",
						IsStruct:   true,
						IsExported: true,
						Tags: &Tags{
							&Tag{Key: "json", Name: "s"}},
						Reference: &TypeInfo{
							Name: "S",

							Fields: Fields{
								&FieldInfo{
									Name: "T",
									Kind: "T4",
								},
							},
						},
					},
				},
			},
		}, // 4 - indirect circular reference
		{
			name: "p",
			input: `	package p
						type T5 struct{
							Children Children ` + "`json:\"children\"`" + `
							StarChildren StarChildren ` + "`json:\"star_children\"`" + `
						}
						type Children []T5
						type StarChildren []*T5`,
			outputName: "T5",
			output: &TypeInfo{
				Name: "T5",
				Fields: Fields{
					&FieldInfo{
						Name:       "Children",
						Kind:       "Children",
						IsExported: true,
						Tags: &Tags{
							&Tag{Key: "json", Name: "children"},
						},
						IsArray: true,
						Reference: &TypeInfo{
							Name: "T5",
							Fields: Fields{
								&FieldInfo{
									Name:       "Children",
									Kind:       "Children",
									IsExported: true,
									Tags: &Tags{
										&Tag{Key: "json", Name: "children"},
									},
									IsArray: true,
								},
								&FieldInfo{
									Name:       "StarChildren",
									Kind:       "StarChildren",
									IsExported: true,
									Tags: &Tags{
										&Tag{Key: "json", Name: "star_children"},
									},
									IsArray: true,
								},
							},
						},
					},
					&FieldInfo{
						Name:       "StarChildren",
						Kind:       "StarChildren",
						IsExported: true,
						Tags: &Tags{
							&Tag{Key: "json", Name: "star_children"},
						},
						IsArray: true,
						Reference: &TypeInfo{
							Name: "T5",
							Fields: Fields{
								&FieldInfo{
									Name:       "Children",
									Kind:       "Children",
									IsExported: true,
									Tags: &Tags{
										&Tag{Key: "json", Name: "children"},
									},
									IsArray: true,
								},
								&FieldInfo{
									Name:       "StarChildren",
									Kind:       "StarChildren",
									IsExported: true,
									Tags: &Tags{
										&Tag{Key: "json", Name: "star_children"},
									},
									IsArray: true,
								},
							},
						},
					},
				},
			},
		}, // 5 - indirect circular reference via array
	}

	for idx, cCase := range cases {
		result := PackageInfo{Name: "test", StructDefs: make(map[string]*TypeInfo)}
		VisitedStructs = make(map[string]struct{}) // reset
		VisitedFields = make(map[string]string)    // reset
		fileSet := token.NewFileSet()
		astNodes, err := parser.ParseFile(fileSet, cCase.name, cCase.input, parser.DeclarationErrors|parser.AllErrors)
		if err != nil {
			t.Fatalf("%d. Fatal (parse) error : %v", idx, err)
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

		var infoErr error
		for _, node := range astNodes.Decls {
			switch nodeType := node.(type) {
			case *ast.GenDecl:
				for _, spec := range nodeType.Specs {
					astSpec := spec.(*ast.TypeSpec)
					switch astSpec.Type.(type) {
					case *ast.ArrayType:
						infoErr = result.ReadArrayInfo(spec.(*ast.TypeSpec), nodeType.Doc)
						if infoErr != nil {
							t.Fatalf("%d.error reading array : %v", idx, infoErr)
						}
					case *ast.StructType:
						infoErr = result.ReadStructInfo(spec.(*ast.TypeSpec), nodeType.Doc)
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
			continue
		}

		if cCase.output != nil {
			if cCase.outputName == "" {
				t.Skipf("%d. output is no-name", idx)
			}
			if err := result.PostProcess(); err != nil {
				t.Fatalf("%d. Post process fatal error : %v", idx, err)
			}
			def := result.Types.Extract(cCase.outputName)
			if def == nil {
				t.Fatalf("You forgot to set outputname so I can select definition")
			}
			typed, ok := cCase.output.(*TypeInfo)
			if !ok {
				t.Fatalf("error : expecting output to be *TypeInfo and it's not")
			}
			var expected, received strings.Builder
			typed.Debug(&expected, 0)
			def.Debug(&received, 0)
			// because circular references, we cannot use reflect.DeepEqual
			if received.String() != expected.String() {
				t.Logf("%d. output error: GOT:\n%s\nWANT:\n%s", idx, received.String(), expected.String())
			} else {
				//t.Logf("test #%d:\n%s\n%s", idx, cCase.input, received.String())
			}
		}
	}
}
