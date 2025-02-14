package main

// Assume 64bit system

type TypeMeta struct {
	Alignment int
	Size      int
}

var (
	pointerSize  = 8
	pointerAlign = 8

	enumSize  = 4
	enumAlign = 4

	charTypes = []string{
		"char",
		"unsigned char",
	}

	shortTypes = []string{
		"short",
		"short int",
		"signed short",
		"signed short int",
		"unsigned short",
		"unsigned short int",
	}

	intTypes = []string{
		"int",
		"signed",
		"signed int",
		"unsigned ",
		"unsigned int",
	}

	longTypes = []string{
		"long",
		"long int",
		"signed long",
		"signed long int",
		"unsigned long",
		"unsigned long int",
	}

	longlongTypes = []string{
		"long long",
		"long long int",
		"signed long long",
		"signed long long int",
		"unsigned long long",
		"unsigned long long int",
	}

	TypeMap = map[string]TypeMeta{
		"char":                   {1, 1},
		"signed char":            {1, 1},
		"unsigned char":          {1, 1},
		"short":                  {2, 2},
		"short int":              {2, 2},
		"signed short":           {2, 2},
		"signed short int":       {2, 2},
		"unsigned short":         {2, 2},
		"unsigned short int":     {2, 2},
		"int":                    {4, 4},
		"signed":                 {4, 4},
		"signed int":             {4, 4},
		"unsigned":               {4, 4},
		"unsigned int":           {4, 4},
		"long":                   {8, 8},
		"long int":               {8, 8},
		"signed long":            {8, 8},
		"signed long int":        {8, 8},
		"unsigned long":          {8, 8},
		"unsigned long int":      {8, 8},
		"long long":              {8, 8},
		"long long int":          {8, 8},
		"signed long long":       {8, 8},
		"signed long long int":   {8, 8},
		"unsigned long long":     {8, 8},
		"unsigned long long int": {8, 8},
		"float":                  {4, 4},
		"double":                 {8, 8},
		"long double":            {16, 16},
		"int8_t":                 {1, 1},
		"uint8_t":                {1, 1},
		"int16_t":                {2, 2},
		"uint16_t":               {2, 2},
		"int32_t":                {4, 4},
		"uint32_t":               {4, 4},
		"int64_t":                {8, 8},
		"uint64_t":               {8, 8},
		"intptr_t":               {8, 8},
		"uintptr_t":              {8, 8},
		"int_least8_t":           {1, 1},
		"uint_least8_t":          {1, 1},
		"int_least16_t":          {2, 2},
		"uint_least16_t":         {2, 2},
		"int_least32_t":          {4, 4},
		"uint_least32_t":         {4, 4},
		"int_least64_t":          {8, 8},
		"uint_least64_t":         {8, 8},
		"int_fast8_t":            {1, 1},
		"uint_fast8_t":           {1, 1},
		"int_fast16_t":           {8, 8},
		"uint_fast16_t":          {8, 8},
		"int_fast32_t":           {8, 8},
		"uint_fast32_t":          {8, 8},
		"int_fast64_t":           {8, 8},
		"uint_fast64_t":          {8, 8},
		"intmax_t":               {8, 8},
		"uintmax_t":              {8, 8},
	}
)

func SetAvrSys() {
	SetPointerAlignSize(1, 2)
	SetEnumAlignSize(1, 2)
	SetCharAlignSize(1, 1)
	SetShortAlignSize(1, 2)
	SetIntAlignSize(1, 2)
	SetLongAlignSize(1, 4)
	SetLongLongAlignSize(1, 8)
	SetFloatAlignSize(1, 4)
	SetDoubleAlignSize(1, 4)
	SetLongDoubleAlignSize(1, 8)
}

func Set32BitSys() {
	SetPointerAlignSize(4, 4)
	SetEnumAlignSize(4, 4)
	SetCharAlignSize(1, 1)
	SetShortAlignSize(2, 2)
	SetIntAlignSize(4, 4)
	SetLongAlignSize(4, 4)
	SetLongLongAlignSize(4, 8)
	SetFloatAlignSize(4, 4)
	SetDoubleAlignSize(4, 8)
	SetLongDoubleAlignSize(4, 12)
}

func SetPointerAlignSize(alignment, size int) {
	pointerAlign = alignment
	pointerSize = size
}

func SetEnumAlignSize(alignment, size int) {
	enumAlign = alignment
	enumSize = size
}

func SetCharAlignSize(alignment, size int) {
	for idx := range charTypes {
		TypeMap[charTypes[idx]] = TypeMeta{alignment, size}
	}
}

func SetShortAlignSize(alignment, size int) {
	for idx := range charTypes {
		TypeMap[shortTypes[idx]] = TypeMeta{alignment, size}
	}
}

func SetIntAlignSize(alignment, size int) {
	for idx := range charTypes {
		TypeMap[intTypes[idx]] = TypeMeta{alignment, size}
	}
}

func SetLongAlignSize(alignment, size int) {
	for idx := range charTypes {
		TypeMap[longTypes[idx]] = TypeMeta{alignment, size}
	}
}

func SetLongLongAlignSize(alignment, size int) {
	for idx := range charTypes {
		TypeMap[longlongTypes[idx]] = TypeMeta{alignment, size}
	}
}

func SetFloatAlignSize(alignment, size int) {
	TypeMap["float"] = TypeMeta{alignment, size}
}

func SetDoubleAlignSize(alignment, size int) {
	TypeMap["double"] = TypeMeta{alignment, size}
}

func SetLongDoubleAlignSize(alignment, size int) {
	TypeMap["long double"] = TypeMeta{alignment, size}
}
