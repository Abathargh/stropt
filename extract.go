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
	EnumPKind
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
	EnumKind
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

		if field.Kind == EnumPKind {
			resMetas = append(resMetas, AggregateMeta{
				Size:      enumSize,
				Alignment: enumAlign,
			})

			if enumAlign > maxAlign {
				maxAlign = enumAlign
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

func (ctx Context) addEnum(name string, typedef bool) {
	var qualName string
	if typedef {
		qualName = name
	} else {
		qualName = fmt.Sprintf("enum %s", name)
	}

	entry := Aggregate{
		Name: qualName,
		Kind: EnumKind,
	}
	ctx[qualName] = entry
	ctx[name] = entry
}

func (ctx Context) addStruct(name string, str *cc.StructType, typedef bool) {
	currStruct := Aggregate{}
	for i := range str.NumFields() {
		field := str.FieldByIndex(i)

		currStruct.Fields = append(currStruct.Fields, Field{
			Name: field.Name(),
			Type: field.Type().String(),
		})
	}
}
func (ctx Context) addUnion(name string, str *cc.UnionType, typedef bool) {

}

func ExtractAggregates(fname, cont string, useCompiler bool) (Context, error) {
	var err error
	config := &cc.Config{
		Predefined: "int __predefined_declarator;",
	}

	if useCompiler {
		config, err = cc.NewConfig(runtime.GOOS, runtime.GOARCH)
		if err != nil {
			return nil, fmt.Errorf("%w:\n%s Original error: \n\t%w", ErrConfig,
				compErrMsg, err)
		}
	}

	srcs := []cc.Source{
		{Name: "<predefined>", Value: config.Predefined},
		{Name: fname, Value: cont},
	}

	if useCompiler {
		srcs = append(srcs, cc.Source{Name: "<builtin>", Value: cc.Builtin})
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

	for name, nodes := range ast.Scope.Nodes {
		switch node := nodes[0].(type) {
		case *cc.Declarator:
			switch typ := node.Type().(type) {
			case *cc.StructType:
				ctx.addStruct(name, typ, true)
			case *cc.UnionType:
				ctx.addUnion(name, typ, true)
			case *cc.EnumType:
				ctx.addEnum(name, true)
			}
		case *cc.StructOrUnionSpecifier:
			switch typ := node.Type().(type) {
			case *cc.StructType:
				ctx.addStruct(name, typ, false)
			case *cc.UnionType:
				ctx.addUnion(name, typ, false)
			}
		case *cc.EnumSpecifier:
			if _, isEnum := node.Type().(*cc.EnumType); isEnum {
				ctx.addEnum(name, false)
			}
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
	case typeSpec != nil && typeSpec.EnumSpecifier != nil:
		es := typeSpec.EnumSpecifier
		baseType = es.Token.SrcStr() + " " + es.Token2.SrcStr()
	}

	declr := declList.StructDeclaration.StructDeclaratorList.StructDeclarator
	qual := extractQualifier(specQualList)
	ptrQual := getPtrQual(declr)
	return qual + ptrQual + baseType
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
