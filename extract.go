package main

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	"modernc.org/cc/v4"
)

type Context map[string]Aggregate

type Field struct {
	Name      string
	Type      string
	IsPointer bool // if this is true do not need to do size/alignment resolution since it is just a pointer
}

type Layout struct {
	Field
	size         int
	alignment    int
	padding      int
	subAggregate []Layout
}

type AggregateKind uint

const (
	StructKind AggregateKind = iota
	UnionKind
)

type Aggregate struct {
	Name   string
	Fields []Field
	Kind   AggregateKind
}

type AggregateMeta struct {
	Size      int
	Alignment int
	Layout    []Layout
}

var (
	ErrParse  = errors.New("cannot parse source")
	ErrSymbol = errors.New("cannot find symbol")
)

func (ctx Context) ResolveMeta(name string) (AggregateMeta, error) {
	agg, ok := ctx[name]
	if !ok {
		return AggregateMeta{}, fmt.Errorf("%w: %v", ErrSymbol, name)
	}

	maxAlign := 0
	resMetas := make([]AggregateMeta, 0, len(agg.Fields))
	layouts := make([]Layout, len(agg.Fields))

	// First pass: evaluate the max alignment in the struct
	for _, field := range agg.Fields {
		// Base type case, just extract and cache locally
		fMeta, isBase := TypeMap[field.Type]
		if isBase {
			resMetas = append(resMetas, AggregateMeta{
				Size:      fMeta.Size,
				Alignment: fMeta.Alignment,
			})

			if fMeta.Alignment > maxAlign {
				maxAlign = fMeta.Alignment
			}
			continue
		}

		// Aggregate case, let's check if this type is defined first
		fAgg, isAggregate := ctx[field.Type]
		if !isAggregate {
			return AggregateMeta{}, fmt.Errorf("%w: %v", ErrSymbol, name)
		}

		// If so, let's recursively resolve its alignment/size/padding
		subMeta, err := ctx.ResolveMeta(fAgg.Name)
		if err != nil {
			return AggregateMeta{}, err
		}

		resMetas = append(resMetas, subMeta)
		if subMeta.Alignment > maxAlign {
			maxAlign = fMeta.Alignment
		}
	}

	// Second pass: evaluate alignment/size/padding

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

func ExtractAggregates(fname, cont string) (Context, error) {
	// TODO: add possible way of selecting the compiler (e.g. avr, arm-none..)
	// TODO: add possible flags to be passed down
	config, err := cc.NewConfig(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, fmt.Errorf("could not create a config for the parser: %w", err)
	}

	// TODO add possible multiple sources
	srcs := []cc.Source{
		{Name: "<predefined>", Value: config.Predefined},
		{Name: "<builtin>", Value: cc.Builtin},
		{Name: fname, Value: cont},
	}

	ast, err := cc.Parse(config, srcs)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParse, err)
	}

	// StructDeclarationList
	// 	StructDeclaration -> type
	//		StructDeclaratorList
	//			StructDeclarator -> identifier
	//	StructDeclarationList -> next item in structdecl

	var ctx = make(Context)

	for name, node := range ast.Scope.Nodes {
		switch def := node[0].(type) {
		// do case base type? does it make sense? typedefs to base type dont
		// get parsed well
		case *cc.StructOrUnionSpecifier:
			currStruct := Aggregate{}
			curr := def.StructDeclarationList
			for ; curr != nil; curr = curr.StructDeclarationList {
				declList := curr.StructDeclaration
				entryType := getType(curr)
				entryName, isPtr := getField(
					declList.StructDeclaratorList.StructDeclarator,
				)

				currStruct.Fields = append(currStruct.Fields, Field{
					Name:      entryName,
					Type:      entryType,
					IsPointer: isPtr,
				})
			}

			qualName := "struct " + name
			if isUnion(def.StructOrUnion) {
				currStruct.Kind = UnionKind
				qualName = "union " + name
			}

			currStruct.Name = qualName
			ctx[qualName] = currStruct
		}
	}
	return ctx, nil
}

func GetSizeAndAlign(typeName string, types []Aggregate) (int, int, error) {
	meta, ok := TypeMap[typeName]
	if ok {
		return meta.Alignment, meta.Size, nil
	}
	return -1, -1, nil
}

// TODO
// All types get the complete str as type so that the cli can show it
// - [x] non typedef structs names should be "struct ...", likewise for unions
// - [x] handle pointer types
// - [ ] handle array types
// - [x] handle qualified types (e.g. unsigned long, volatile short)
// - [ ] handle func pointers
func getType(declList *cc.StructDeclarationList) string {
	if declList == nil || declList.StructDeclaration == nil ||
		declList.StructDeclaration.SpecifierQualifierList == nil {
		return "error"
	}

	specQualList := declList.StructDeclaration.SpecifierQualifierList
	typeSpec := specQualList.TypeSpecifier

	baseType := ""

	switch {
	case typeSpec != nil && typeSpec.StructOrUnionSpecifier != nil:
		sou := typeSpec.StructOrUnionSpecifier
		baseType = sou.StructOrUnion.Token.SrcStr() + " " + sou.Token.SrcStr()
	}

	declr := declList.StructDeclaration.StructDeclaratorList.StructDeclarator
	return extractQualifier(specQualList) + getPtrQual(declr) + baseType
}

func extractQualifier(specQualList *cc.SpecifierQualifierList) string {
	var builder strings.Builder
	switch {
	case specQualList.TypeSpecifier != nil:
		builder.WriteString(specQualList.TypeSpecifier.Token.SrcStr())
	case specQualList.TypeQualifier != nil:
		builder.WriteString(specQualList.TypeQualifier.Token.SrcStr())
	case specQualList.TypeQualifier == nil && specQualList.TypeSpecifier == nil:
		return ""
	}

	curr := specQualList.SpecifierQualifierList
	for ; curr != nil; curr = curr.SpecifierQualifierList {
		switch {
		case curr.TypeQualifier != nil:
			builder.WriteRune(' ')
			builder.WriteString(curr.TypeQualifier.Token.SrcStr())
		case curr.TypeSpecifier != nil:
			builder.WriteRune(' ')
			builder.WriteString(curr.TypeSpecifier.Token.SrcStr())
		}
	}
	return builder.String()
}

func getPtrQual(declr *cc.StructDeclarator) string {
	decl := declr.Declarator
	switch {
	case decl.Pointer == nil:
		return ""
	case decl.Pointer != nil && decl.Pointer.TypeQualifiers != nil:
		return " * const"
	default:
		return " *"
	}
}

func getField(declr *cc.StructDeclarator) (string, bool) {
	decl := declr.Declarator
	isPtr := false
	if decl.Pointer != nil {
		isPtr = true
	}
	return decl.DirectDeclarator.Token.SrcStr(), isPtr
}

func isUnion(sou *cc.StructOrUnion) bool {
	return sou.Token.SrcStr() == "union"
}
