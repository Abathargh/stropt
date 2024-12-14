package main

import (
	"errors"
	"testing"
)

func TestStructBasicTypes(t *testing.T) {
	testCases := []struct {
		test        string
		expected    Context
		useCompiler bool
		expectedErr error
	}{
		{
			"#include <stdint.h> struct test_struct { uint64_t a; };",
			Context{},
			false,
			ErrConfig,
		},
		{
			"#include <stdint.h> struct test_struct { uint64_t a; };",
			Context{
				"struct test_struct": {
					Name:   "struct test_struct",
					Fields: []Field{{Name: "a", Type: "uint64_t", Kind: BasePKind}},
					Kind:   StructKind,
				},
			},
			true,
			nil,
		},
		{
			"union un { double d; float f; unsigned char uc; };",
			Context{
				"union un": {
					Name: "union un",
					Fields: []Field{
						{Name: "d", Type: "double", Kind: BasePKind},
						{Name: "f", Type: "float", Kind: BasePKind},
						{Name: "uc", Type: "unsigned char", Kind: BasePKind},
					},
					Kind: UnionKind,
				},
			},
			false,
			nil,
		},
		{
			"union un { double d; float f; unsigned char uc; };",
			Context{
				"union un": {
					Name: "union un",
					Fields: []Field{
						{Name: "d", Type: "double", Kind: BasePKind},
						{Name: "f", Type: "float", Kind: BasePKind},
						{Name: "uc", Type: "unsigned char", Kind: BasePKind},
					},
					Kind: UnionKind,
				},
			},
			true,
			nil,
		},
		{
			"struct test_mul { int a; float b; double d; long long l; };",
			Context{
				"struct test_mul": {
					Name: "struct test_mul",
					Fields: []Field{
						{Name: "a", Type: "int", Kind: BasePKind},
						{Name: "b", Type: "float", Kind: BasePKind},
						{Name: "d", Type: "double", Kind: BasePKind},
						{Name: "l", Type: "long long", Kind: BasePKind},
					},
					Kind: StructKind,
				},
			},
			false,
			nil,
		},
		{
			"struct test_mul { int a; float b; double d; long long l; };",
			Context{
				"struct test_mul": {
					Name: "struct test_mul",
					Fields: []Field{
						{Name: "a", Type: "int", Kind: BasePKind},
						{Name: "b", Type: "float", Kind: BasePKind},
						{Name: "d", Type: "double", Kind: BasePKind},
						{Name: "l", Type: "long long", Kind: BasePKind},
					},
					Kind: StructKind,
				},
			},
			true,
			nil,
		},
		{
			"struct test_ptr { int * a; int arr[100]; };",
			Context{
				"struct test_ptr": {
					Name: "struct test_ptr",
					Fields: []Field{
						{Name: "a", Type: "int *", Kind: PointerPKind},
						{Name: "arr", Type: "int", ArraySize: 100, Kind: ArrayPKind},
					},
					Kind: StructKind,
				},
			},
			false,
			nil,
		},
		{
			"struct test_ptr { int * a; int arr[100]; };",
			Context{
				"struct test_ptr": {
					Name: "struct test_ptr",
					Fields: []Field{
						{Name: "a", Type: "int *", Kind: PointerPKind},
						{Name: "arr", Type: "int", ArraySize: 100, Kind: ArrayPKind},
					},
					Kind: StructKind,
				},
			},
			true,
			nil,
		},
		{
			`#include <stdint.h> typedef union {float f; int i; uint64_t ui; } un; 
			struct test_ptr { int * a; };`,
			Context{},
			false,
			ErrConfig,
		},
		{
			`#include <stdint.h> typedef union {float f; int i; uint64_t ui; } un; 
			struct test_ptr { int * a; };`,
			Context{
				"un": {
					Name: "un",
					Fields: []Field{
						{Name: "f", Type: "float", Kind: BasePKind},
						{Name: "i", Type: "int", Kind: BasePKind},
						{Name: "ui", Type: "uint64_t", Kind: BasePKind},
					},
					Kind: UnionKind,
				},
				"struct un": {
					Name: "struct un",
					Fields: []Field{
						{Name: "f", Type: "float", Kind: BasePKind},
						{Name: "i", Type: "int", Kind: BasePKind},
						{Name: "ui", Type: "uint64_t", Kind: BasePKind},
					},
					Kind: UnionKind,
				},
				"struct test_ptr": {
					Name: "struct test_ptr",
					Fields: []Field{
						{Name: "a", Type: "int *", Kind: PointerPKind},
					},
					Kind: StructKind,
				},
			},
			true,
			nil,
		},
		{
			"struct test_ptr { const int * a; const int * const b; };",
			Context{
				"struct test_ptr": {
					Name: "struct test_ptr",
					Fields: []Field{
						{Name: "a", Type: "const int *", Kind: PointerPKind},
						{Name: "b", Type: "const int * const", Kind: PointerPKind},
					},
					Kind: StructKind,
				},
			},
			false,
			nil,
		},
		{
			"struct test_ptr { const int * a; const int * const b; };",
			Context{
				"struct test_ptr": {
					Name: "struct test_ptr",
					Fields: []Field{
						{Name: "a", Type: "const int *", Kind: PointerPKind},
						{Name: "b", Type: "const int * const", Kind: PointerPKind},
					},
					Kind: StructKind,
				},
			},
			true,
			nil,
		},
		{
			`struct inner { int a; };
			struct test_inner { int a1; struct inner a2; };`,
			Context{
				"struct inner": {
					Name: "struct inner",
					Fields: []Field{
						{Name: "a", Type: "int", Kind: BasePKind},
					},
					Kind: StructKind,
				},
				"struct test_inner": {
					Name: "struct test_inner",
					Fields: []Field{
						{Name: "a1", Type: "int", Kind: BasePKind},
						{Name: "a2", Type: "struct inner", Kind: BasePKind},
					},
					Kind: StructKind,
				},
			},
			false,
			nil,
		},
		{
			`struct inner { int a; };
			struct test_inner { int a1; struct inner a2; };`,
			Context{
				"struct inner": {
					Name: "struct inner",
					Fields: []Field{
						{Name: "a", Type: "int", Kind: BasePKind},
					},
					Kind: StructKind,
				},
				"struct test_inner": {
					Name: "struct test_inner",
					Fields: []Field{
						{Name: "a1", Type: "int", Kind: BasePKind},
						{Name: "a2", Type: "struct inner", Kind: BasePKind},
					},
					Kind: StructKind,
				},
			},
			false,
			nil,
		},
	}

	for _, testCase := range testCases {
		structs, err := ExtractAggregates("", testCase.test, testCase.useCompiler)
		if err != nil {
			if errors.Is(err, testCase.expectedErr) {
				t.Errorf("Expected error %v: got %v", testCase.expectedErr, err)
			}
			continue
		}

		for name, agg := range structs {
			exp, ok := testCase.expected[name]
			if !ok {
				t.Errorf("Cannot find name %q in parsed struct", agg.Name)
				continue
			}

			if agg.Name != exp.Name {
				t.Errorf("Expected aggregate name %q: got %q", exp.Name, agg.Name)
			}

			if agg.Kind != exp.Kind {
				expStruct := exp.Kind == UnionKind
				if expStruct {
					t.Errorf("Expected aggregate struct: got union")
				} else {
					t.Errorf("Expected aggregate union: got struct")
				}
			}

			for jdx, aType := range agg.Fields {
				expField := exp.Fields[jdx]
				if aType.Name != expField.Name {
					t.Errorf("Expected field name %q: got %q", expField.Name, aType.Name)
				}
				if aType.Type != expField.Type {
					t.Errorf("Expected field type %q: got %q", expField.Type, aType.Type)
				}
				if aType.Kind != expField.Kind {
					t.Errorf("Expected ptr: %v: got: %v", expField.Kind,
						aType.Kind)
				}
				if aType.Kind == ArrayPKind && aType.ArraySize != expField.ArraySize {
					t.Errorf("Expected array size: %d: got: %d", aType.ArraySize,
						expField.ArraySize)
				}
			}
		}
	}
}

