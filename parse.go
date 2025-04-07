package main

import (
	"fmt"
	"strconv"
	"strings"

	"modernc.org/cc/v4"
)

type FieldKind uint

const (
	ValueKind = iota
	PointerKind
	ArrayKind
	FunctionPointerKind
)

type AggregateKind1 uint

const (
	StructKind1 = iota
	UnionKind1
	EnumKind1
)

type Basic struct {
	Qualifiers []string
	TypeName   string
	Name       string
}

func (b Basic) Type() string {
	var builder strings.Builder
	for _, qualifier := range b.Qualifiers {
		builder.WriteString(qualifier)
		builder.WriteRune(' ')
	}
	builder.WriteString(b.TypeName)
	return builder.String()
}

type Pointer struct {
	Basic
	PointerQualifiers []string
}

func (p Pointer) Type() string {
	var builder strings.Builder
	for _, qualifier := range p.Qualifiers {
		builder.WriteString(qualifier)
		builder.WriteRune(' ')
	}
	builder.WriteString(" * ")
	builder.WriteString(p.TypeName)
	return builder.String()
}

type Array struct {
	Basic
	Elements int
}

func (a Array) Type() string {
	var builder strings.Builder
	for _, qualifier := range a.Qualifiers {
		builder.WriteString(qualifier)
		builder.WriteRune(' ')
	}
	builder.WriteString(a.TypeName)
	builder.WriteRune('[')
	builder.WriteString(strconv.Itoa(a.Elements))
	builder.WriteRune(']')
	return builder.String()
}

type FuncPointer struct {
	ReturnType string
	Name       string
	Args       []string
}

func (fp FuncPointer) Type() string {
	var builder strings.Builder
	builder.WriteString(fp.ReturnType)
	builder.WriteString(" (*) ")

	for idx, arg := range fp.Args {
		builder.WriteString(arg)
		if idx != len(fp.Args)-1 {
			builder.WriteString(", ")
		}
	}
	return builder.String()
}

type EnumEntry string

func (ee EnumEntry) Type() string {
	return string(ee)
}

type Field1 interface {
	Type() string
}

type Aggregate1 struct {
	Name    string
	Typedef string
	Kind    AggregateKind1
	Fields  []Field1
}

func ParseAggregate(decl *cc.Declaration) Aggregate1 {
	var ret Aggregate1

	// if the type was typedef'd, we etrieve the typedef name
	if decl.InitDeclaratorList != nil {
		ret.Typedef = decl.InitDeclaratorList.Token.SrcStr()
	}

	// check the specifiers list, the type section contains the aggregate one
	specs := decl.DeclarationSpecifiers.DeclarationSpecifiers
	aggrSpec := specs.TypeSpecifier.StructOrUnionSpecifier

	// check which kind of aggregate this is, and its name
	aggregateId := aggrSpec.Token.SrcStr()
	aggregateKind := aggrSpec.StructOrUnion.Token.SrcStr()

	switch aggregateKind {
	case "struct":
		ret.Kind = StructKind1
	case "union":
		ret.Kind = UnionKind1
	case "enum":
		ret.Kind = EnumKind1
	}

	// if this is not a anonymous typedef'd struct, we get the name from here
	if aggregateId != "" { // TODO check
		ret.Name = fmt.Sprintf("%s %s", aggregateKind, aggregateId)
	}

	// let us extract the fields and fully qualify them
	declList := aggrSpec.StructDeclarationList
	for ; declList != nil; declList = declList.StructDeclarationList {
		ret.Fields = append(ret.Fields, ParseField(declList.StructDeclaration))
	}

	return ret
}

func ParseField(fieldDecl *cc.StructDeclaration) Field1 {
	//qualifiers := parseQualifiers(fieldDecl)
	//typeName := qualifiers[len(qualifiers)-1]
	return nil
}

func GetIdentifiers(aggregate *Aggregate1) []string {
	var identifiers []string

	if aggregate.Name != "" {
		identifiers = append(identifiers, aggregate.Name)
	}

	if aggregate.Typedef != "" {
		identifiers = append(identifiers, aggregate.Typedef)
	}
	return identifiers
}

func parseQualifiers(fieldDecl *cc.StructDeclaration) []string {
	var qualifierId string
	var qualifiers []string

	list := fieldDecl.SpecifierQualifierList
	for ; list != nil; list = list.SpecifierQualifierList {
		qual := list.TypeSpecifier
		switch qual.Case {
		// if this is a non-typedef'd aggregate, parse the kind too
		case cc.TypeSpecifierStructOrUnion, cc.TypeSpecifierEnum:
			souSpec := qual.StructOrUnionSpecifier
			qualifierKind := souSpec.StructOrUnion.Token.SrcStr()
			qualifierName := souSpec.Token.SrcStr()
			qualifierId = fmt.Sprintf("%s %s", qualifierKind, qualifierName)
		// otherwise, just the type is fine
		default:
			qualifierId = list.TypeQualifier.Token.SrcStr()
		}

		qualifiers = append(qualifiers, qualifierId)
	}

	return qualifiers
}

func parseName(list *cc.StructDeclaratorList) (string, []string, FieldKind) {
	var qualifiers []string

	structDecl := list.StructDeclarator
	declarator := structDecl.Declarator

	// the direct declarator holds the name of the field
	fieldName := declarator.DirectDeclarator.Token.SrcStr()

	// let us check if this is a pointer field
	if declarator.Pointer != nil {
		qual := declarator.Pointer.TypeQualifiers
		for ; qual != nil; qual = qual.TypeQualifiers {
			qualifierId := qual.TypeQualifier.Token.SrcStr()
			qualifiers = append(qualifiers, qualifierId)
		}

		return fieldName, qualifiers, PointerKind
	}

	return fieldName, nil, ValueKind
}
