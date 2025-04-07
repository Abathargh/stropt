package main

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"

	"modernc.org/cc/v4"
)

func TestStructBasicTypes(t *testing.T) {
	testCases := []struct {
		test     string
		expected map[string]Aggregate1
	}{
		{
			"#include <stdint.h> struct test_struct { uint64_t a; };",
			map[string]Aggregate1{
				"struct test_struct": {
					Name:    "struct test_struct",
					Typedef: "",
					Kind:    StructKind1,
					Fields: []Field1{
						Basic{nil, "uint64_t", "a"},
					},
				},
			},
		},
		{
			"union un { double d; float f; unsigned char uc; };",
			map[string]Aggregate1{
				"union un": {
					Name:    "union un",
					Typedef: "",
					Kind:    StructKind1,
					Fields: []Field1{
						Basic{nil, "double", "d"},
						Basic{nil, "float", "f"},
						Basic{[]string{"unsigned"}, "char", "uc"},
					},
				},
			},
		},
		{
			"struct test_mul { int a; float b; double d; long long l; };",
			map[string]Aggregate1{
				"struct test_mul": {
					Name:    "struct test_mul",
					Typedef: "",
					Kind:    StructKind1,
					Fields: []Field1{
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
			map[string]Aggregate1{
				"struct test_ptr": {
					Name:    "struct test_ptr",
					Typedef: "",
					Kind:    StructKind1,
					Fields: []Field1{
						Pointer{Basic{nil, "int", "a"}, nil},
						Array{Basic{nil, "int", "arr"}, 100},
					},
				},
			},
		},
		{
			`#include <stdint.h> typedef union {float f; int i; uint64_t ui; } un; 
			struct test_ptr { int * a; };`,
			map[string]Aggregate1{
				"un": {
					Name:    "",
					Typedef: "un",
					Kind:    UnionKind1,
					Fields: []Field1{
						Basic{nil, "float", "f"},
						Basic{nil, "int", "i"},
						Basic{nil, "uint64_t", "ui"},
					},
				},
				"struct test_ptr": {
					Name:    "struct test_ptr",
					Typedef: "un",
					Kind:    StructKind1,
					Fields: []Field1{
						Pointer{Basic{nil, "int", "f"}, nil},
					},
				},
			},
		},
		{
			"struct test_ptr { const int * a; const int * const b; };",
			map[string]Aggregate1{
				"struct test_ptr": {
					Name:    "struct test_ptr",
					Typedef: "",
					Kind:    StructKind1,
					Fields: []Field1{
						Pointer{Basic{[]string{"const"}, "int", "a"}, nil},
						Pointer{Basic{[]string{"const"}, "int", "b"}, []string{"const"}},
					},
				},
			},
		},
		{
			`struct inner { int a; };
			struct test_inner { int a1; struct inner a2; };`,
			map[string]Aggregate1{
				"struct inner": {
					Name:    "struct inner",
					Typedef: "",
					Kind:    StructKind1,
					Fields: []Field1{
						Basic{nil, "int", "a"},
					},
				},
				"struct test_inner": {
					Name:    "struct test_inner",
					Typedef: "",
					Kind:    StructKind1,
					Fields: []Field1{
						Basic{nil, "int", "a1"},
						Basic{nil, "struct inner", "a2"},
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
			map[string]Aggregate1{
				"example_t": {
					Name:    "struct exs",
					Typedef: "example_t",
					Kind:    StructKind1,
					Fields: []Field1{
						Basic{nil, "float", "disc"},
						Basic{nil, "double", "d"},
						Array{Basic{nil, "char", "data"}, 50},
					},
				},
			},
		},
		{
			"typedef union { double d; char c; } example_u;",
			map[string]Aggregate1{
				"example_u": {
					Name:    "",
					Typedef: "example_u",
					Kind:    UnionKind1,
					Fields: []Field1{
						Basic{nil, "double", "d"},
						Basic{nil, "char", "c"},
					},
				},
			},
		},
		{
			"typedef union exu { double d; char c; } example_u;",
			map[string]Aggregate1{
				"example_u": {
					Name:    "union exu",
					Typedef: "example_u",
					Kind:    UnionKind1,
					Fields: []Field1{
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
			"typedef struct { int fptr(int, float); } fptr_t;",
			map[string]Aggregate1{
				"example_t": {
					Name:    "",
					Typedef: "fptr_t",
					Kind:    StructKind1,
					Fields: []Field1{
						FuncPointer{"int", "fptr", []string{"int", "float"}},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		var aggregates []Aggregate1
		fmt.Printf("Testing: '%s'\n", testCase.test)
		ast := initAst(testCase.test)

		for l := ast.TranslationUnit; l != nil; l = l.TranslationUnit {
			ed := l.ExternalDeclaration
			switch ed.Case {
			case cc.ExternalDeclarationDecl:
				aggregates = append(aggregates, ParseAggregate(ed.Declaration))
			}
		}

		for _, aggregate := range aggregates {
			name := getAggregateName(aggregate)
			aggCase, ok := testCase.expected[name]
			if !ok {
				t.Errorf("aggregate name not found: '%s'", name)
				continue
			}

			if !reflect.DeepEqual(aggregate, aggCase) {
				t.Errorf("expected %v, got %v", aggCase, aggregate)
			}
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

func getAggregateName(agg Aggregate1) string {
	if agg.Typedef != "" {
		return agg.Typedef
	}
	return agg.Name
}
