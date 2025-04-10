package main

import (
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strings"

	"modernc.org/cc/v4"
)

type Context map[string]*Aggregate

type Layout struct {
	Field
	size         int
	alignment    int
	padding      int
	subAggregate []Layout
}

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
	includeErrMsg = `you are using an '#include' directive, but the tool does not 
resolve include paths by default. Use the '-use-compiler' flag to force that
behavior.`
	compErrMsg = `you are attempting to use the system compiler through the 
'-use-compiler' flag, but it could be that you do not have one installed or 
that this tool cannot find it. The tool checks the CC environment variable, cc
alias and gcc executable for a compiler and fails if no one works.`
)

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

func (ctx Context) resolveAggregate(aggType string) (AggregateMeta, error) {
	// Let us check if this type is defined first
	fAgg, isAggregate := ctx[aggType]
	if !isAggregate {
		return AggregateMeta{}, ErrSymbol
	}

	// If so, let us recursively resolve its alignment/size/padding
	subMeta, err := ctx.ResolveMeta(fAgg.Name)
	if err != nil {
		return AggregateMeta{}, err
	}
	return subMeta, nil
}

func (ctx Context) ResolveMeta(name string) (AggregateMeta, error) {
	agg, ok := ctx[name]
	if !ok {
		return AggregateMeta{}, fmt.Errorf("%w: %v", ErrSymbol, name)
	}

	resMetas, maxAlign, err := ctx.firstPass(agg.Fields)
	if err != nil {
		return AggregateMeta{}, fmt.Errorf("name %s: %w", name, err)
	}

	layouts := make([]Layout, len(agg.Fields))

	// Second pass: evaluate alignment/size/padding

	// Simplified case: enum
	if agg.Kind == EnumKind {
		return AggregateMeta{
			Size:      enumSize,
			Alignment: enumAlign,
		}, nil
	}

	// Simplified case: union - info on the padding formula later
	if agg.Kind == UnionKind {
		maxIdx := -1
		for idx, curr := range resMetas {
			if curr.Size > maxIdx {
				maxIdx = idx
			}
		}
		maxElem := resMetas[maxIdx]
		maxSize := maxElem.Size
		padding := (maxAlign - (maxSize % maxAlign)) % maxAlign

		return AggregateMeta{
			Size:      maxElem.Size + padding,
			Alignment: maxAlign,
			Layout: []Layout{{
				Field:   agg.Fields[maxIdx],
				padding: padding,
			}},
		}, nil
	}

	// Other simplified case: array member
	// Other simplified case: pointer (fp, normal)

	totSize := 0
	for idx, field := range agg.Fields {
		curr := resMetas[idx]
		totSize += curr.Size

		// this is the important part: how does one evaluate the correct padding?
		// once the alignment is computed for the aggregate, then there are two
		// cases:
		// - if the field is not the last one, padding must be added if the next
		// field would be misaligned with reference to its natural alignment, if
		// put directly into the next byte after the current field.
		// - if the field is the last one, padding must be added in such a way
		// that, if another aggregate of the same type would be lied next to this
		// one, it would be aligned too.
		padding := 0
		if idx == len(agg.Fields)-1 {
			padding = (maxAlign - (totSize % maxAlign)) % maxAlign
		} else {
			next := resMetas[idx+1]
			padding = (maxAlign - (totSize % maxAlign)) % next.Alignment
		}

		totSize += padding
		layouts[idx] = Layout{
			Field:     field,
			size:      curr.Size,
			alignment: curr.Alignment,
			padding:   padding,
		}

		if curr.Layout != nil {
			// this is an aggregate field, let's add some metadata to the Layout
			layouts[idx].subAggregate = curr.Layout
		}
	}

	return AggregateMeta{
		Size:      totSize,
		Alignment: maxAlign,
		Layout:    layouts,
	}, nil
}

func (ctx Context) Optimize(name string, meta AggregateMeta) (AggregateMeta, error) {
	layout := make([]Layout, len(meta.Layout))
	copy(layout, meta.Layout)

	slices.SortFunc(layout, func(i, j Layout) int {
		return -(i.alignment - j.alignment)
	})

	agg := ctx[name]
	for idx := range agg.Fields {
		agg.Fields[idx] = layout[idx].Field
	}

	return ctx.ResolveMeta(name)
}

func ExtractAggregates(fname, cont string, useCompiler bool) (Context, error) {

	config, err := cc.NewConfig(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, fmt.Errorf("%w:\n%s Original error: \n\t%w", ErrConfig,
			compErrMsg, err)
	}

	srcs := []cc.Source{
		{Name: "<predefined>", Value: config.Predefined},
		{Name: "<builtin>", Value: cc.Builtin},
		{Name: fname, Value: cont},
	}

	ast, err := cc.Translate(config, srcs)
	if err != nil {
		if strings.Contains(err.Error(), "include") {
			return nil, fmt.Errorf("%w:\n%s Original error: \n\t%w", ErrParse,
				includeErrMsg, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrParse, err)
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
