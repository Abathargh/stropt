package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"modernc.org/cc/v4"
)

// FieldKind represents the kind of field in an aggregate
type FieldKind uint

const (
	ValueKind FieldKind = iota
	PointerKind
	ArrayKind
	FunctionPointerKind
	EnumEntryKind
)

// AggregateKind represents the kind of aggregate
type AggregateKind uint

const (
	StructKind AggregateKind = iota
	UnionKind
	EnumKind
)

// An Aggregate represents a C aggregate type (struct, union, enum).
type Aggregate struct {
	Name    string
	Typedef string
	Kind    AggregateKind
	Fields  []Field
}

// A Field is an entry that can be found within an aggregate, be it a struct
// or a union. It defines the Type method, useful to get its type description
// as a string.
type Field interface {
	Type() string
	Declaration() string
	UnqualifiedType() string
}

// A Basic field is a field of either a primitive type, or an aggregate type.
// It describes a C value type.
type Basic struct {
	Qualifiers []string
	TypeName   string
	Name       string
}

var (
	keyQualifiers = map[string]struct{}{
		"signed":   {},
		"unsigned": {},
		"long":     {},
	}
)

// Type returns the type of the Basic field.
func (b Basic) Type() string {
	var builder strings.Builder
	for _, qualifier := range b.Qualifiers {
		builder.WriteString(qualifier)
		builder.WriteRune(' ')
	}
	builder.WriteString(b.TypeName)
	return builder.String()
}

// Declaration returns the fully qualified name for the field. For a basic
// field, that's just its name.
func (b Basic) Declaration() string {
	return b.Name
}

// UnqualifiedType returns the underlying type of the field without qualifiers
// that only affect access/storage. Signedness and `longness` are kept.
func (b Basic) UnqualifiedType() string {
	var unqualType []string
	for idx, qual := range b.Qualifiers {
		if _, ok := keyQualifiers[qual]; ok || idx == len(b.Qualifiers)-1 {
			unqualType = append(unqualType, qual)
		}
	}

	unqualType = append(unqualType, b.TypeName)
	return strings.Join(unqualType, " ")
}

// A Pointer is an aggregate field which is a pointer to any Basic type. It
// describes a C pointer type.
type Pointer struct {
	Basic
	PointerQualifiers []string
}

// Type returns the type of the Pointer field.
func (p Pointer) Type() string {
	var builder strings.Builder
	for _, qualifier := range p.Qualifiers {
		builder.WriteString(qualifier)
		builder.WriteRune(' ')
	}

	builder.WriteString(p.TypeName)
	builder.WriteString(" * ")

	for _, pQualifier := range p.PointerQualifiers {
		builder.WriteString(pQualifier)
		builder.WriteRune(' ')
	}

	return builder.String()
}

// UnqualifiedType returns the underlying type of the field without qualifiers
// that only affect access/storage. Signedness and `longness` are kept.
func (p Pointer) UnqualifiedType() string {
	return p.Basic.UnqualifiedType()
}

// Declaration returns the fully qualified name for the field. For a Pointer
// field, that's just its name.
func (p Pointer) Declaration() string {
	return p.Name
}

// An Array is an aggregate field which is an array to any Basic type. It
// describes a C array type.
type Array struct {
	Basic
	Elements int
}

// Type returns the type of the Array field.
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

// UnqualifiedType returns the underlying type of the field without qualifiers
// that only affect access/storage. Signedness and `longness` are kept.
func (a Array) UnqualifiedType() string {
	return a.Basic.UnqualifiedType()
}

// Declaration returns the fully qualified name for the field. For an Array
// field, that's its name and size.
func (a Array) Declaration() string {
	return fmt.Sprintf("%s[%d]", a.Name, a.Elements)
}

// An FuncPointer is an aggregate field which describes a C function pointer.
type FuncPointer struct {
	ReturnType string
	Name       string
	Args       []string
}

// Type returns the type of the FuncPointer field.
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

func (fp FuncPointer) UnqualifiedType() string {
	return fp.Type()
}

