package main

import (
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strings"

	"modernc.org/cc/v4"
)

// A Context object holds the name/aggregate mappings for all parsed
// aggregates within the source code under analysis. A pointer to an aggregate
// may be present under multiple identifiers, which are the Context keys.
type Context map[string]*Aggregate

// A Layout object holds size/alignment/padding information with reference to
// a field of an aggregate. If the field is an Aggregate type, then the
// subAggregate slice is non-nil and contains the layout information for
// each of its sub-fields.
type Layout struct {
	Field
	size         int
	alignment    int
	padding      int
	subAggregate []Layout
}

// An AggregateMeta object holds the size and alignment data related to an
// aggregate in general, and a list of the same layout information for each
// of its fields-
type AggregateMeta struct {
	Size      int
	Alignment int
	Layout    []Layout
}

var (
	ErrConfig = errors.New("could not create a config for the parser")
	ErrParse  = errors.New("cannot parse source")
	ErrSymbol = errors.New("cannot find symbol")
)

const (
	includeErrMsg = `
you are using an '#include' directive, but the tool does not resolve include 
paths by default. Use the '-use-compiler' flag to force that behavior.`
	compErrMsg = `
you are attempting to use the system compiler through the '-use-compiler' 
flag, but it could be that you do not have one installed or that this tool 
cannot find it. The tool checks the CC environment variable, cc alias and gcc 
executable for a compiler and fails if no one works.`
	parseErrMsg = `
you may have a C syntax error or you may be using a type defined somewhere 
else, without including it via '#include'.`

	predefined = `
int __predefined_declarator;

#if defined(__i386__) || defined(__arm__)
typedef unsigned __predefined_size_t;
#else
typedef unsigned long long __predefined_size_t;
#endif
`

	// if the user is not using an include path from a local compiler
	// installation, we add a simplified version of 'stdint.h' so that quick
	// test snippets do not need '#include'
	stdint = `
typedef signed char        int8_t;
typedef short              int16_t;
typedef int                int32_t;
typedef long long          int64_t;

typedef unsigned char      uint8_t;
typedef unsigned short     uint16_t;
typedef unsigned int       uint32_t;
typedef unsigned long long uint64_t;

typedef int8_t             int_least8_t;
typedef int16_t            int_least16_t;
typedef int32_t            int_least32_t;
typedef int64_t            int_least64_t;

typedef uint8_t            uint_least8_t;
typedef uint16_t           uint_least16_t;
typedef uint32_t           uint_least32_t;
typedef uint64_t           uint_least64_t;

typedef int8_t             int_fast8_t;
typedef int16_t            int_fast16_t;
typedef int32_t            int_fast32_t;
typedef int64_t            int_fast64_t;

typedef uint8_t            uint_fast8_t;
typedef uint16_t           uint_fast16_t;
typedef uint32_t           uint_fast32_t;
typedef uint64_t           uint_fast64_t;

typedef long               intptr_t;
typedef unsigned long      uintptr_t;
typedef int64_t            intmax_t;
typedef uint64_t           uintmax_t;
	`
)

// ExtractAggregates parses the passed C source code contained in `cont` to
// produce a mapping of the aggregates in use within a context object instance.
// On error, it returns a description of where the process failed.
// You can specify a name together with the source code passed as input.
//
// Note that by default, this works without a compiler installation, but an
// include path will be needed if any `#include` directives are used. In that
// case, the `useCompiler` flag can be used.
func ExtractAggregates(fname, cont string, useCompiler bool) (Context, error) {
	config, sources, err := getConfigs(useCompiler)
	if err != nil {
		return nil, fmt.Errorf("%w:\n%s Original error: \n\t%w", ErrConfig,
			compErrMsg, err)
	}

	sources = append(sources, cc.Source{Name: fname, Value: cont})

	ast, err := cc.Translate(config, sources)
	if err != nil {
		msg := parseErrMsg
		if strings.Contains(err.Error(), "include") {
			msg = includeErrMsg
		}
		return nil, fmt.Errorf("%w:%s\nOriginal error: \n\t%w", ErrParse, msg, err)
	}

	var ctx = make(Context)

	// let us iterate over all declaration in the translation unit
	for l := ast.TranslationUnit; l != nil; l = l.TranslationUnit {
		extDecl := l.ExternalDeclaration

		// we are only interested in ExternalDeclarationDecl...
		if extDecl.Case != cc.ExternalDeclarationDecl {
			continue
		}

		specifiers := extDecl.Declaration.DeclarationSpecifiers

		// ...and specifically to Structs, Unions and Enums definitions
		switch specifiers.Type().(type) {
		case *cc.StructType, *cc.UnionType, *cc.EnumType:
			aggregate, err := ParseAggregate(extDecl.Declaration)
			if err != nil {
				return nil, err
			}

			names := GetAggregateNames(aggregate)
			for _, name := range names {
				ctx[name] = aggregate
			}
		}
	}
	return ctx, nil
}

