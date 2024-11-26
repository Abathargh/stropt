package main

// Assume 64bit system

type TypeMeta struct {
	Alignment int
	Size      int
}

var (
	pointerSize  = 8
	pointerAlign = 8
)

var TypeMap = map[string]TypeMeta{
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
