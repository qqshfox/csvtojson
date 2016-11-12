package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	flag "github.com/spf13/pflag"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var VERSION = "UNKNOWN"

const (
	csvExt  = ".csv"
	jsonExt = ".json"
)

type options struct {
	verbose   bool
	debug     bool
	delimiter rune
	comment   rune
	header    string
	indent    string
}

func debugLog(v ...interface{}) {
	if opts.debug {
		fmt.Print(v...)
	}
}

func debugLogf(format string, v ...interface{}) (n int, err error) {
	if opts.debug {
		return fmt.Printf(format, v...)
	}
	return 0, nil
}

func debugLogln(v ...interface{}) (n int, err error) {
	if opts.debug {
		return fmt.Println(v...)
	}
	return 0, nil
}

func verboseLog(v ...interface{}) (n int, err error) {
	if opts.verbose {
		return fmt.Print(v...)
	}
	return 0, nil
}

func verboseLogf(format string, v ...interface{}) (n int, err error) {
	if opts.verbose {
		return fmt.Printf(format, v...)
	}
	return 0, nil
}

func verboseLogln(v ...interface{}) (n int, err error) {
	if opts.verbose {
		return fmt.Println(v...)
	}
	return 0, nil
}

func readCsv(r io.Reader, opts options) ([][]string, []string, error) {
	rr := csv.NewReader(r)
	rr.Comma = opts.delimiter
	rr.Comment = opts.comment

	rows, err := rr.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if opts.debug {
		for i, row := range rows {
			debugLogf("%d: %v\n", i, row)
		}
	}

	var headers []string
	var records [][]string
	if opts.header == "" {
		headers = rows[0]
		records = rows[1:]
	} else {
		headers = nil
		records = rows
	}

	return records, headers, nil
}

func recordsToMaps(records [][]string, headers []string) []map[string]string {
	maps := []map[string]string{}

	for _, record := range records {
		m := make(map[string]string)
		for i, value := range record {
			m[headers[i]] = value
		}
		maps = append(maps, m)
	}

	return maps
}

func csvToJson(r io.Reader, opts options) ([]byte, error) {
	records, headers, err := readCsv(r, opts)
	if err != nil {
		return nil, err
	}
	if opts.header != "" {
		tmp, _, err := readCsv(strings.NewReader(opts.header), opts)
		if err != nil {
			return nil, err
		}
		headers = tmp[0]
	}

	debugLogf("records: %v, headers: %v\n", records, headers)
	maps := recordsToMaps(records, headers)

	var b []byte
	if opts.indent == "" {
		b, err = json.Marshal(maps)
	} else {
		b, err = json.MarshalIndent(maps, "", opts.indent)
	}
	if err != nil {
		return nil, err
	}

	return b, nil
}

func csvFileToJsonFile(input, output string, opts options) error {
	inputFile, err := os.Open(input)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	b, err := csvToJson(inputFile, opts)
	if err != nil {
		return err
	}

	dir := filepath.Dir(output)
	debugLogf("mkdir -p %q\n", dir)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	outputFile, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	outputFile.Write(b)

	return nil
}

func inputPathToOutputPath(path string) string {
	dirname, filename := filepath.Split(path)

	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]

	outputFilename := strings.Join([]string{base, jsonExt}, "")

	return filepath.Join(dirname, outputFilename)
}

var versionVar bool
var inputDir string
var outputDir string
var delimiter string
var comment string

var opts options

func init() {
	flag.BoolVarP(&versionVar, "version", "v", false, "print the version")
	flag.StringVarP(&inputDir, "input", "i", "", "the input dir of CSV files, e.g. \".\"")
	flag.StringVarP(&outputDir, "output", "o", "", "the output dir of JSON files, e.g. \".\"")
	flag.BoolVarP(&opts.verbose, "verbose", "V", false, "enable verbose mode")
	flag.BoolVarP(&opts.debug, "debug", "D", false, "enable debug mode")
	flag.StringVarP(&delimiter, "delimiter", "d", ",", "the CSV delimiter character")
	flag.StringVarP(&comment, "comment", "c", "", "the CSV comment character")
	flag.StringVarP(&opts.header, "header", "H", "", "use this as the CSV header instead of the first line of CSV file")
	flag.StringVarP(&opts.indent, "indent", "t", "", "the JSON indent")
}

func main() {
	flag.Usage = func() {

		fmt.Fprintf(os.Stderr, "csvtojson is a tool to convert CSV files into JSON files\n\nUsage:\n  %s [flags]\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if versionVar {
		fmt.Println(VERSION)
		return
	}

	opts.verbose = opts.debug || opts.verbose
	if inputDir == "" {
		fmt.Fprintf(os.Stderr, "Error: the input directory can't be empty\n\n")
		flag.Usage()
		os.Exit(-1)
	}
	if outputDir == "" {
		fmt.Fprintf(os.Stderr, "Error: the output directory can't be empty\n\n")
		flag.Usage()
		os.Exit(-1)
	}
	if len(delimiter) != 1 {
		fmt.Fprintf(os.Stderr, "Error: the delimiter should be exact one characater\n\n")
		flag.Usage()
		os.Exit(-1)
	}
	if len(comment) > 1 {
		fmt.Fprintf(os.Stderr, "Error: the comment should be exact one characater\n\n")
		flag.Usage()
		os.Exit(-1)
	}

	opts.delimiter = ([]rune(delimiter))[0]
	if len(comment) != 0 {
		opts.comment = ([]rune(comment))[0]
	}

	debugLogln(opts)

	paths := []string{}
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(strings.ToLower(path)) == csvExt {
			path, _ = filepath.Rel(inputDir, path)
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
		flag.Usage()
		os.Exit(-1)
	}

	for _, path := range paths {
		inputPath := filepath.Join(inputDir, path)
		outputPath := filepath.Join(outputDir, inputPathToOutputPath(path))

		verboseLogf("Convert %q to %q\n", inputPath, outputPath)

		err := csvFileToJsonFile(inputPath, outputPath, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to convert %q: %v, ignored\n", inputPath, err)
		}
	}
}
