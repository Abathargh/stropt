package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

const (
	nameMessage = "usage: stropt [flags] [type name] [source code]"
	helpMessage = `
stropt is a struct optimizer program, which analyzes the C types you pass 
in, and prints back the type size, alignment and layout, including padding 
bytes, alongside with suggestions on how to minimize the type size.

The program attempts to generate such information for the passed "type name". 
If a "type name" and some "source code" are passed, then the program will 
attempt to parse said code and find out the type inside of the passed source 
code as a string.

If no source code is passed as a string, then it is mandatory to use the 
"-file" option, and pass an existing file name.
`

	helpUsage     = "show the help message"
	bareUsage     = "just print the data without table formatting or graphics"
	versionUsage  = "print the version for this build"
	verboseUsage  = "print more information, e.g. sub-aggregate metadata"
	useCompUsage  = "attempts to resolve includes using the system compiler"
	ptrUsage      = "sets the pointer size/alignment, as comma-separated values"
	enumUsage     = "sets the enum size/alignment, as comma-separated values"
	s32bitUsage   = "sets the type size/alignment as on a 32bit system"
	avrUsage      = "sets the type size/alignment as on a AVR system"
	charUsage     = "sets the char size/alignment, as comma-separated values"
	shortUsage    = "sets the short size/alignment, as comma-separated values"
	intUsage      = "sets the int size/alignment, as comma-separated values"
	longUsage     = "sets the long size/alignment, as comma-separated values"
	longLongUsage = "sets the long long size/alignment, as comma-separated" +
		"values"
	floatUsage      = "sets the float size/alignment, as comma-separated values"
	doubleUsage     = "sets the double size/alignment, as comma-separated values"
	longDoubleUsage = "sets the long double size/alignment, as " +
		"comma-separated values"
	optimizeUsage = "suggests an optimized layout and shows related statistics"
	fileUsage     = "pass a file containing the type definitions"

	entryWidth     = 15
	titleWidth     = entryWidth*4 + 3 // 4 entries per row + padding
	structBoxWidth = entryWidth * 2   // 2 boxes per row

	headerColorHex = "#ececec"
	entryColorHex  = "#aeaeae"
)

var (
	Version = ""

	ErrSizeAlignParsing = errors.New("could not parse size/alignment")
	ErrSizeAlignNelem   = errors.New("expected 2 elements")
	ErrSizeAlignValue   = errors.New("size and alignment must not be zero")

	headerColor = lipgloss.Color(headerColorHex)
	entryColor  = lipgloss.Color(entryColorHex)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Width(entryWidth).
			Foreground(headerColor).
			Align(lipgloss.Center)

	structStyle = lipgloss.NewStyle().
			Bold(true).
			Width(entryWidth).
			Foreground(entryColor).
			Align(lipgloss.Center)

	rowStyle = lipgloss.NewStyle().
			Bold(false).
			Width(entryWidth).
			Foreground(entryColor).
			Align(lipgloss.Center)

	titleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Width(titleWidth).
			Foreground(entryColor).
			Align(lipgloss.Center)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Width(structBoxWidth).
			Margin(0, 1, 1, 0).
			Padding(1, 1, 1, 2).
			Align(lipgloss.Left)

	alignSizeMeta = []struct {
		name string
		fun  func(int, int)
	}{
		{"ptr", SetPointerAlignSize},
		{"enum", SetEnumAlignSize},
		{"char", SetCharAlignSize},
		{"short", SetShortAlignSize},
		{"int", SetIntAlignSize},
		{"long", SetLongAlignSize},
		{"long long", SetLongLongAlignSize},
		{"float", SetFloatAlignSize},
		{"double", SetDoubleAlignSize},
		{"long double", SetLongDoubleAlignSize},
	}
)

