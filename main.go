package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"path"
	"slices"
	"strings"
	"unicode/utf8"
)

var (
	discreteOutput = flag.Bool("d", false, "每个输入文件对应一个输出文件")
	l3File         = flag.String("l3boot", "", "由程序为给定字符集分配三级简码并输出方案")
	shortFile      = flag.String("short", "", "简码码表，须包含一二级简码以及手动设置的三级简码")
)

func openOutput(inputFile string, ext string) *os.File {
	if !*discreteOutput {
		fmt.Println("#", inputFile)
		return os.Stdout
	}
	inputFile = path.Base(inputFile)
	ext0 := path.Ext(inputFile)
	if ext0 == ext {
		log.Fatalf("输入文件名%s和输出有同样的扩展名", inputFile)
	}
	outputFile := strings.TrimSuffix(inputFile, ext0) + ext
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatal(err)
	}
	return f
}

func processCharDefFile(filename string, lookup *LookupTable) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)

	out := bufio.NewWriter(openOutput(filename, ".mb"))
	defer out.Flush()

	count := 0
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		char, _ := utf8.DecodeRune(line)
		code, err := lookup.CharFullCode(char)
		if err != nil {
			return err
		}
		stem, err := lookup.CharBriefCode(char)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%c\t%s\t%s\n", char, code, stem)
		count++
	}
	log.Printf("输入文件 %s: %d字符", filename, count)
	return sc.Err()
}

func processL3File(l3File string, lookup *LookupTable) (err error) {
	chars := []rune{}
	err = ParseFile(l3File, func(fields []string) error {
		r := []rune(fields[0])
		if len(r) != 1 {
			return errors.New("简码输入文件应只有单字")
		}
		chars = append(chars, r[0])
		return nil
	})
	err = L3Bootstrap(os.Stdout, chars, lookup)
	if err != nil {
		return
	}

	out := bufio.NewWriter(openOutput(l3File, ".mb"))
	defer out.Flush()

	keys := slices.SortedFunc(maps.Keys(lookup.shortCodes), CodeCmp)
	for _, code := range keys {
		text := lookup.shortCodes[code]
		if !code.Valid() {
			log.Fatalf("代码%s无效（%s）", code, text)
		}
		fmt.Fprintf(out, "%s\t%s\n", text, code)
	}
	return
}

func main() {
	flag.Parse()

	lookup := NewLookupTable()

	if *shortFile != "" {
		if err := ParseShortCodeFile(*shortFile, lookup); err != nil {
			log.Fatal(err)
		}
	}

	charDefFiles := flag.Args()

	if len(charDefFiles) == 0 {
		flag.Usage()
		return
	}

	// 两趟算法，因为可能存在交叉引用
	for _, filename := range charDefFiles {
		if err := ParseCharDefFile(filename, lookup); err != nil {
			log.Fatal(err)
		}
	}
	for _, filename := range charDefFiles {
		if err := processCharDefFile(filename, lookup); err != nil {
			log.Fatal(err)
		}
	}

	if *l3File != "" {
		if err := processL3File(*l3File, lookup); err != nil {
			log.Fatal(err)
		}
	}
}
