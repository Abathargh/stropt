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

type Aggregate struct {
	Name      string
	Fields    []Field
	Union     bool
	size      int
	alignment int
}

var (
	SymbolError = errors.New("cannot find symbol")

	resolved = make(Context)
)

func (ctx Context) GetMeta(name string) (TypeMeta, error) {
	agg, ok := ctx[name]
	if !ok {
		return TypeMeta{-1, -1}, fmt.Errorf("%w: %v", SymbolError, name)
	}

	if agg.size > 0 && agg.alignment > 0 {
		return TypeMeta{
			Alignment: agg.alignment,
			Size:      agg.size,
		}, nil
	}

	// Compute it for the first time

	maxAlign := 0
	size := 0
	for _, field := range agg.Fields {
		fMeta, isBase := TypeMap[field.Type]
		if isBase {
			if fMeta.Alignment > maxAlign {
				maxAlign = fMeta.Alignment
			}

			switch {
			case agg.Union && fMeta.Size > size:
				size = fMeta.Size
			case !agg.Union:
				// here the size to add depends on the previous one and passing must
				// be accounted for and stored somewhere
			}

		}
	}
	return TypeMeta{}, nil
}

func (agg Aggregate) Alignment() int {
	return -1
}

func ExtractAggregates(fname, cont string) ([]Aggregate, error) {
	config, err := cc.NewConfig(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, fmt.Errorf("could not create a config for the parser: %w", err)
	}

	srcs := []cc.Source{
		{Name: "<predefined>", Value: config.Predefined},
		{Name: "<builtin>", Value: cc.Builtin},
		{Name: fname, Value: cont},
	}

	ast, err := cc.Parse(config, srcs)
	if err != nil {
		return nil, fmt.Errorf("could not extract aggregates: %w", err)
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
			currStruct.Union = isUnion(def.StructOrUnion)
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
