package main

// TODOs
// - show struct fields meta? Like for each substruct field where padding is
// located -> this could easily be obtained by internally doing
// ctx.ResolveMeta for the inner type
// - by specifying `-table`, show it, otherwise only orint size/align/pad
// - by specifying `-graph`, show the struct layout with padding blocks
// - by specifying `-optimize` search for the best layouts
// - add ptr size/align, word size/align, short size/align etc. flags
// - add specific known combinations of the above (e.g. avr => 16bit int, alugn 1)
// - add support for function pointers parsing
// - remove log.Fatal/panics

import (
	"flag"
	"fmt"
	"log"
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

	helpUsage    = "show the help message"
	versionUsage = "print the version for this build"
	fileUsage    = "pass a file containing the type definitions"
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

	var help bool
	var version bool
	var file string

	fs := flag.NewFlagSet("stropt", flag.ExitOnError)
	fs.BoolVar(&help, "help", false, helpUsage)
	fs.BoolVar(&version, "version", false, versionUsage)
	fs.StringVar(&file, "file", "", fileUsage)

	if err := fs.Parse(os.Args[1:]); err != nil {
		panic(err) // TODO print it nicely
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
			log.Fatalf("Failed to open file: %v", err)
		}
		stropt(file, fs.Arg(0), string(cont))
	case len(fs.Args()) == 2:
		stropt("", fs.Arg(0), fs.Arg(1))
	default:
		fmt.Println(nameMessage)
		os.Exit(1)
	}
}

func stropt(fname, aggName, cont string) {
	aggregates, err := ExtractAggregates(fname, cont)
	if err != nil {
		log.Fatal(err)
	}

	meta, err := aggregates.ResolveMeta(aggName)
	if err != nil {
		log.Fatal(err)
	}
	printAggregateMeta(aggName, meta)
}

func printAggregateMeta(name string, meta AggregateMeta) {
	var headerStyle = lipgloss.NewStyle().
		Bold(true).
		Width(15).
		Foreground(lipgloss.Color("#39e75f")).
		Align(lipgloss.Center)

	var structStyle = lipgloss.NewStyle().
		Bold(true).
		Width(15).
		Foreground(lipgloss.Color("#ff8080")).
		Align(lipgloss.Center)

	var rowStyle = lipgloss.NewStyle().
		Bold(true).
		Width(15).
		Foreground(lipgloss.Color("#aeaeae")).
		Align(lipgloss.Center)

	totPadding := 0
	for _, fLayout := range meta.Layout {
		totPadding += fLayout.padding
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
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
		Headers("Type", "Size", "Alignment", "Padding")

	size := strconv.Itoa(meta.Size)
	align := strconv.Itoa(meta.Alignment)
	pad := strconv.Itoa(totPadding)
	t.Row(name, size, align, pad)

	for _, fLayout := range meta.Layout {
		size := strconv.Itoa(fLayout.size)
		align := strconv.Itoa(fLayout.alignment)
		pad := strconv.Itoa(fLayout.padding)
		t.Row(fLayout.Name, size, align, pad)
	}

	fmt.Println(t)
}

func debugVersion() {
	// Define the path to your C file.
	fn := os.Args[2]

	// Open the C file.
	f, err := os.Open(fn)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}

	// Set up the parser configuration.
	config, err := cc.NewConfig(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		log.Fatalf("could not create a config for the parser: %v", err)
	}

	srcs := []cc.Source{
		{Name: "<predefined>", Value: config.Predefined},
		{Name: "<builtin>", Value: cc.Builtin},
		{Name: fn, Value: f},
	}

	ast, err := cc.Parse(config, srcs)
	if err != nil {
		log.Fatalln(err)
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
