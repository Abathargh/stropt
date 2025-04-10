package main

import (
	"errors"
	"testing"
)

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
			struct a2 { int32_t a; int64_t b; int8_t c; int32_t d; };`,
			"struct a2",
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
			"struct p2 { char * str; int a; float f[100]; };",
			"struct p2",
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
			struct p3 { char * str; int a; struct T f[100]; };`,
			"struct p3",
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
		{
			`typedef enum E { a, b, c } E;
			struct p4 { enum E en; char * str; int a; };`,
			"struct p4",
			24,
			8,
			[]Layout{
				{size: 4, alignment: 4, padding: 4},
				{size: 8, alignment: 8, padding: 0},
				{size: 4, alignment: 4, padding: 4},
			},
			nil,
		},
		{
			"typedef struct { char * str; int a; } p1_t;",
			"p1_t",
			16,
			8,
			[]Layout{
				{size: 8, alignment: 8, padding: 0},
				{size: 4, alignment: 4, padding: 4},
			},
			nil,
		},
		{
			"typedef struct p1_plain{ char * str; int a; } p1_t;",
			"struct p1_plain",
			16,
			8,
			[]Layout{
				{size: 8, alignment: 8, padding: 0},
				{size: 4, alignment: 4, padding: 4},
			},
			nil,
		},
		{
			"typedef struct p1_plain{ char * str; int a; } p1_t;",
			"p1_plain",
			16,
			8,
			[]Layout{
				{size: 8, alignment: 8, padding: 0},
				{size: 4, alignment: 4, padding: 4},
			},
			nil,
		},
		{
			"typedef struct p6 { char * str; int a; } p6_t;",
			"struct p6",
			16,
			8,
			[]Layout{
				{size: 8, alignment: 8, padding: 0},
				{size: 4, alignment: 4, padding: 4},
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
					actualLayout.Declaration(), layout.size, actualLayout.size,
				)
			}

			if layout.alignment != actualLayout.alignment {
				t.Errorf("Expected alignment for field %s: %d: got: %d",
					actualLayout.Declaration(), layout.alignment, actualLayout.alignment,
				)
			}

			if layout.padding != actualLayout.padding {
				t.Errorf("Expected padding for field %s: %d: got: %d",
					actualLayout.Declaration(), layout.padding, actualLayout.padding,
				)
			}

			if layout.subAggregate != nil {
				for jdx, subLayout := range layout.subAggregate {
					actualSubL := layout.subAggregate[jdx]

					if subLayout.size != actualSubL.size {
						t.Errorf("Expected size for field %s: %d: got: %d",
							actualSubL.Declaration(), subLayout.size, actualSubL.size,
						)
					}

					if subLayout.alignment != actualSubL.alignment {
						t.Errorf("Expected alignment for field %s: %d: got: %d",
							actualSubL.Declaration(), subLayout.alignment,
							actualSubL.alignment,
						)
					}

					if subLayout.padding != actualSubL.padding {
						t.Errorf("Expected padding for field %s: %d: got: %d",
							actualSubL.Declaration(), subLayout.padding, actualSubL.padding,
						)
					}
				}
			}
		}
	}
}
