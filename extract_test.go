package main

import "testing"

func TestStructBasicTypes(t *testing.T) {
	testCases := []struct {
		test        string
		expected    []Aggregate
		expectedErr error
	}{
		{
			`#include <stdint.h>
			struct test_struct { uint64_t a; };`,
			[]Aggregate{
				{
					Name: "test_struct",
					Fields: []Field{
						{Name: "a", Type: "uint64_t", IsPointer: false},
					},
					Union: false,
				},
			},
			nil,
		},
	}

	for _, testCase := range testCases {
		structs, err := ExtractAggregates("", testCase.test)
		if err != nil {
			if err != testCase.expectedErr {
				t.Errorf("Expected error %v: got %v", testCase.expectedErr, err)
			}
			continue
		}

		for idx, agg := range structs {
			exp := testCase.expected[idx]
			if agg.Name != exp.Name {
				t.Errorf("Expected aggregate name %q: got %q", exp.Name, agg.Name)
			}
			if agg.Union != exp.Union {
				expStruct := exp.Union == false
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
