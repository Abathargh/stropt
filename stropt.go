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

		Version = "" // leave this empty, it gets filled elsewhere
	)

	info, ok := debug.ReadBuildInfo()
	if ok {
		Version = info.Main.Version
	}

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
			logErrorMessage("failed to open file: %v", err)
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
		if !bare {
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

		splitted := strings.Split(flag, ",")
		if len(splitted) != 2 {
			return fmt.Errorf("%s - %s", meta.name, ErrSizeAlignNelem)
		}

		f, ferr := strconv.ParseInt(splitted[0], 0, 0)
		s, serr := strconv.ParseInt(splitted[1], 0, 0)
		if ferr != nil || serr != nil {
			return fmt.Errorf("%s - %s", meta.name, ErrSizeAlignParsing)
		}

		meta.fun(int(f), int(s))
	}
	return nil
}

func printAggregateMeta(name string, meta AggregateMeta, opt, bare, verbose bool) {
	var (
		totPadding = 0
		typeName   = "Name"
	)

	for _, fLayout := range meta.Layout {
		totPadding += fLayout.padding
	}

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

	builder.WriteRune('\n')
	builder.WriteKeyword(name)
	builder.WriteBase(" {")
	builder.WriteRune('\n')

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

// RenderBuilder is a wrapper around `strings.Builder` which exposes methods
// for building strings with lipgloss styles applied upon them.
type RenderBuilder struct {
	strings.Builder
}

var (
	baseStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
	keywordStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#B29BC5"))
	commentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#747893"))
)

// WriteBase adds a string to the builder using the base style.
func (b *RenderBuilder) WriteBase(s string) {
	b.WriteString(baseStyle.Render(s))
}

// WriteKeyword adds a string to the builder using the keyword style.
func (b *RenderBuilder) WriteKeyword(s string) {
	b.WriteString(keywordStyle.Render(s))
}

// WriteComment adds a string to the builder using the comment style.
func (b *RenderBuilder) WriteComment(s string) {
	b.WriteString(commentStyle.Render(s))
}

// String returns the final string built with this builder using the box style.
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