func TestComputeMeta(t *testing.T) {
	testCases := []struct {
		test        string
		name        string
		expSize     int
		expAlig     int
		expLayout   []Layout
		expectedErr error
	}{
		{
			`#include <stdint.h> 
			struct a1 { int32_t a; int8_t b; int16_t c; int32_t d; int64_t e; };`,
			"struct a1",
			24,
			8,
			[]Layout{
				{size: 4, alignment: 4, padding: 0},
				{size: 1, alignment: 1, padding: 1},
				{size: 2, alignment: 2, padding: 0},
				{size: 4, alignment: 4, padding: 4},
				{size: 8, alignment: 8, padding: 0},
			},
			nil,
		},
		{
			`#include <stdint.h> 
			struct a1 { int32_t a; int64_t b; int8_t c; int32_t d; };`,
			"struct a1",
			24,
			8,
			[]Layout{
				{size: 4, alignment: 4, padding: 4},
				{size: 8, alignment: 8, padding: 0},
				{size: 1, alignment: 1, padding: 3},
				{size: 4, alignment: 4, padding: 0},
			},
			nil,
		},
		{
			`#include <stdint.h> 
			struct t1 { int16_t a; int8_t b; };
			struct s1 { int32_t a; struct t1 t; int32_t d; };`,
			"struct s1",
			12,
			4,
			[]Layout{
				{size: 4, alignment: 4, padding: 0},
				{size: 4, alignment: 2, padding: 0, subAggregate: []Layout{
					{size: 2, alignment: 2, padding: 0},
					{size: 1, alignment: 1, padding: 1},
				}},
				{size: 4, alignment: 4, padding: 0},
			},
			nil,
		},
		{
			"struct p1 { char * str; int a; };",
			"struct p1",
			16,
			8,
			[]Layout{
				{size: 8, alignment: 8, padding: 0},
				{size: 4, alignment: 4, padding: 4},
			},
			nil,
		},
		{
			"struct p1 { char * str; int a; float f[100]; };",
			"struct p1",
			416,
			8,
			[]Layout{
				{size: 8, alignment: 8, padding: 0},
				{size: 4, alignment: 4, padding: 0},
				{size: 400, alignment: 4, padding: 4},
			},
			nil,
		},
		{
			`typedef struct T { short b; char c; } T;
			struct p1 { char * str; int a; struct T f[100]; };`,
			"struct p1",
			416,
			8,
			[]Layout{
				{size: 8, alignment: 8, padding: 0},
				{size: 4, alignment: 4, padding: 0},
				{size: 400, alignment: 2, padding: 4, subAggregate: []Layout{
					{size: 2, alignment: 2, padding: 0},
					{size: 1, alignment: 1, padding: 1},
				}},
			},
			nil,
		},
	}

	for _, testCase := range testCases {
		structs, err := ExtractAggregates("", testCase.test, true)
		if err != nil {
			t.Errorf("Unexpected error when parsing %s: %s", testCase.test, err)
			continue
		}

		meta, err := structs.ResolveMeta(testCase.name)
		if err != nil {
			if !errors.Is(err, testCase.expectedErr) {
				t.Errorf("Expected error %v: got %v", testCase.expectedErr, err)
				t.Errorf("%v", structs)
			}
			continue
		}

		if meta.Size != testCase.expSize {
			t.Errorf("Expected size: %d: got: %d", testCase.expSize, meta.Size)
		}

		if meta.Alignment != testCase.expAlig {
			t.Errorf("Expected alignment: %d: got: %d", testCase.expAlig,
				meta.Alignment)
		}

		for idx, layout := range testCase.expLayout {
			actualLayout := meta.Layout[idx]

			if layout.size != actualLayout.size {
				t.Errorf("Expected size for field %s: %d: got: %d",
					actualLayout.Name, layout.size, actualLayout.size,
				)
			}

			if layout.alignment != actualLayout.alignment {
				t.Errorf("Expected alignment for field %s: %d: got: %d",
					actualLayout.Name, layout.alignment, actualLayout.alignment,
				)
			}

			if layout.padding != actualLayout.padding {
				t.Errorf("Expected padding for field %s: %d: got: %d",
					actualLayout.Name, layout.padding, actualLayout.padding,
				)
			}

			if layout.subAggregate != nil {
				for jdx, subLayout := range layout.subAggregate {
					actualSubL := layout.subAggregate[jdx]

					if subLayout.size != actualSubL.size {
						t.Errorf("Expected size for field %s: %d: got: %d",
							actualSubL.Name, subLayout.size, actualSubL.size,
						)
					}

					if subLayout.alignment != actualSubL.alignment {
						t.Errorf("Expected alignment for field %s: %d: got: %d",
							actualSubL.Name, subLayout.alignment, actualSubL.alignment,
						)
					}

					if subLayout.padding != actualSubL.padding {
						t.Errorf("Expected padding for field %s: %d: got: %d",
							actualSubL.Name, subLayout.padding, actualSubL.padding,
						)
					}
				}
			}
		}
	}
}
