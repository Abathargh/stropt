package main

import (
	"fmt"
	"reflect"
	"runtime"
	"slices"
	"testing"

	"modernc.org/cc/v4"
)

func TestGetName(t *testing.T) {
	testCases := []struct {
		test     Aggregate
		expected []string
	}{
		{
			Aggregate{"struct s1", "", StructKind, nil},
			[]string{"struct s1", "s1"},
		},
		{
			Aggregate{"", "s2_t", StructKind, nil},
			[]string{"s2_t"},
		},
		{
			Aggregate{"struct s3", "s3_t", StructKind, nil},
			[]string{"struct s3", "s3", "s3_t"},
		},
	}

	for _, testCase := range testCases {
		names := GetAggregateNames(&testCase.test)
		expected := testCase.expected

		slices.Sort(names)
		slices.Sort(expected)

		if !slices.Equal(names, expected) {
			t.Errorf("Names not matching, got %v, expected %v", names, expected)
		}
	}
}

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
		{
			`enum example { test, prova, ssa };
			struct test_inner { int a1; enum example ex; };`,
			map[string]Aggregate{
				"enum example": {
					Name:    "enum example",
					Typedef: "",
					Kind:    EnumKind,
					Fields: []Field{
						EnumEntry("test"),
						EnumEntry("prova"),
						EnumEntry("ssa"),
					},
				},
				"struct test_inner": {
					Name:    "struct test_inner",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Basic{nil, "int", "a1"},
						Basic{nil, "enum example", "ex"},
					},
				},
			},
		},
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
		{
			"typedef enum { test, prova, versuch } example_e;",
			map[string]Aggregate{
				"example_e": {
					Name:    "",
					Typedef: "example_e",
					Kind:    EnumKind,
					Fields: []Field{
						EnumEntry("test"),
						EnumEntry("prova"),
						EnumEntry("versuch"),
					},
				},
			},
		},
		{
			"typedef enum exe { test, prova, versuch } example_e;",
			map[string]Aggregate{
				"example_e": {
					Name:    "enum exe",
					Typedef: "example_e",
					Kind:    EnumKind,
					Fields: []Field{
						EnumEntry("test"),
						EnumEntry("prova"),
						EnumEntry("versuch"),
					},
				},
			},
		},
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
		{
			`struct sub { const char * foo; };
			struct aos { struct sub arr[100]; };`,
			map[string]Aggregate{
				"struct sub": {
					Name:    "struct sub",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Pointer{Basic{[]string{"const"}, "char", "foo"}, nil},
					},
				},
				"struct aos": {
					Name:    "struct aos",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "struct sub", "arr"}, 100},
					},
				},
			},
		},
		{
			`#define ARR_SIZE 100
			struct def { char arr[ARR_SIZE]; };`,
			map[string]Aggregate{
				"struct def": {
					Name:    "struct def",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "char", "arr"}, 100},
					},
				},
			},
		},
		{
			`#define ARR_SIZE (100)
			struct def { char arr[ARR_SIZE]; };`,
			map[string]Aggregate{
				"struct def": {
					Name:    "struct def",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "char", "arr"}, 100},
					},
				},
			},
		},
		{
			`
			typedef struct int_cont {
				volatile int a;
				int * b;
				const int * const c;
				} int_cont_t;`,
			map[string]Aggregate{
				"int_cont_t": {
					Name:    "struct int_cont",
					Typedef: "int_cont_t",
					Kind:    StructKind,
					Fields: []Field{
						Basic{[]string{"volatile"}, "int", "a"},
						Pointer{Basic{nil, "int", "b"}, nil},
						Pointer{Basic{[]string{"const"}, "int", "c"}, []string{"const"}},
					},
				},
			},
		},
		{
			"struct expr1 { char arr[sizeof(double)]};",
			map[string]Aggregate{
				"struct expr1": {
					Name:    "struct expr1",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "char", "arr"}, 8},
					},
				},
			},
		},
		{
			"struct expr2 { char arr[1 + 1]};",
			map[string]Aggregate{
				"struct expr2": {
					Name:    "struct expr2",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "char", "arr"}, 2},
					},
				},
			},
		},
		{
			"struct expr3 { char arr[1 + 1 + 9]};",
			map[string]Aggregate{
				"struct expr3": {
					Name:    "struct expr3",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "char", "arr"}, 11},
					},
				},
			},
		},
		{
			"struct expr4 { char arr[6 - 1]};",
			map[string]Aggregate{
				"struct expr4": {
					Name:    "struct expr4",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "char", "arr"}, 5},
					},
				},
			},
		},
		{
			"struct expr5 { char arr[6 - 1 + sizeof(double) - sizeof(signed int)]};",
			map[string]Aggregate{
				"struct expr5": {
					Name:    "struct expr5",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "char", "arr"}, 9},
					},
				},
			},
		},
		{
			"struct expr6 { char arr[sizeof(double) + 1]};",
			map[string]Aggregate{
				"struct expr6": {
					Name:    "struct expr6",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "char", "arr"}, 9},
					},
				},
			},
		},
		{
			"struct expr7 { char arr[sizeof(double) - 1]};",
			map[string]Aggregate{
				"struct expr7": {
					Name:    "struct expr7",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "char", "arr"}, 7},
					},
				},
			},
		},
		{
			"struct expr8 { char arr[2 * 3]};",
			map[string]Aggregate{
				"struct expr8": {
					Name:    "struct expr8",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "char", "arr"}, 6},
					},
				},
			},
		},
		{
			"struct expr9 { char arr[15 / 3]};",
			map[string]Aggregate{
				"struct expr9": {
					Name:    "struct expr9",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "char", "arr"}, 5},
					},
				},
			},
		},
		{
			"struct expr10 { char arr[6 % 5]};",
			map[string]Aggregate{
				"struct expr10": {
					Name:    "struct expr10",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "char", "arr"}, 1},
					},
				},
			},
		},
		{
			"struct expr11 { char arr[(7 % 4) * sizeof(double) - 2 + (sizeof(int) / 4)]};",
			map[string]Aggregate{
				"struct expr11": {
					Name:    "struct expr11",
					Typedef: "",
					Kind:    StructKind,
					Fields: []Field{
						Array{Basic{nil, "char", "arr"}, 23},
					},
				},
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

			if msg, equal := aggregatesEqual(aggregate, aggCase); !equal {
				t.Errorf("test: %s\n%s", testCase.test, msg)
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

func aggregatesEqual(a1, a2 Aggregate) (string, bool) {
	if a1.Name != a2.Name {
		return fmt.Sprintf("different names, got %q - %q", a1.Name, a2.Name), false
	}

	if a1.Typedef != a2.Typedef {
		return fmt.Sprintf("different typedefs, got %q - %q",
			a1.Typedef, a2.Typedef), false
	}

	if a1.Kind != a2.Kind {
		return fmt.Sprintf("different names, got %d - %d", a1.Kind, a2.Kind), false
	}

	if len(a1.Fields) != len(a2.Fields) {
		return fmt.Sprintf("different field num, got %+v - %+v", a1.Fields,
			a2.Fields), false
	}

	for idx := range len(a1.Fields) {
		if !reflect.DeepEqual(a1.Fields[idx], a2.Fields[idx]) {
			return fmt.Sprintf("different fields at index %d, got %+v - %+v",
				idx, a1.Fields[idx], a2.Fields[idx]), false
		}
	}
	return "", true
}
