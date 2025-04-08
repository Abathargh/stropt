package main

import (
	"errors"
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

var (
	ErrNotanAggregate = errors.New("not an aggregate")
)

func getTypedefToken(decl *cc.Declaration) *cc.Token {
	if decl.DeclarationSpecifiers.Case == cc.DeclarationSpecifiersStorage {
		// anonymous typedef'd enum case: this is found somewhere else
		var (
			declList   = decl.InitDeclaratorList
			initDecl   = declList.InitDeclarator
			directDecl = initDecl.Declarator.DirectDeclarator
		)
		return &directDecl.Token
	}
	return &decl.InitDeclaratorList.Token
}

// ParseAggregate parse a declaration tree in search for an Aggregate
func ParseAggregate(decl *cc.Declaration) (*Aggregate1, error) {
	var ret Aggregate1

	specs := decl.DeclarationSpecifiers

	// if the type was typedef'd, we retrieve the typedef name
	if decl.InitDeclaratorList != nil {
		token := getTypedefToken(decl)
		ret.Typedef = token.SrcStr()
	}

	// check the specifiers list, the type section contains the aggregate one
	if specs.Case == cc.DeclarationSpecifiersStorage {
		specs = specs.DeclarationSpecifiers
	}

	// at this point, a type specifier must be present...
	if specs.TypeSpecifier == nil {
		return nil, ErrNotanAggregate
	}

	// ...as it will lead us to the struct/union specifier
	aggrSpec := specs.TypeSpecifier.StructOrUnionSpecifier
	if aggrSpec == nil {
		return nil, ErrNotanAggregate
	}

	// check which kind of aggregate this is, and its name
	var (
		aggregateId   = aggrSpec.Token.SrcStr()
		aggregateKind = aggrSpec.StructOrUnion.Token.SrcStr()
	)

	switch aggregateKind {
	case "struct":
		ret.Kind = StructKind1
	case "union":
		ret.Kind = UnionKind1
	case "enum":
		ret.Kind = EnumKind1
	}

	// if this is not a anonymous typedef'd struct, we get the name from here
	if aggregateId != "" {
		ret.Name = fmt.Sprintf("%s %s", aggregateKind, aggregateId)
	}

	// let us extract the fields and fully qualify them
	declList := aggrSpec.StructDeclarationList
	for ; declList != nil; declList = declList.StructDeclarationList {
		ret.Fields = append(ret.Fields, ParseField(declList.StructDeclaration))
	}

	return &ret, nil
}

func ParseField(fieldDecl *cc.StructDeclaration) Field1 {
	qualifiers, typeName := parseQualifiers(fieldDecl)
	name, meta, kind := parseName(fieldDecl.StructDeclaratorList)

	switch kind {
	case ValueKind:
		return Basic{qualifiers, typeName, name}
	case PointerKind:
		return Pointer{Basic{qualifiers, typeName, name}, meta.ptrQualifiers}
	case ArrayKind:
		return Array{Basic{qualifiers, typeName, name}, meta.arraySize}
	case FunctionPointerKind:
		return FuncPointer{typeName, name, meta.argsTypes}
	default:
		return nil
	}
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

func parseQualifiers(fieldDecl *cc.StructDeclaration) ([]string, string) {
	var (
		qualifierId string
		qualifiers  []string
	)

	list := fieldDecl.SpecifierQualifierList
	for ; list != nil; list = list.SpecifierQualifierList {
		listCase := list.Case

		switch listCase {
		case cc.SpecifierQualifierListTypeQual:
			qual := list.TypeQualifier
			qualifierId = qual.Token.SrcStr()
		case cc.SpecifierQualifierListTypeSpec:
			qual := list.TypeSpecifier
			switch qual.Case {
			case cc.TypeSpecifierStructOrUnion, cc.TypeSpecifierEnum:
				qualifierId = parseStructOrUnionQualifier(qual.StructOrUnionSpecifier)
			default:
				qualifierId = qual.Token.SrcStr()
			}
		}

		qualifiers = append(qualifiers, qualifierId)
	}

	lastIdx := len(qualifiers) - 1

	if len(qualifiers) > 1 {
		return qualifiers[0:lastIdx], qualifiers[lastIdx]
	}

	return nil, qualifiers[lastIdx]
}

// parseStructOrUnionQualifier
func parseStructOrUnionQualifier(spec *cc.StructOrUnionSpecifier) string {
	qualifierKind := spec.StructOrUnion.Token.SrcStr()
	qualifierName := spec.Token.SrcStr()
	return fmt.Sprintf("%s %s", qualifierKind, qualifierName)
}

type FieldMeta struct {
	ptrQualifiers []string
	argsTypes     []string
	arraySize     int
}

func parseName(list *cc.StructDeclaratorList) (string, FieldMeta, FieldKind) {
	var (
		structDecl = list.StructDeclarator
		decl       = structDecl.Declarator
		direct     = decl.DirectDeclarator
		fieldName  = direct.Token.SrcStr() // this holds the name of the field...
	)

	// ...except in the array field case
	if direct.AssignmentExpression != nil {
		name, size := parseArrayName(direct)
		return name, FieldMeta{arraySize: size}, ArrayKind
	}

	// ...and in the function pointer case
	if direct.ParameterTypeList != nil {
		name, args := parseFunctionPointerName(direct)
		return name, FieldMeta{argsTypes: args}, FunctionPointerKind
	}

	// let us check if this is a pointer field
	if decl.Pointer != nil {
		qualifiers := parsePointerQualifiers(decl.Pointer)
		return fieldName, FieldMeta{ptrQualifiers: qualifiers}, PointerKind
	}

	return fieldName, FieldMeta{}, ValueKind
}

func parsePointerQualifiers(ptr *cc.Pointer) []string {
	var qualifiers []string

	for q := ptr.TypeQualifiers; q != nil; q = q.TypeQualifiers {
		qualifierId := q.TypeQualifier.Token.SrcStr()
		qualifiers = append(qualifiers, qualifierId)
	}

	return qualifiers
}

func parseArrayName(direct *cc.DirectDeclarator) (string, int) {
	// AssignmentExpression not nil if this is being called, this will be a
	// PrimaryExpression, holding the array size
	primaryExpr := direct.AssignmentExpression.(*cc.PrimaryExpression)

	size, _ := strconv.Atoi(primaryExpr.Token.SrcStr())
	name := direct.DirectDeclarator.Token.SrcStr()
	return name, size
}

func parseFunctionPointerName(direct *cc.DirectDeclarator) (string, []string) {
	var (
		evenMoreDirect             = direct.DirectDeclarator
		superDeclarator            = evenMoreDirect.Declarator
		yoIHeardYouLikeDeclarators = superDeclarator.DirectDeclarator
	)

	fptrName := yoIHeardYouLikeDeclarators.Token.SrcStr()
	args := parseParameterList(direct.ParameterTypeList)
	return fptrName, args
}

func parseParameterList(typeList *cc.ParameterTypeList) []string {
	var args []string
	for list := typeList.ParameterList; list != nil; list = list.ParameterList {
		var (
			paramDecl = list.ParameterDeclaration
			declSpec  = paramDecl.DeclarationSpecifiers
			argType   = declSpec.TypeSpecifier.Token.SrcStr()
		)
		args = append(args, argType)
	}
	return args
}
