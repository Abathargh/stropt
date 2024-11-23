package main

import (
	"errors"
	"testing"
)

func TestStructBasicTypes(t *testing.T) {
	testCases := []struct {
		test        string
		expected    Context
		expectedErr error
	}{
		{
			"#include <stdint.h> struct test_struct { uint64_t a; };",
			Context{
				"test_struct": {
					Name:   "test_struct",
					Fields: []Field{{Name: "a", Type: "uint64_t", IsPointer: false}},
					Kind:   StructKind,
				},
			},
			nil,
		},
		{
			"struct test_mul { int a; float b; double d; long long l; };",
			Context{
				"test_mul": {
					Name: "test_mul",
					Fields: []Field{
						{Name: "a", Type: "int", IsPointer: false},
						{Name: "b", Type: "float", IsPointer: false},
						{Name: "d", Type: "double", IsPointer: false},
						{Name: "l", Type: "long long", IsPointer: false},
					},
					Kind: StructKind,
				},
			},
			nil,
		},
		{
			"struct test_ptr { int * a; void (*fp)(float, double); int arr[100]; };",
			Context{
				"test_ptr": {
					Name: "test_ptr",
					Fields: []Field{
						{Name: "a", Type: "int", IsPointer: true},
						{Name: "fp", Type: "", IsPointer: true},
						{Name: "arr", Type: "int[100]", IsPointer: false},
					},
					Kind: StructKind,
				},
			},
			nil,
		},
		{
			`#include <stdint.h> typedef union {float f; int i; uint64_t ui; } un; 
			struct test_ptr { int * a; void (*fp)(float, double); };`,
			Context{
				"un": {
					Name: "un",
					Fields: []Field{
						{Name: "f", Type: "float", IsPointer: false},
						{Name: "i", Type: "int", IsPointer: false},
						{Name: "ui", Type: "uint64_t", IsPointer: false},
					},
					Kind: UnionKind,
				},
				"test_ptr": {
					Name: "test_ptr",
					Fields: []Field{
						{Name: "a", Type: "int*", IsPointer: true},
						{Name: "fp", Type: "", IsPointer: true},
						{Name: "arr", Type: "int[100]", IsPointer: false},
					},
					Kind: StructKind,
				},
			},
			nil,
		},
		{
			`struct test_ptr { const int * a; const int * const b; };`,
			Context{
				"test_ptr": {
					Name: "test_ptr",
					Fields: []Field{
						{Name: "a", Type: "const int *", IsPointer: true},
						{Name: "b", Type: "const int *const", IsPointer: true},
					},
					Kind: StructKind,
				},
			},
			nil,
		},
		// add struct in struct case
	}

	for _, testCase := range testCases {
		structs, err := ExtractAggregates("", testCase.test)
		if err != nil {
			if errors.Is(err, testCase.expectedErr) {
				t.Errorf("Expected error %v: got %v", testCase.expectedErr, err)
			}
			continue
		}

		for name, agg := range structs {
			exp := testCase.expected[name]
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
				if aType.IsPointer != expField.IsPointer {
					t.Errorf("Expected ptr: %v: got: %v", expField.IsPointer,
						aType.IsPointer)
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
		expectedErr error
	}{
		{
			`#include <stdint.h> 
			struct a1 { int32_t a; int8_t b; int16_t c; int32_t d; int64_t e; };`,
			"a1",
			8,
			24,
			nil,
		},
		{
			`#include <stdint.h> 
			struct a1 { int32_t a; int64_t b; int8_t c; int32_t d; };`,
			"a1",
			8,
			24,
			nil,
		},
	}

	for _, testCase := range testCases {
		structs, err := ExtractAggregates("", testCase.test)
		if err != nil {
			t.Errorf("Unexpected error when parsing %s", testCase.test)
			continue
		}

		meta, err := structs.ResolveMeta(testCase.name)
		if err != nil {
			if errors.Is(err, testCase.expectedErr) {
				t.Errorf("Expected error %v: got %v", testCase.expectedErr, err)
			}
			continue
		}

		if meta.Size != testCase.expSize {
			t.Errorf("Expected size: %d: got: %d", testCase.expSize, meta.Size)
		}

		if meta.Alignment != testCase.expAlig {
			t.Errorf("Expected size: %d: got: %d", testCase.expAlig, meta.Alignment)
		}

		// add layout checks later
	}
}
