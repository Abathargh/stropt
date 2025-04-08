package main

import (
	"reflect"
	"runtime"
	"testing"

	"modernc.org/cc/v4"
)

func TestStructBasicTypes(t *testing.T) {
	testCases := []struct {
		test     string
		expected map[string]Aggregate
	}{
		{
			"struct test_struct { long a; };",
			map[string]Aggregate{
				"struct test_struct": {
					Name:    "struct test_struct",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Basic{nil, "long", "a"},
					},
				},
			},
		},
		{
			"union un { double d; float f; unsigned char uc; };",
			map[string]Aggregate{
				"union un": {
					Name:    "union un",
					Typedef: "",
					Kind:    UnionKind,
					Fields: []Field{
						Basic{nil, "double", "d"},
						Basic{nil, "float", "f"},
						Basic{[]string{"unsigned"}, "char", "uc"},
					},
				},
			},
		},
		{
			"struct test_mul { int a; float b; double d; long long l; };",
			map[string]Aggregate{
				"struct test_mul": {
					Name:    "struct test_mul",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Basic{nil, "int", "a"},
						Basic{nil, "float", "b"},
						Basic{nil, "double", "d"},
						Basic{[]string{"long"}, "long", "l"},
					},
				},
			},
		},
		{
			"struct test_ptr { int * a; int arr[100]; };",
			map[string]Aggregate{
				"struct test_ptr": {
					Name:    "struct test_ptr",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Pointer{Basic{nil, "int", "a"}, nil},
						Array{Basic{nil, "int", "arr"}, 100},
					},
				},
			},
		},
		{
			`typedef union {float f; int i; long ui; } un;
			struct test_ptr { int * a; };`,
			map[string]Aggregate{
				"un": {
					Name:    "",
					Typedef: "un",
					Kind:    UnionKind,
					Fields: []Field{
						Basic{nil, "float", "f"},
						Basic{nil, "int", "i"},
						Basic{nil, "long", "ui"},
					},
				},
				"struct test_ptr": {
					Name:    "struct test_ptr",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Pointer{Basic{nil, "int", "a"}, nil},
					},
				},
			},
		},
		{
			"struct test_ptr { const int * a; const int * const b; };",
			map[string]Aggregate{
				"struct test_ptr": {
					Name:    "struct test_ptr",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Pointer{Basic{[]string{"const"}, "int", "a"}, nil},
						Pointer{Basic{[]string{"const"}, "int", "b"}, []string{"const"}},
					},
				},
			},
		},
		{
			`struct inner { int a; };
			struct test_inner { int a1; volatile struct inner a2; };`,
			map[string]Aggregate{
				"struct inner": {
					Name:    "struct inner",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Basic{nil, "int", "a"},
					},
				},
				"struct test_inner": {
					Name:    "struct test_inner",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Basic{nil, "int", "a1"},
						Basic{[]string{"volatile"}, "struct inner", "a2"},
					}},
			},
		},
		// {
		// 	`enum example { test, prova, ssa };
		// 	struct test_inner { int a1; enum example ex; };`,
		// 	map[string]Aggregate1{
		// 		"enum example": {
		// 			Name:    "enum example",
		// 			Typedef: "",
		// 			Kind:    EnumKind1,
		// 			Fields: []Field1{
		// 				EnumEntry("test"),
		// 				EnumEntry("prova"),
		// 				EnumEntry("ssa"),
		// 			},
		// 		},
		// 		"struct test_inner": {
		// 			Name:    "struct test_inner",
		// 			Typedef: "",
		// 			Kind:    StructKind1,
		// 			Fields: []Field1{
		// 				Basic{nil, "int", "a1"},
		// 				Basic{nil, "enum example", "ex"},
		// 			},
		// 		},
		// 	},
		// },
		{
			"typedef struct exs { float disc; double d; char data[50]; } example_t;",
			map[string]Aggregate{
				"example_t": {
					Name:    "struct exs",
					Typedef: "example_t",
					Kind:    StructKind,
					Fields: []Field{
						Basic{nil, "float", "disc"},
						Basic{nil, "double", "d"},
						Array{Basic{nil, "char", "data"}, 50},
					},
				},
			},
		},
		{
			"typedef union { double d; char c; } example_u;",
			map[string]Aggregate{
				"example_u": {
					Name:    "",
					Typedef: "example_u",
					Kind:    UnionKind,
					Fields: []Field{
						Basic{nil, "double", "d"},
						Basic{nil, "char", "c"},
					},
				},
			},
		},
		{
			"typedef union exu { double d; char c; } example_u;",
			map[string]Aggregate{
				"example_u": {
					Name:    "union exu",
					Typedef: "example_u",
					Kind:    UnionKind,
					Fields: []Field{
						Basic{nil, "double", "d"},
						Basic{nil, "char", "c"},
					},
				},
			},
		},
		// {
		// 	"typedef enum { test, prova, versuch } example_e;",
		// 	map[string]Aggregate1{
		// 		"example_e": {
		// 			Name:    "",
		// 			Typedef: "example_e",
		// 			Kind:    EnumKind1,
		// 			Fields: []Field1{
		// 				EnumEntry("test"),
		// 				EnumEntry("prova"),
		// 				EnumEntry("versuch"),
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	"typedef enum exe { test, prova, versuch } example_e;",
		// 	map[string]Aggregate1{
		// 		"example_t": {
		// 			Name:    "enum exe",
		// 			Typedef: "example_e",
		// 			Kind:    EnumKind1,
		// 			Fields: []Field1{
		// 				EnumEntry("test"),
		// 				EnumEntry("prova"),
		// 				EnumEntry("versuch"),
		// 			},
		// 		},
		// 	},
		// },
		{
			"typedef struct { int (*fptr)(int, float); } fptr_t;",
			map[string]Aggregate{
				"fptr_t": {
					Name:    "",
					Typedef: "fptr_t",
					Kind:    StructKind,
					Fields: []Field{
						FuncPointer{"int", "fptr", []string{"int", "float"}},
					},
				},
			},
		},
		{
			`struct inner { int a; };
			struct test_inner { int a1; const struct inner * const a2; };`,
			map[string]Aggregate{
				"struct inner": {
					Name:    "struct inner",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Basic{nil, "int", "a"},
					},
				},
				"struct test_inner": {
					Name:    "struct test_inner",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Basic{nil, "int", "a1"},
						Pointer{Basic{[]string{"const"}, "struct inner", "a2"}, []string{"const"}},
					}},
			},
		},
	}

	for _, testCase := range testCases {
		var aggregates []Aggregate
		ast := initAst(testCase.test)

		for l := ast.TranslationUnit; l != nil; l = l.TranslationUnit {
			ed := l.ExternalDeclaration
			switch ed.Case {
			case cc.ExternalDeclarationDecl:
				switch ed.Declaration.DeclarationSpecifiers.Type().(type) {
				case *cc.StructType, *cc.UnionType, *cc.EnumType:
					agg, err := ParseAggregate(ed.Declaration)
					if err != nil {
						continue
					}
					aggregates = append(aggregates, *agg)
				}
			}
		}

		count := 0
		for _, aggregate := range aggregates {
			name := getAggregateName(aggregate)

			aggCase, ok := testCase.expected[name]
			if !ok {
				continue
			}

			if !reflect.DeepEqual(aggregate, aggCase) {
				t.Errorf("test: %s\nexpected %+v, got %+v", testCase.test,
					aggCase, aggregate)
				continue
			}
			count++
		}

		if count != len(testCase.expected) {
			t.Errorf("test: %s\nexpected %d aggregates, got %d\n complete list %+v",
				testCase.test, len(testCase.expected), count, aggregates)
		}
	}
}

func initAst(data string) *cc.AST {
	config, _ := cc.NewConfig(runtime.GOOS, runtime.GOARCH)

	srcs := []cc.Source{
		{Name: "<predefined>", Value: config.Predefined},
		{Name: "<builtin>", Value: cc.Builtin},
		{Name: "", Value: data},
	}

	ast, _ := cc.Translate(config, srcs)
	return ast
}

func getAggregateName(agg Aggregate) string {
	if agg.Typedef != "" {
		return agg.Typedef
	}
	return agg.Name
}
