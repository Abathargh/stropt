package main

import (
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"modernc.org/cc/v4"
)

type Context map[string]Aggregate

type PrimitiveKind uint

const (
	BasePKind PrimitiveKind = iota
	PointerPKind
	ArrayPKind
)

type Field struct {
	Name      string
	Type      string
	ArraySize int
	Kind      PrimitiveKind
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

func (ctx Context) firstPass(fields []Field) ([]AggregateMeta, int, error) {
	maxAlign := 0
	resMetas := make([]AggregateMeta, 0, len(fields))
	// First pass: evaluate the max alignment in the struct
	for _, field := range fields {
		// Base type case, just extract and cache locally
		fMeta, isBase := TypeMap[field.Type]
		if isBase {
			size := fMeta.Size
			if field.Kind == ArrayPKind {
				size *= field.ArraySize
			}

			resMetas = append(resMetas, AggregateMeta{
				Size:      size,
				Alignment: fMeta.Alignment,
			})

			if fMeta.Alignment > maxAlign {
				maxAlign = fMeta.Alignment
			}
			continue
		}

		// Pointer value: return system word align/size
		if field.Kind == PointerPKind {
			resMetas = append(resMetas, AggregateMeta{
				Size:      pointerSize,
				Alignment: pointerAlign,
			})

			if pointerAlign > maxAlign {
				maxAlign = pointerAlign
			}
			continue
		}

		// Aggregate case
		subMeta, err := ctx.resolveAggregate(field.Type)
		if err != nil {
			return nil, -1, err
		}

		if field.Kind == ArrayPKind {
			subMeta.Size *= field.ArraySize
		}

		resMetas = append(resMetas, subMeta)
		if subMeta.Alignment > maxAlign {
			maxAlign = fMeta.Alignment
		}
	}
	return resMetas, maxAlign, nil
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
	slices.SortFunc(meta.Layout, func(i, j Layout) int {
		return -(i.alignment - j.alignment)
	})

	agg := ctx[name]
	for idx := range agg.Fields {
		agg.Fields[idx] = meta.Layout[idx].Field
	}

	return ctx.ResolveMeta(name)
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
				declarator := declList.StructDeclaratorList.StructDeclarator

				entryType := getType(curr)
				entryName, isPtr := getField(declarator)
				arraySize, isArr := isArray(declarator)

				var kind PrimitiveKind
				switch {
				case isPtr:
					kind = PointerPKind
				case isArr:
					kind = ArrayPKind
				default:
					kind = BasePKind
				}

				currStruct.Fields = append(currStruct.Fields, Field{
					Name:      entryName,
					Type:      entryType,
					ArraySize: arraySize,
					Kind:      kind,
				})
			}

			var qualName string

			switch {
			case isUnion(def.StructOrUnion):
				currStruct.Kind = UnionKind
				qualName = "union " + name
			default:
				currStruct.Kind = StructKind
				qualName = "struct " + name
			}

			currStruct.Name = qualName
			ctx[qualName] = currStruct
		}
	}
	return ctx, nil
}

// TODO
// All types get the complete str as type so that the cli can show it
// - [x] non typedef structs names should be "struct ...", likewise for unions
//   - [x] typedefed anonymous structs/union (e.g. typedef struct { ... } S; )
//
// - [x] handle pointer types
// - [x] handle multiple pointer types
// - [x] handle array types
//   - [x] handle struct array types (struct x arr[10])
//
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
	case decl.Pointer != nil && decl.Pointer.TypeQualifiers == nil:
		return " *"
	case decl.Pointer != nil && decl.Pointer.TypeQualifiers != nil:
		return " * const"
	default:
		return ""
	}
}

func getField(declr *cc.StructDeclarator) (string, bool) {
	decl := declr.Declarator
	isPtr := false
	if decl.Pointer != nil {
		isPtr = true
	}

	pre := decl.DirectDeclarator
	for inner := pre.DirectDeclarator; pre != nil && inner != nil; {
		pre = inner
		inner = inner.DirectDeclarator
	}

	return pre.Token.SrcStr(), isPtr
}

func isUnion(sou *cc.StructOrUnion) bool {
	return sou.Token.SrcStr() == "union"
}

func isArray(declr *cc.StructDeclarator) (int, bool) {
	dir := declr.Declarator.DirectDeclarator
	if dir.Token.SrcStr() == "[" && dir.Token2.SrcStr() == "]" {
		aExpr := dir.AssignmentExpression
		if aExpr == nil {
			return -1, false
		}

		switch expr := aExpr.(type) {
		case *cc.PrimaryExpression:
			size, err := strconv.ParseInt(expr.Token.SrcStr(), 10, 64)
			if err != nil {
				return -1, false
			}
			return int(size), true
		}
	}
	return -1, false
}