// ResolveMeta attempts to get the alignment/size metadata for the aggregate
// identified by name, within an initialized context. This will also
// recursively compute said metadata for any inner aggregate.
func (ctx Context) ResolveMeta(name string) (AggregateMeta, error) {
	// retrieve the aggregate
	agg, ok := ctx[name]
	if !ok {
		return AggregateMeta{}, fmt.Errorf("%w: %v", ErrSymbol, name)
	}

	// then perform the first pass of the algorithm
	resMetas, maxAlign, err := ctx.firstPass(agg.Fields)
	if err != nil {
		return AggregateMeta{}, fmt.Errorf("name %s: %w", name, err)
	}

	// simplified case: enum
	if agg.Kind == EnumKind {
		return AggregateMeta{
			Size:      enumSize,
			Alignment: enumAlign,
		}, nil
	}

	// simplified case: union
	if agg.Kind == UnionKind {
		// find the biggest element in size
		return resolveUnion(agg, resMetas, maxAlign), nil
	}

	// second pass: evaluate alignment/size/padding for structs
	var (
		totSize = 0
		layouts = make([]Layout, 0, len(agg.Fields))
	)

	for idx, field := range agg.Fields {
		curr := resMetas[idx]
		totSize += curr.Size

		// this is the important part: how does one evaluate the correct padding?
		// once the alignment is computed for the aggregate, then there are two
		// cases:
		// - if the field is the last one, padding must be added in such a way
		// that, if another aggregate of the same type would be lied next to this
		// one, it would be aligned too.
		// - if the field is not the last one, padding must be added if the next
		// field would be misaligned with reference to its natural alignment, if
		// put directly into the next byte after the current field.

		var padToAlignment int
		if idx == len(agg.Fields)-1 {
			padToAlignment = maxAlign
		} else {
			padToAlignment = resMetas[idx+1].Alignment
		}

		padding := (maxAlign - (totSize % maxAlign)) % padToAlignment

		totSize += padding
		layouts = append(layouts, Layout{
			Field:     field,
			size:      curr.Size,
			alignment: curr.Alignment,
			padding:   padding,
		})

		// this is an aggregate field, let's add some metadata to the Layout
		if curr.Layout != nil {
			layouts[idx].subAggregate = curr.Layout
		}
	}

	return AggregateMeta{
		Size:      totSize,
		Alignment: maxAlign,
		Layout:    layouts,
	}, nil
}

// Optimize applies the optimization algorithm for minimizing the padding in
// C aggregates on the passed AggregateMeta, returning a new copy where its
// field may have been re-ordered.
func (ctx Context) Optimize(name string, meta AggregateMeta) (AggregateMeta, error) {
	layout := make([]Layout, len(meta.Layout))
	copy(layout, meta.Layout)

	slices.SortFunc(layout, func(i, j Layout) int {
		return -(i.alignment - j.alignment)
	})

	agg := ctx[name]
	if agg.Kind != StructKind {
		return ctx.ResolveMeta(name)
	}

	for idx := range agg.Fields {
		agg.Fields[idx] = layout[idx].Field
	}

	return ctx.ResolveMeta(name)
}