func main() {
	if Version == "" {
		info, ok := debug.ReadBuildInfo()
		if ok {
			Version = info.Main.Version
		}
	}

	var (
		help     bool
		bare     bool
		useComp  bool
		version  bool
		verbose  bool
		optimize bool

		s32bit bool
		avr    bool

		ptr        string
		enum       string
		char       string
		short      string
		intM       string
		long       string
		longLong   string
		float      string
		double     string
		longDouble string
		file       string
	)

	fs := flag.NewFlagSet("stropt", flag.ExitOnError)
	fs.BoolVar(&help, "help", false, helpUsage)
	fs.BoolVar(&bare, "bare", false, bareUsage)
	fs.BoolVar(&useComp, "use-compiler", false, useCompUsage)
	fs.BoolVar(&version, "version", false, versionUsage)
	fs.BoolVar(&verbose, "verbose", false, verboseUsage)
	fs.BoolVar(&optimize, "optimize", false, optimizeUsage)
	fs.BoolVar(&s32bit, "32bit", false, s32bitUsage)
	fs.BoolVar(&avr, "avr", false, avrUsage)
	fs.StringVar(&ptr, "ptr", "", ptrUsage)
	fs.StringVar(&enum, "enum", "", enumUsage)
	fs.StringVar(&char, "char", "", charUsage)
	fs.StringVar(&short, "short", "", shortUsage)
	fs.StringVar(&intM, "int", "", intUsage)
	fs.StringVar(&long, "long", "", longUsage)
	fs.StringVar(&longLong, "longlong", "", longLongUsage)
	fs.StringVar(&float, "float", "", floatUsage)
	fs.StringVar(&double, "double", "", doubleUsage)
	fs.StringVar(&longDouble, "longdouble", "", longDoubleUsage)
	fs.StringVar(&file, "file", "", fileUsage)

	if err := fs.Parse(os.Args[1:]); err != nil {
		logErrorMessage("could not parse args: %s", err)
	}

	flags := []string{
		ptr, enum, char, short, intM, long, longLong, float, double, longDouble,
	}

	switch {
	case s32bit:
		Set32BitSys()
	case avr:
		SetAvrSys()
	default:
		err := handleSizeAlignOptions(flags)
		if err != nil {
			logError(fmt.Errorf("wrong option value: %w", err))
		}
	}

	switch {
	case help:
		// -help flag, show usage and full help message
		fmt.Printf("%s\n", nameMessage)
		fmt.Printf("%s\n", helpMessage)
		fs.PrintDefaults()
		return
	case version:
		// -version flag, show the current embedded version
		fmt.Printf("stropt %s\n", Version)
		return
	case len(fs.Args()) == 1 && file != "":
		cont, err := os.ReadFile(file)
		if err != nil {
			logErrorMessage("Failed to open file: %v", err)
		}
		stropt(file, fs.Arg(0), string(cont), bare, verbose, optimize, useComp)
	case len(fs.Args()) == 2:
		stropt("", fs.Arg(0), fs.Arg(1), bare, verbose, optimize, useComp)
	default:
		logErrorMessage(nameMessage)
	}
}

func stropt(fname, aggName, cont string, bare, verbose, optimize, comp bool) {
	aggregates, err := ExtractAggregates(fname, cont, comp)
	if err != nil {
		logError(err)
	}

	meta, err := aggregates.ResolveMeta(aggName)
	if err != nil {
		logError(err)
	}

	if bare {
		fmt.Fprintf(os.Stdout, "(def) ")
	} else {
		fmt.Println(titleBox.Render(fmt.Sprintf("stropt - %s", aggName)))

	}
	printAggregateMeta(aggName, meta, false, bare, verbose)

	if optimize {
		optMeta, err := aggregates.Optimize(aggName, meta)
		if err != nil {
			logError(err)
		}

		if optMeta.Size == meta.Size {
			fmt.Println("The passed layout is already minimal")
			return
		}

		if bare {
			fmt.Fprintf(os.Stdout, "(opt) ")
		}

		printAggregateMeta(aggName, optMeta, true, bare, verbose)
		if optimize {
			fmt.Println(lipgloss.JoinHorizontal(
				lipgloss.Top,
				printAggregate(aggName, meta, false),
				printAggregate(aggName, optMeta, true),
			))
		}
	}
}

func handleSizeAlignOptions(flags []string) error {
	for idx, flag := range flags {
		if flag == "" {
			continue
		}

		meta := alignSizeMeta[idx]
		size, align, err := getSizeAlign(flag)
		if err != nil {
			return fmt.Errorf("%s - %s", meta.name, err)
		}

		meta.fun(size, align)
	}
	return nil
}

