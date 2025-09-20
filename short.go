package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"slices"
	"unicode/utf8"
)

func fullCodeHasPrefix(lookup *LookupTable, c rune, prefix Code) bool {
	fullCode, err := lookup.CharFullCode(c)
	if err != nil {
		panic(err)
	}
	prefixBytes := prefix.Bytes()
	return bytes.Equal(fullCode[:len(prefixBytes)], prefixBytes)
}

func knownShortChars(shortCodes map[Code]string) (m map[rune]Code) {
	m = make(map[rune]Code)
	for code, t := range shortCodes {
		if runes := []rune(t); len(runes) == 1 {
			m[runes[0]] = code
		}
	}
	return m
}

type l3Processor struct {
	out        *bufio.Writer
	lookup     *LookupTable
	codeMap    map[Code][]rune
	shortChars map[rune]Code
	candidates map[Code][]rune

	missedChars   []rune
	fullOnlyChars []rune

	shortCount int
}

func stringIsRune(s string, r rune) bool {
	r1, n := utf8.DecodeRuneInString(s)
	return n == len(s) && r1 == r
}

func (p *l3Processor) processCode(code Code) {
	k := code[2]
	coded, _ := p.codeMap[code]
	for _, c := range coded {
		p.out.WriteRune(c)
		p.out.WriteByte(k)
		if s, _ := p.lookup.shortCodes[code]; stringIsRune(s, c) {
			p.out.WriteByte('^')
			if c == '变' {
				log.Printf("%s %s", s, code)
			}
		}
		p.out.WriteByte(' ')
	}
	candidates, ok := p.candidates[code]
	if !ok {
		return
	}
	// 先过滤掉全码字和高频字
	candidates = slices.DeleteFunc(candidates, func(c rune) bool {
		if slices.Contains(coded, c) {
			return true
		}
		if _, ok := p.lookup.fastChars[c]; ok {
			return true
		}
		/*
			// 以及二级简码是前缀的字
			parts, _ := p.lookup.Find(c)
			if len(parts) <= 3 {
				code1, isShort := p.shortChars[c]
				return isShort && bytes.HasPrefix(code.Bytes(), code1.Bytes())
			}
		*/
		return false
	})
	omitted := len(coded) > 0
	if omitted || len(candidates) > 1 {
		// 仍然有重码则过滤掉简码字
		candidates = slices.DeleteFunc(candidates, func(c rune) bool {
			_, isShort := p.shortChars[c]
			return isShort
		})
	}
	/*
		// 仍然有重码，则简码不是全码前三位的优先
		if len(candidates) > 1 {
			lowPriorities := make([]rune, 0, len(candidates))
			n := 0
			for _, c := range candidates {
				if fullCodeHasPrefix(p.lookup, c, code) {
					lowPriorities = append(lowPriorities, c)
				} else {
					candidates[n] = c
					n++
				}
			}
			candidates = append(candidates[:n], lowPriorities...)
		}
	*/
	for _, c := range candidates {
		var left, right string
		if !omitted {
			right = "*"
			p.lookup.AddShortCode(code.String(), string(c))
			p.shortCount++
			omitted = true
		} else if fullCodeHasPrefix(p.lookup, c, code) {
			left, right = "[", "]"
			p.fullOnlyChars = append(p.fullOnlyChars, c)
		} else {
			left, right = "(", ")"
			p.missedChars = append(p.missedChars, c)
		}
		p.out.WriteString(left)
		p.out.WriteRune(c)
		p.out.WriteByte(k)
		p.out.WriteString(right)
	}
}

func (p *l3Processor) processGroup(code Code) {
	i, j := code[0], code[1]
	coded, _ := p.lookup.shortCodes[code]
	fmt.Fprintf(p.out, "%c%c\t%s\t", i, j, coded)
	for code[2] = 'A'; code[2] <= 'Z'; code[2]++ {
		p.processCode(code)
	}
}

func L3Bootstrap(w io.Writer, chars []rune, lookup *LookupTable) error {
	p := &l3Processor{
		out:        bufio.NewWriter(w),
		lookup:     lookup,
		codeMap:    map[Code][]rune{},
		shortChars: knownShortChars(lookup.shortCodes),
		candidates: map[Code][]rune{},
	}
	defer p.out.Flush()
	for c := range lookup.parts {
		fcode, err := lookup.CharFullCode(c)
		if err != nil {
			return err
		}
		mapAppendUnique(p.codeMap, fcode, c)
	}
	log.Printf("三级简码自举：%d个字符", len(chars))
	for _, c := range chars {
		if sc, ok := p.shortChars[c]; ok && len(sc.Bytes()) == 3 {
			// 已有（特殊）简码则不考虑缺省编码
			continue
		}
		code, err := p.lookup.DefaultL3Code(c)
		if err != nil {
			return err
		}
		if code.Valid() {
			mapAppendUnique(p.candidates, code, c)
		}
	}
	for code, s := range lookup.shortCodes {
		c, n := utf8.DecodeRuneInString(s)
		if n == len(s) {
			mapAppendUnique(p.codeMap, code, c)
		}
	}
	for i := byte('A'); i <= 'Z'; i++ {
		for j := byte('A'); j <= 'Z'; j++ {
			p.processGroup(Code{i, j})
			fmt.Fprintln(p.out)
		}
	}
	log.Printf("生成简码%d个\n", p.shortCount)
	fmt.Fprintf(p.out, "无法按三级简码输入的字(%d)： %c\n", len(p.missedChars), p.missedChars)
	fmt.Fprintf(p.out, "三级简码是全码前缀的字(%d)： %c\n", len(p.fullOnlyChars), p.fullOnlyChars)

	for char := range p.lookup.parts {
		code, err := p.lookup.CharFullCode(char)
		if err != nil {
			panic(err)
		}
		codeBytes := code.Bytes()
		if len(codeBytes) < 3 {
			continue
		}
		if shortCode, ok := p.shortChars[char]; ok && bytes.Equal(codeBytes, shortCode.Bytes()) {
			continue
		}
		candidates := p.codeMap[code]
		if len(candidates) < 2 {
			continue
		}
		for _, c := range candidates {
			if c == char {
				continue
			}
			shortCode, ok := p.shortChars[c]
			if !ok {
				continue
			}
			fullCode, _ := p.lookup.CharFullCode(c)
			if sb, fb := shortCode.Bytes(), fullCode.Bytes(); len(sb) < len(fb) && bytes.HasPrefix(fb, sb) {
				fmt.Fprintf(p.out, "%s %c\t%s %c\n", sb, c, fb, char)
			}
		}
	}
	/* for code, chars := range p.codeMap {
		if len(code.Bytes()) != maxCodeLen || len(chars) < 2 {
			continue
		}
		shortCode, ok := p.shortChars[chars[0]]
		if ok && bytes.HasPrefix(code.Bytes(), shortCode.Bytes()) {
			star := ""
			if _, ok := p.shortChars[chars[1]]; !ok {
				star = "*"
			}
			fmt.Fprintf(p.out, "%s %c\t%s %c%s\n", shortCode, chars[0], code, chars[1:], star)
		}
	} */
	return nil
}