// firstPass implements, as the name suggests, the first pass in the
// metadata resolution algorithm: it computes the max alignment for the
// current aggregate, and handles any kind of field found, recursively
// calling the resolution algorithm once more if it encounter another
// aggregate.
func (ctx Context) firstPass(fields []Field) ([]AggregateMeta, int, error) {
	maxAlign := 0
	resMetas := make([]AggregateMeta, 0, len(fields))

	// First pass: evaluate the max alignment in the struct
	for _, field := range fields {
		switch field.(type) {
		case Basic, Array:
			agg, err := ctx.handleValueType(field)
			if err != nil {
				return nil, -1, err
			}

			resMetas = append(resMetas, agg)
			if agg.Alignment > maxAlign {
				maxAlign = agg.Alignment
			}
		case FuncPointer, Pointer:
			resMetas = append(resMetas, AggregateMeta{
				Size:      pointerSize,
				Alignment: pointerAlign,
			})

			if pointerAlign > maxAlign {
				maxAlign = pointerAlign
			}
		case EnumEntry:
			resMetas = append(resMetas, AggregateMeta{
				Size:      enumSize,
				Alignment: enumAlign,
			})

			if enumAlign > maxAlign {
				maxAlign = enumAlign
			}
		}
	}
	return resMetas, maxAlign, nil
}

// handleValueType implements the value type (and array) handling strategy
// for the metadata resolution algorithm. If the type is a primitive one, it
// checks for its metadata within a lookup table, otherwise it attempts to
// compute the aggregate metadata.
func (ctx Context) handleValueType(field Field) (AggregateMeta, error) {
	arrType, isArray := field.(Array)
	fType := field.UnqualifiedType()

	fMeta, isBase := TypeMap[fType]
	if isBase {
		size := fMeta.Size
		if isArray {
			size *= arrType.Elements
		}

		return AggregateMeta{
			Size:      size,
			Alignment: fMeta.Alignment,
		}, nil
	}

	// Aggregate case
	subMeta, err := ctx.resolveAggregate(fType)
	if err != nil {
		return AggregateMeta{}, err
	}

	if isArray {
		subMeta.Size *= arrType.Elements
	}

	return subMeta, nil
}

// resolveAggregate tries to resolve the sub-aggregate passed by its type.
func (ctx Context) resolveAggregate(aggType string) (AggregateMeta, error) {
	// Let us check if this type is defined first
	fAgg, isAggregate := ctx[aggType]
	if !isAggregate {
		return AggregateMeta{}, fmt.Errorf("%w: inner '%s'", ErrSymbol, aggType)
	}

	// If so, let us recursively resolve its alignment/size/padding
	subMeta, err := ctx.ResolveMeta(fAgg.Name)
	if err != nil {
		return AggregateMeta{}, err
	}
	return subMeta, nil
}

// resolveUnion is a builder for AggregateMeta when encountering an enum.
func resolveUnion(agg *Aggregate, meta []AggregateMeta, max int) AggregateMeta {
	var (
		layouts []Layout
		maxSize = -1
	)

	for idx, curr := range meta {
		if curr.Size > maxSize {
			maxSize = curr.Size
		}

		layouts = append(layouts, Layout{
			Field:     agg.Fields[idx],
			size:      curr.Size,
			alignment: curr.Alignment,
			padding:   0,
		})

		// this is an aggregate field, let's add some metadata to the Layout
		if curr.Layout != nil {
			layouts[idx].subAggregate = curr.Layout
		}
	}

	// if the biggest element in size is not the one with bigger alignment
	// then we must account for some padding -- it's the same case as for
	// padding the last element of a struct
	padding := (max - (maxSize % max)) % max
	layouts[len(layouts)-1].padding = padding

	return AggregateMeta{
		Size:      maxSize + padding,
		Alignment: max,
		Layout:    layouts,
	}
}

// getConfigs initialize the various configurations structs/slices depending
// on if the user wants to use a local compiler include path or not.
func getConfigs(useCompiler bool) (*cc.Config, []cc.Source, error) {
	if !useCompiler {
		abi, err := cc.NewABI(runtime.GOOS, runtime.GOARCH)
		if err != nil {
			return nil, nil, err
		}

		return &cc.Config{ABI: abi}, []cc.Source{
			{Name: "<predefined>", Value: predefined},
			{Name: "stdint.h", Value: stdint},
		}, nil
	}

	config, err := cc.NewConfig(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, nil, err
	}

	return config, []cc.Source{
		{Name: "<predefined>", Value: config.Predefined},
		{Name: "<builtin>", Value: cc.Builtin},
	}, nil
}
