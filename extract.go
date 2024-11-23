package main

import (
	"errors"
	"fmt"
	"runtime"

	"modernc.org/cc/v4"
)

type Context map[string]Aggregate

type Field struct {
	Name      string
	Type      string
	IsPointer bool // if this is true do not need to do size/alignment resolution since it is just a pointer
}

type LayoutType interface {
}

type Layout struct {
	Field
	size      int
	alignment int
	padding   int

	subName      string
	subKind      AggregateKind
	subAggregate []Layout
}

type AggregateKind uint

const (
	StructKind AggregateKind = iota
	UnionKind
	PaddingKind
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
	ParseError  = errors.New("cannot parse source")
	SymbolError = errors.New("cannot find symbol")
)

// Maybe this should also return a simil struct type containing the fileds and
// padding, so that the cli can show it later (use charm/lipgloss),
// could also use it for line diagrams to show padding, fields, byte per byte
// like in my notepad drawings, ecc.
//  "github.com/charmbracelet/lipgloss"
/* func main() {
  var style = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#39e75f")).
    //Background(lipgloss.Color("#7D56F4")).
    PaddingTop(2).
    PaddingLeft(4).
    Width(22)
  fmt.Println(style.Render("Hello, kitty"))
}   */

func (ctx Context) ResolveMeta(name string) (AggregateMeta, error) {
	agg, ok := ctx[name]
	if !ok {
		return AggregateMeta{}, fmt.Errorf("%w: %v", SymbolError, name)
	}

	maxAlign := 0
	resMetas := make([]AggregateMeta, len(agg.Fields))
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
				continue
			}
		}

		// Aggregate case, let's check if this type is defined first
		fAgg, isAggregate := ctx[field.Type]
		if !isAggregate {
			return AggregateMeta{}, fmt.Errorf("%w: %v", SymbolError, name)
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
	totSize := 0
	for idx, field := range agg.Fields {
		curr := resMetas[idx]
		totSize += curr.Size

		if idx != len(agg.Fields)-1 {
			next := resMetas[idx+1]
			if thresh := (maxAlign - (totSize % maxAlign)) % next.Alignment; thresh != 0 {
				// got some padding to add
				totSize += thresh
				layouts[idx] = Layout{
					Field:     field,
					size:      curr.Size,
					alignment: curr.Alignment,
					padding:   thresh,
				}
			}
		} else {
			// need to pad for max_align - totSize % curr?

		}
	}

	return AggregateMeta{
		Size:      totSize,
		Alignment: maxAlign,
		Layout:    resMetas,
	}, nil

}

func ExtractAggregates(fname, cont string) ([]Aggregate, error) {
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
		return nil, fmt.Errorf("%w: %w", ParseError, err)
	}

	// StructDeclarationList
	// 	StructDeclaration -> type
	//		StructDeclaratorList
	//			StructDeclarator -> identifier
	//	StructDeclarationList -> next item in structdecl

	var aggregates []Aggregate

	for name, node := range ast.Scope.Nodes {
		switch def := node[0].(type) {
		// do case base type? does it make sense? typedefs to base type dont
		// get parsed well
		case *cc.StructOrUnionSpecifier:
			currStruct := Aggregate{Name: name}
			curr := def.StructDeclarationList
			for ; curr != nil; curr = curr.StructDeclarationList {
				declList := curr.StructDeclaration
				entryType := getType(curr)
				entryName := getField(declList.StructDeclaratorList.StructDeclarator)
				currStruct.Fields = append(currStruct.Fields, Field{
					Name: entryName,
					Type: entryType,
				})
			}

			if isUnion(def.StructOrUnion) {
				currStruct.Kind = UnionKind
			}

			aggregates = append(aggregates, currStruct)
		}
	}
	return aggregates, nil
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
// - [ ] handle func pointers
// - [ ] handle struct types
// - [ ] handle array types
// - [ ] handle pointer types
// - [ ] handle qualified types (e.g. unsigned long, volatile short)
func getType(declList *cc.StructDeclarationList) string {
	if declList != nil && declList.StructDeclaration.SpecifierQualifierList != nil {
		return declList.StructDeclaration.SpecifierQualifierList.TypeSpecifier.Token.SrcStr()
	}
	return "<empty>"
}

func getField(declr *cc.StructDeclarator) string {
	return declr.Declarator.DirectDeclarator.Token.SrcStr()
}

func isUnion(sou *cc.StructOrUnion) bool {
	return sou.Token.SrcStr() == "union"
}
