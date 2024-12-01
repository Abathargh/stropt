package main

// TODOs
// - by specifying `-graph`, show the struct layout with padding blocks
// - add ptr size/align, word size/align, short size/align etc. flags
// - add specific known combinations of the above (e.g. avr => 16bit int, alugn 1)
// - add support for function pointers parsing
// - test as wasm app
// --all shoudld show tables (before after opt) padding blocks, structs
// reordering printed (like inpurple in the example)

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"modernc.org/cc/v4"
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
	optimizeUsage = "suggests an optimized layout and shows related statistics"
	fileUsage     = "pass a file containing the type definitions"
)

var Version = ""

func main() {
	if len(os.Args) > 1 && os.Args[1] == "debug" {
		debugVersion()
		return
	}

	if Version == "" {
		info, ok := debug.ReadBuildInfo()
		if ok {
			Version = info.Main.Version
		}
	}

	var (
		help     bool
		bare     bool
		version  bool
		verbose  bool
		optimize bool
		file     string
	)

	fs := flag.NewFlagSet("stropt", flag.ExitOnError)
	fs.BoolVar(&help, "help", false, helpUsage)
	fs.BoolVar(&bare, "bare", false, bareUsage)
	fs.BoolVar(&version, "version", false, versionUsage)
	fs.BoolVar(&verbose, "verbose", false, verboseUsage)
	fs.BoolVar(&optimize, "optimize", false, optimizeUsage)
	fs.StringVar(&file, "file", "", fileUsage)

	if err := fs.Parse(os.Args[1:]); err != nil {
		logErrorMessage("could not parse args: %s", err)
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
		stropt(file, fs.Arg(0), string(cont), bare, verbose, optimize)
	case len(fs.Args()) == 2:
		stropt("", fs.Arg(0), fs.Arg(1), bare, verbose, optimize)
	default:
		logErrorMessage(nameMessage)
	}
}

func stropt(fname, aggName, cont string, bare, verbose, optimize bool) {
	aggregates, err := ExtractAggregates(fname, cont)
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
		fmt.Println(titleBox.Render(aggName))

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

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Width(15).
			Foreground(lipgloss.Color("#ececec")).
			Align(lipgloss.Center)

	structStyle = lipgloss.NewStyle().
			Bold(true).
			Width(15).
			Foreground(lipgloss.Color("#aeaeae")).
			Align(lipgloss.Center)

	rowStyle = lipgloss.NewStyle().
			Bold(false).
			Width(15).
			Foreground(lipgloss.Color("#aeaeae")).
			Align(lipgloss.Center)

	titleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Width(63).
			Foreground(lipgloss.Color("#AEAEAE")).
			Align(lipgloss.Center)
)

func printAggregateMeta(name string, meta AggregateMeta, opt, bare, verbose bool) {
	totPadding := 0
	for _, fLayout := range meta.Layout {
		totPadding += fLayout.padding
	}

	typeName := "Type"
	if opt {
		typeName = "Type (opt)"
	}

	t := table.New().
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

	doPrint(name, meta.Size, meta.Alignment, totPadding, t, bare)

	for _, fLayout := range meta.Layout {
		size := strconv.Itoa(fLayout.size)
		align := strconv.Itoa(fLayout.alignment)
		pad := strconv.Itoa(fLayout.padding)
		t.Row(fLayout.Name, size, align, pad)

		if fLayout.subAggregate != nil && verbose {
			for _, sub := range fLayout.subAggregate {
				name := fmt.Sprintf("%s::%s", fLayout.Type, sub.Name)
				doPrint(name, sub.size, sub.alignment, sub.padding, t, bare)
				t.Row(name, size, align, pad)
			}
		}
	}

	if !bare {
		fmt.Println(t)
	}
}

// TODO func generateBlocks()

func doPrint(name string, size, align, pad int, tab *table.Table, bare bool) {
	if bare {
		fmt.Fprintf(
			os.Stdout, "%s, size: %d, alignment: %d, padding: %d\n",
			name, size, align, pad,
		)
		return
	}

	sizeStr := strconv.Itoa(size)
	alignStr := strconv.Itoa(align)
	padStr := strconv.Itoa(pad)
	tab.Row(name, sizeStr, alignStr, padStr)
}

func debugVersion() {
	// Define the path to your C file.
	fn := os.Args[2]

	// Open the C file.
	f, err := os.Open(fn)
	if err != nil {
		logErrorMessage("Failed to open file: %v", err)
	}

	// Set up the parser configuration.
	config, err := cc.NewConfig(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		logErrorMessage("could not create a config for the parser: %v", err)
	}

	srcs := []cc.Source{
		{Name: "<predefined>", Value: config.Predefined},
		{Name: "<builtin>", Value: cc.Builtin},
		{Name: fn, Value: f},
	}

	ast, err := cc.Parse(config, srcs)
	if err != nil {
		logError(err)
	}

	// Access the AST of the parsed translation unit.
	fmt.Println("Parsed AST:")

	for name, node := range ast.Scope.Nodes {
		if strings.HasPrefix(name, "__") {
			continue
		}
		fmt.Printf("%s: %v\n", name, node)
	}
}

var (
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Width(30).
			Margin(0, 1, 1, 0).
			Padding(1, 1, 1, 2).
			Align(lipgloss.Left)

	baseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA"))

	keywordStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B29BC5"))

	commentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#747893"))
)

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
		builder.WriteBase("\t")
		builder.WriteKeyword(field.Type)
		builder.WriteBase(" ")
		builder.WriteBase(field.Name)
		builder.WriteBase(";")
		builder.WriteNewline()
	}
	builder.WriteBase("};")

	return builder.String()
}

type RenderBuilder struct {
	strings.Builder
}

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