func getSizeAlign(in string) (int, int, error) {
	list, err := parseIntList(in)
	if err != nil {
		return -1, -1, fmt.Errorf("%w, got '%s'", err, in)
	}

	if list[0] == 0 || list[1] == 0 {
		return -1, -1, ErrSizeAlignValue
	}

	return list[0], list[1], nil
}

func parseIntList(in string) ([]int, error) {
	splitted := strings.Split(in, ",")
	if len(splitted) != 2 {
		return nil, ErrSizeAlignNelem
	}

	list := make([]int, len(splitted))

	for idx, elem := range splitted {
		ielem, err := strconv.ParseInt(elem, 0, 0)
		if err != nil {
			return nil, ErrSizeAlignParsing
		}
		list[idx] = int(ielem)
	}
	return list, nil
}

func printAggregateMeta(name string, meta AggregateMeta, opt, bare, verbose bool) {
	totPadding := 0
	for _, fLayout := range meta.Layout {
		totPadding += fLayout.padding
	}

	typeName := "Name"
	if opt {
		typeName = "Name (opt)"
	}

	t := makeTable(typeName)

	doPrint(name, meta.Size, meta.Alignment, totPadding, t, bare)

	for _, fLayout := range meta.Layout {
		var (
			size  = strconv.Itoa(fLayout.size)
			align = strconv.Itoa(fLayout.alignment)
			pad   = strconv.Itoa(fLayout.padding)
		)

		t.Row(fLayout.Declaration(), size, align, pad)

		if fLayout.subAggregate != nil && verbose {
			for _, sub := range fLayout.subAggregate {
				name := fmt.Sprintf("%s::%s", fLayout.Declaration(), sub.Declaration())
				doPrint(name, sub.size, sub.alignment, sub.padding, t, bare)
			}
		}
	}

	if !bare {
		fmt.Println(t)
	}
}

func makeTable(typeName string) *table.Table {
	return table.New().
		Border(lipgloss.RoundedBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == -1:
				return headerStyle
			case row == 0:
				return structStyle
			default:
				return rowStyle
			}
		}).
		Headers(typeName, "Size", "Alignment", "Padding")
}

func doPrint(name string, size, align, pad int, tab *table.Table, bare bool) {
	if bare {
		fmt.Fprintf(
			os.Stdout, "%s, size: %d, alignment: %d, padding: %d\n",
			name, size, align, pad,
		)
		return
	}

	var (
		sizeStr  = strconv.Itoa(size)
		alignStr = strconv.Itoa(align)
		padStr   = strconv.Itoa(pad)
	)
	tab.Row(name, sizeStr, alignStr, padStr)
}

func printAggregate(name string, meta AggregateMeta, opt bool) string {
	var builder RenderBuilder
	if !opt {
		builder.WriteComment("// default")
	} else {
		builder.WriteComment("// optimized")
	}

	builder.WriteNewline()
	builder.WriteKeyword(name)
	builder.WriteBase(" {")
	builder.WriteNewline()
	for _, field := range meta.Layout {
		var (
			rType = keywordStyle.Render(field.Type())
			rDecl = baseStyle.Render(field.Declaration())
			rSemi = baseStyle.Render(";")
		)

		fmt.Fprintf(&builder, "\t%s %s%s\n", rType, rDecl, rSemi)
	}

	builder.WriteBase("};")
	return builder.String()
}

type RenderBuilder struct {
	strings.Builder
}

var (
	baseStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
	keywordStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#B29BC5"))
	commentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#747893"))
)

func (b *RenderBuilder) WriteBase(s string) {
	b.Builder.WriteString(baseStyle.Render(s))
}

func (b *RenderBuilder) WriteKeyword(s string) {
	b.Builder.WriteString(keywordStyle.Render(s))
}

func (b *RenderBuilder) WriteComment(s string) {
	b.Builder.WriteString(commentStyle.Render(s))
}

func (b *RenderBuilder) WriteNewline() {
	b.Builder.WriteRune('\n')
}

func (b *RenderBuilder) String() string {
	return boxStyle.Render(b.Builder.String())
}

func logError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func logErrorMessage(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}
