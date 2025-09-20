package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func newCsvReader(f io.Reader) *csv.Reader {
	rd := csv.NewReader(f)
	rd.Comma = '\t'
	rd.Comment = '#'
	rd.FieldsPerRecord = -1
	rd.ReuseRecord = true
	return rd
}

func ParseFile(filename string, callback func([]string) error) (err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()
	rd := newCsvReader(f)
	for {
		var fields []string
		fields, err = rd.Read()
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return
		}
		err = callback(fields)
		if err != nil {
			line, _ := rd.FieldPos(0)
			err = fmt.Errorf("%s:%d: %w", filename, line, err)
			return
		}
	}
	return

}

func ParseShortCodeFile(filename string, lookup *LookupTable) error {
	return ParseFile(filename, func(fields []string) error {
		if len(fields) < 2 {
			return errors.New("至少应有2个域")
		}
		text, code := fields[0], fields[1]
		if len(code) < 1 || len(code) >= 4 {
			return fmt.Errorf("无效代码（%s）；简码应为1-3字节", code)
		}
		lookup.AddShortCode(code, text)
		return nil
	})
}

func ParseCharDefFile(filename string, lookup *LookupTable) error {
	return ParseFile(filename, func(fields []string) error {
		if len(fields) < 2 {
			return errors.New("至少应有2个域")
		}
		ch := []rune(fields[0])
		if len(ch) != 1 {
			return errors.New("应当只有1个字")
		}
		parts := strings.Split(fields[1], " ")
		if len(parts) < 1 {
			return errors.New("至少应有1个部件")
		}
		convertedParts := make([]Part, len(parts))
		for i, part := range parts {
			if len(part) < 1 || len(part) > 4 {
				return errors.New("部件字节数应为1-4")
			}
			convertedParts[i] = MakePart(part)
		}
		var tag Tag
		if len(fields) >= 3 {
			tagField := fields[2]
			if strings.HasPrefix(tagField, "AAA") {
				tag = tagSubPart
			} else if strings.HasPrefix(tagField, "A") {
				tag = tagMainPart
			}
		}
		lookup.AddCharDef(ch[0], convertedParts, tag)
		return nil
	})
}