// Declaration returns the fully qualified name for the field. For a
// FuncPointer field, that's its name and argument list.
func (f FuncPointer) Declaration() string {
	var builder strings.Builder

	builder.WriteString(f.Name)

	for idx, arg := range f.Args {
		builder.WriteString(arg)
		if idx != len(f.Args)-1 {
			builder.WriteString(", ")
		}
	}

	return builder.String()
}

type EnumEntry string

func (ee EnumEntry) Type() string {
	return string(ee)
}

func (ee EnumEntry) UnqualifiedType() string {
	return string(ee)
}

// Declaration returns the fully qualified name for the field. For an Enum
// field, that's just its name.
func (e EnumEntry) Declaration() string {
	return string(e)
}

var (
	ErrNotAnAggregate = errors.New("not an aggregate")
)

// ParseAggregate parses a declaration tree in search for an Aggregate.
// If it does find one, it returns a pointer to it, otherwise fails reporting
// an error and returning a nil aggregate.
func ParseAggregate(decl *cc.Declaration) (*Aggregate, error) {
	var ret Aggregate

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
		return nil, ErrNotAnAggregate
	}

	// ...as it will lead us to the struct/union/enum specifier
	aggrSpec := specs.TypeSpecifier.StructOrUnionSpecifier
	enumSpec := specs.TypeSpecifier.EnumSpecifier

	if aggrSpec == nil {
		if enumSpec != nil {
			err := parseEnum(enumSpec, &ret)
			if err != nil {
				return nil, err
			}
			return &ret, nil
		}
		return nil, ErrNotAnAggregate
	}

	// check which kind of aggregate this is, and its name
	var (
		aggregateId   = aggrSpec.Token.SrcStr()
		aggregateKind = aggrSpec.StructOrUnion.Token.SrcStr()
	)

	switch aggregateKind {
	case "struct":
		ret.Kind = StructKind
	case "union":
		ret.Kind = UnionKind
	}

	// if this is not a anonymous typedef'd struct, we get the name from here
	if aggregateId != "" {
		ret.Name = fmt.Sprintf("%s %s", aggregateKind, aggregateId)
	}

	// let us extract the fields and fully qualify them
	declList := aggrSpec.StructDeclarationList
	for ; declList != nil; declList = declList.StructDeclarationList {
		ret.Fields = append(ret.Fields, parseField(declList.StructDeclaration))
	}

	return &ret, nil
}

// GetAggregateNames returns the identifier with which a user can refer to the
// passed aggregate.
// If the aggregate is not anonymous, then this function returns both the
// fully qualified name e.g. `struct foo`, and just the unqualified aggregate
// name, e.g. `foo`. If this is a typedef'd type, it returns the typedef name
// for the aggregate. Any combination of the two is possible, but not having
// any name should not be possible.
func GetAggregateNames(aggregate *Aggregate) []string {
	const (
		maxNames  = 3
		structTag = "struct "
	)

	names := make([]string, 0, maxNames)

	if aggregate.Typedef != "" {
		names = append(names, aggregate.Typedef)
	}

	if aggregate.Name != "" {
		structIndex := strings.Index(aggregate.Name, structTag)
		noStructName := aggregate.Name[structIndex+len(structTag):]
		names = append(names, aggregate.Name)
		names = append(names, noStructName)
	}

	return names
}

// parseEnum parses the whole enum in one go, since it's the simplest
// aggregate kind, and no special checks must be performed.
func parseEnum(spec *cc.EnumSpecifier, enum *Aggregate) error {
	// if this is nil, we just know this cannot be any other kind of aggregate
	if spec == nil {
		return ErrNotAnAggregate
	}

	// extract kind and name: typedef should already be here from earlier
	enum.Kind = EnumKind

	// Token is always `enum`, Token2 contains the non-anonymous enum name
	name := spec.Token2.SrcStr()
	if name != "" {
		enum.Name = fmt.Sprintf("enum %s", spec.Token2.SrcStr())
	}

	// add the enum entries one by one, the parsing is quite simple in this case
	for list := spec.EnumeratorList; list != nil; list = list.EnumeratorList {
		entry := list.Enumerator.Token.SrcStr()
		enum.Fields = append(enum.Fields, EnumEntry(entry))
	}
	return nil
}

