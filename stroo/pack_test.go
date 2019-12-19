package stroo_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
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
			input:      `package p; type T []int;`,
			outputName: "T",
			output:     FieldInfo{Name: "T", TypeName: "int", IsBasic: true, IsArray: true},
		}, // 0
		{
			name:       "p",
			input:      `package p; type T []*int;`,
			outputName: "T",
			output:     FieldInfo{Name: "T", TypeName: "int", IsBasic: true, IsPointer: true, IsArray: true},
		}, // 1
		{
			name:       "",
			input:      `package p; type T [][]int;`,
			outputName: "T",
			output:     `*ast.ArrayType found on "T" (not implemented)`,
		}, // 2
		{
			name:       "",
			input:      `package p; type T []map[string]string;`,
			outputName: "T",
			output:     `*ast.MapType found on "T" (not implemented)`,
		}, // 3
		{
			name:       "",
			input:      `package p; type S struct { Name string }; type T []map[S]string;`,
			outputName: "T",
			output:     `*ast.MapType found on "T" (not implemented)`,
		}, // 4
		{
			name:       "",
			input:      `package p; type S struct { Name string }; type T []*map[S]string;`,
			outputName: "T",
			output:     `*ast.StarExpr found on "T" (not implemented)`,
		}, // 5
		{
			name:       "",
			input:      `package p; type T []struct{ Name string };`,
			outputName: "T",
			output:     `*ast.StructType found on "T" (not implemented)`,
		}, // 6
		{
			name:       "",
			input:      `package p; type T []chan string;`,
			outputName: "T",
			output:     `*ast.ChanType found on "T" (not implemented)`,
		}, // 7
		{
			name:       "",
			input:      `package p; type T []*chan string;`,
			outputName: "T",
			output:     `*ast.StarExpr found on "T" (not implemented)`,
		}, // 8
		{
			name:       "",
			outputName: "T",
			input:      `package p; type T []chan struct{};`,
		}, // 9
		{
			name:       "",
			input:      `package p; type T []*chan *struct{};`,
			outputName: "T",
		}, // 10
		{
			name:       "",
			input:      `package p; type T []*struct{ Name string };`,
			outputName: "T",
			output:     `*ast.StarExpr found on "T" (not implemented)`,
		}, // 11
		{
			name:       "",
			input:      `package p; type S struct{ Name string }; type T []S;`,
			outputName: "T",
			output:     FieldInfo{Name: "T", TypeName: "S", IsStruct: true, IsArray: true},
		}, // 12
		{
			name:       "",
			input:      `package p; type S struct{ Name string }; type T []*S;`,
			outputName: "T",
			output:     FieldInfo{Name: "T", TypeName: "S", IsStruct: true, IsArray: true, IsPointer: true},
		}, // 13
	}

	for idx, cCase := range cases {
		result := PackageInfo{Name: "test", StructDefs: make(map[string]*TypeInfo), ArrayDefs: make(map[string]*FieldInfo)}
		fileSet := token.NewFileSet()
		astNodes, err := parser.ParseFile(fileSet, cCase.name, cCase.input, parser.DeclarationErrors|parser.AllErrors)
		if err != nil {
			t.Fatalf("%d. Fatal error : %v", idx, err)
		}
		var infoErr error
		for _, node := range astNodes.Decls {
			switch nodeType := node.(type) {
			case *ast.GenDecl:
				for _, spec := range nodeType.Specs {
					astSpec := spec.(*ast.TypeSpec)
					switch astSpec.Type.(type) {
					case *ast.ArrayType:
						infoErr = result.ReadArrayInfo(spec, nil, nodeType.Doc)
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
			if !reflect.DeepEqual(result.ArrayDefs[cCase.outputName], cCase.output) {
				t.Errorf("%d. output error :\nexpected\n`%#v`\ngot\n`%#v`", idx, cCase.output, result.ArrayDefs[cCase.outputName])
			}
		}
	}
}

func TestStructDefinition(t *testing.T) {
	cases := []testCase{

		{
			name: "p",
			input: `package p 
					type T struct{
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
					}
					func add(a,b int){}`,
			outputName: "T",
			output: &TypeInfo{
				Name: "T",
				Fields: Fields{
					&FieldInfo{
						TypeName:   "S",
						IsStruct:   true,
						IsEmbedded: true,
						Reference: &TypeInfo{
							Name: "S",
							Fields: Fields{
								&FieldInfo{
									Name:       "Name",
									TypeName:   "string",
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
						TypeName:   "S2",
						IsPointer:  true,
						IsStruct:   true,
						IsEmbedded: true,
						Reference: &TypeInfo{
							Name: "S2",
							Fields: Fields{
								&FieldInfo{
									Name:       "Name",
									TypeName:   "string",
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
						TypeName:   "error",
						IsBasic:    true,
						IsEmbedded: true,
					},
					&FieldInfo{
						Name:       "Name",
						TypeName:   "string",
						IsBasic:    true,
						IsExported: true,
						Tags: &Tags{
							&Tag{Key: "json", Name: "name"},
						},
					},
					&FieldInfo{
						Name:       "PtrName",
						TypeName:   "string",
						IsBasic:    true,
						IsPointer:  true,
						IsExported: true,
					},
					&FieldInfo{
						Name:     "unexported",
						TypeName: "string",
						IsBasic:  true,
					},
				},
			},
		}, // 0 - some fields
		{
			name: "p",
			input: `package p 
					type T struct{
						Itemz	Items ` + "`json:\"itmz\"`" + `
						Pricez	Prices ` + "`json:\"prcz\"`" + `
                    }
					type Prices []Price 
					type Items []*Item
					type Item struct { 
						Name string ` + "`json:\"name\"`" + `
					}
					type Price struct { 
						Name string ` + "`json:\"name\"`" + `
					}`,
			outputName: "T",
			output: &TypeInfo{
				Name: "T",
				Fields: Fields{
					&FieldInfo{
						Name:       "Itemz",
						TypeName:   "Items",
						IsArray:    true,
						IsExported: true,
						Reference: &TypeInfo{
							Name:               "Item",
							ReferenceIsPointer: true,
							Fields: Fields{
								&FieldInfo{
									Name:       "Name",
									TypeName:   "string",
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
						TypeName:   "Prices",
						IsArray:    true,
						IsExported: true,
						Reference: &TypeInfo{
							Name: "Price",
							Fields: Fields{
								&FieldInfo{
									Name:       "Name",
									TypeName:   "string",
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
					type T struct{
						Items
                    }
					type Items []*Item
					type Item struct { 
						Name string
					}`,
			outputName: "T",
			output: &TypeInfo{
				Name: "T",
				Fields: Fields{
					&FieldInfo{
						TypeName:   "Items",
						IsArray:    true,
						IsEmbedded: true,
						Reference: &TypeInfo{
							Name:               "Item",
							ReferenceIsPointer: true,
							Fields: Fields{
								&FieldInfo{
									Name:       "Name",
									TypeName:   "string",
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
					type T struct{
						Child T
						PtrChild *T
                    }`,
			outputName: "T",
			output: &TypeInfo{
				Name: "T",
				Fields: Fields{
					&FieldInfo{
						TypeName:   "Items",
						IsArray:    true,
						IsEmbedded: true,
						Reference: &TypeInfo{
							Name:               "Item",
							ReferenceIsPointer: true,
							Fields: Fields{
								&FieldInfo{
									Name:       "Name",
									TypeName:   "string",
									IsBasic:    true,
									IsExported: true,
								},
							},
						},
					},
				},
			},
		}, // 3 - circular reference
		{
			name: "p",
			input: `package p 
					type T struct{
						S S
                    }
					type S struct{
						T T
                    }`,
			outputName: "T",
			output: &TypeInfo{
				Name: "T",
				Fields: Fields{
					&FieldInfo{
						TypeName:   "Items",
						IsArray:    true,
						IsEmbedded: true,
						Reference: &TypeInfo{
							Name:               "Item",
							ReferenceIsPointer: true,
							Fields: Fields{
								&FieldInfo{
									Name:       "Name",
									TypeName:   "string",
									IsBasic:    true,
									IsExported: true,
								},
							},
						},
					},
				},
			},
		}, // 3 - indirect circular reference
	}

	for idx, cCase := range cases {
		result := PackageInfo{Name: "test", StructDefs: make(map[string]*TypeInfo), ArrayDefs: make(map[string]*FieldInfo)}
		fileSet := token.NewFileSet()
		astNodes, err := parser.ParseFile(fileSet, cCase.name, cCase.input, parser.DeclarationErrors|parser.AllErrors)
		if err != nil {
			t.Fatalf("%d. Fatal (parse) error : %v", idx, err)
		}
		var infoErr error
		for _, node := range astNodes.Decls {
			switch nodeType := node.(type) {
			case *ast.GenDecl:
				for _, spec := range nodeType.Specs {
					astSpec := spec.(*ast.TypeSpec)
					switch astSpec.Type.(type) {
					case *ast.ArrayType:
						infoErr = result.ReadArrayInfo(spec, nil, nodeType.Doc)
						if infoErr != nil {
							t.Fatalf("%d.error reading array : %v", idx, infoErr)
						}
					case *ast.StructType:
						infoErr = result.ReadStructInfo(spec, nil, nodeType.Doc)
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
			if !reflect.DeepEqual(def, cCase.output) {
				var sb1, sb2 strings.Builder
				typed, ok := cCase.output.(*TypeInfo)
				if ok {
					typed.Debug(&sb1, 1)
					def.Debug(&sb2, 1)
					t.Errorf("%d. output error : WANT\n%s\n\t\tGOT\n%s", idx, sb1.String(), sb2.String())
				} else {
					t.Fatalf("error : expecting output to be *TypeInfo and it's not")
				}
			}
		}
	}
}