// parseField is a builder for the Field type. It constructs and returns a
// Field type described by the passed declaration.
func parseField(fieldDecl *cc.StructDeclaration) Field {
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
	case EnumEntryKind:
		return EnumEntry(name)
	default:
		return nil
	}
}

// parseQualifiers checks for qualifiers on the passed declaration and returns
// them, alongside with the type of the declaration, which is contained as the
// last qualifier in the declaration.
func parseQualifiers(fieldDecl *cc.StructDeclaration) ([]string, string) {
	var (
		qualifierId string
		qualifiers  []string
	)

	list := fieldDecl.SpecifierQualifierList
	for ; list != nil; list = list.SpecifierQualifierList {
		switch list.Case {
		case cc.SpecifierQualifierListTypeQual: // case 1: TypeQualifier present
			qual := list.TypeQualifier
			qualifierId = qual.Token.SrcStr()
		case cc.SpecifierQualifierListTypeSpec: // case 2: TypeSpecifier present
			qual := list.TypeSpecifier
			switch qual.Case {
			case cc.TypeSpecifierStructOrUnion:
				qualifierId = parseStructOrUnionQualifier(qual.StructOrUnionSpecifier)
			case cc.TypeSpecifierEnum:
				qualifierId = parseEnumQualifier(qual.EnumSpecifier)
			default:
				qualifierId = qual.Token.SrcStr()
			}
		}
		qualifiers = append(qualifiers, qualifierId)
	}

	// this is the index of the type of the declaration
	lastIdx := len(qualifiers) - 1

	if len(qualifiers) > 1 {
		return qualifiers[0:lastIdx], qualifiers[lastIdx]
	}

	return nil, qualifiers[lastIdx]
}

// parseStructOrUnionQualifier parses the aggregate qualifier in case the
// type is a fully qualified aggregate type.
func parseStructOrUnionQualifier(spec *cc.StructOrUnionSpecifier) string {
	qualifierKind := spec.StructOrUnion.Token.SrcStr()
	qualifierName := spec.Token.SrcStr()
	return fmt.Sprintf("%s %s", qualifierKind, qualifierName)
}

// parseEnumQualifier parses the enum qualifier in case it is a fully
// qualified enum type.
func parseEnumQualifier(spec *cc.EnumSpecifier) string {
	qualifierKind := spec.Token.SrcStr()
	qualifierName := spec.Token2.SrcStr()
	return fmt.Sprintf("%s %s", qualifierKind, qualifierName)
}

// A FieldMeta struct contains information related to the parsed field. It is
// populated differently based on which kind of field is encountered.
type FieldMeta struct {
	ptrQualifiers []string
	argsTypes     []string
	arraySize     int
}

// parseName parses a declaratoer list in search of the field name. At this
// point in the parsing, it also extracts the kind for the Field which is
// being parsed, and any other metadata that may be available within the
// declarator list. This is a list of the pointer qualifiers, the argument
// types for function pointers, and array sizes.
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

// parsePointerQualifiers extracts the pointer qualifiers from the pointer
// description.
func parsePointerQualifiers(ptr *cc.Pointer) []string {
	var qualifiers []string

	for q := ptr.TypeQualifiers; q != nil; q = q.TypeQualifiers {
		qualifierId := q.TypeQualifier.Token.SrcStr()
		qualifiers = append(qualifiers, qualifierId)
	}

	return qualifiers
}

// parseArrayName parses the direct declarator for the array type and returns
// its name, alongside with its size.
func parseArrayName(direct *cc.DirectDeclarator) (string, int) {
	// AssignmentExpression not nil if this is being called, this will be a
	// PrimaryExpression, holding the array size
	primaryExpr := direct.AssignmentExpression.(*cc.PrimaryExpression)

	size, _ := strconv.Atoi(primaryExpr.Token.SrcStr())
	name := direct.DirectDeclarator.Token.SrcStr()
	return name, size
}

// parseFunctionPointerName extracts the function pointer name and argument
// types from the passed direct declarator.
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

// parseParameterList parses the parameter list for a function pointer field.
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

// getTypedefToken extracts the typedef token from the passed declaration.
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
