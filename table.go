package main

import (
	"fmt"
	"log"
	"unicode/utf8"
)

const maxCodeLen = 4

type Code [maxCodeLen]byte

func MakeCode(s string) (c Code) {
	copy(c[:], s)
	return
}

func (c Code) Bytes() []byte {
	l := 0
	for l = 0; l < len(c); l++ {
		if c[l] == 0 {
			break
		}
	}
	return c[:l]
}

func (c Code) Valid() bool {
	var l int
	for l = 0; l < len(c) && c[l] != 0; l++ {
		if !('A' <= c[l] && c[l] <= 'Z') {
			return false
		}
	}
	return l > 0
}

func (c Code) String() string {
	return string(c.Bytes())
}

func CodeCmp(a, b Code) int {
	for i := 0; i < maxCodeLen; i++ {
		switch {
		case a[i] < b[i]:
			return -1
		case a[i] > b[i]:
			return 1
		}
	}
	return 0
}

type Part struct {
	Code
}

func MakePart(s string) (p Part) {
	p.Code = MakeCode(s)
	return
}

func (p Part) IsCode() bool {
	return p.Code.Valid()
}

func (p Part) GetChar() rune {
	r, _ := utf8.DecodeRune(p.Code[:])
	return r
}

type Tag byte

const (
	tagMainPart Tag = 1 + iota
	tagSubPart
)

type LookupTable struct {
	parts      map[rune][]Part
	altParts   map[rune][][]Part
	flattened  map[rune][]Code
	shortCodes map[Code]string // code to text
	fastChars  map[rune]byte   // rune to code
	tags       map[rune]Tag
}

func NewLookupTable() *LookupTable {
	return &LookupTable{
		parts:      make(map[rune][]Part),
		flattened:  make(map[rune][]Code),
		altParts:   make(map[rune][][]Part),
		shortCodes: make(map[Code]string),
		fastChars:  make(map[rune]byte),
		tags:       make(map[rune]Tag),
	}
}

func (t *LookupTable) AddCharDef(c rune, parts []Part, tag Tag) {
	if _, ok := t.parts[c]; ok {
		log.Printf("%c 已有定义", c)
		t.altParts[c] = append(t.altParts[c], parts)
	} else {
		t.parts[c] = parts
		t.flattened[c] = nil
		if tag != 0 {
			t.tags[c] = tag
		}
	}
}

func (t *LookupTable) AddShortCode(code, text string) {
	t.shortCodes[MakeCode(code)] = text
	if len(code) == 1 {
		if r, n := utf8.DecodeRuneInString(text); n == len(text) {
			t.fastChars[r] = code[0]
		}
	}
}

func (t *LookupTable) Find(c rune) (parts []Part, ok bool) {
	parts, ok = t.parts[c]
	return
}

func (t *LookupTable) ExpandChar(c rune) ([]Code, error) {
	if parts := t.flattened[c]; parts != nil {
		return parts, nil
	}
	parts, ok := t.parts[c]
	if !ok {
		return nil, fmt.Errorf("找不到“%c”的定义", c)
	}
	flattened, err := t.ExpandParts(parts)
	if err != nil {
		return nil, err
	}
	t.flattened[c] = flattened
	return flattened, nil
}

func (t *LookupTable) ExpandParts(parts []Part) (ret []Code, err error) {
	ret = make([]Code, 0, len(parts))
	for _, p := range parts {
		if p.IsCode() {
			ret = append(ret, p.Code)
		} else {
			var pp []Code
			pp, err = t.ExpandChar(p.GetChar())
			if err != nil {
				return
			}
			ret = append(ret, pp...)
		}
	}
	return
}

func (t *LookupTable) CharFullCode(c rune) (code Code, err error) {
	parts, err := t.ExpandChar(c)
	if err != nil {
		return
	}
	cs := code[:0:len(code)]
	cs = append(cs, parts[0].Bytes()...)
	n := len(cs)
	switch l := len(parts); l {
	case 1:
		switch n {
		case 1:
			if _, ok := t.fastChars[c]; ok {
				break
			}
			// 不是高频字的，加A/AA/AAA
			cs = append(cs, 'A')
			if s, ok := t.shortCodes[code]; ok && s != string(c) {
				cs = append(cs, 'A')
				if tag, ok := t.tags[c]; ok && tag == tagSubPart {
					cs = append(cs, 'A')
				}
			}
		case 2:
			if s, ok := t.shortCodes[code]; ok && s != string(c) {
				tag, _ := t.tags[c]
				switch tag {
				case tagSubPart:
					cs = append(cs, "AA"...)
				default:
					cs = append(cs, 'A')
				}
			}
		case 3:
			switch tag, _ := t.tags[c]; tag {
			case tagSubPart:
				cs = append(cs, 'A')
			}
		}
	case 2:
		cs = appendCapped(cs, parts[1].Bytes())
		if len(cs) == 2 {
			cs = append(cs, "VV"...)
		}
	case 3:
		if n < 3 {
			cs = append(cs, parts[1][0])
		}
		cs = appendCapped(cs, parts[2].Bytes())
	default:
		switch n {
		case 1:
			cs = append(cs, parts[1][0])
			fallthrough
		case 2:
			cs = append(cs, parts[l-2][0])
			fallthrough
		default:
			cs = appendCapped(cs, parts[l-1].Bytes())
		}
	}
	return
}

func (t *LookupTable) CharBriefCode(c rune) (code [2]byte, err error) {
	if fastCode, ok := t.fastChars[c]; ok {
		code[0] = fastCode
		code[1] = 'V'
	} else {
		var parts []Code
		parts, err = t.ExpandChar(c)
		if err != nil {
			return
		}
		c0 := parts[0].Bytes()
		code[0] = c0[0]
		if len(parts) == 1 {
			if len(c0) == 1 {
				code[1] = 'A'
			} else {
				code[1] = c0[1]
			}
		} else {
			code[1] = parts[1][0]
		}
	}
	return
}

/*
func (t *LookupTable) PhraseCode(phr []rune) (code Code, err error) {
	var steps []int
	switch l := len(phr); l {
	case 0, 1:
		err = errors.New("词组至少2个字符")
		return
	case 2:
		steps = []int{2, 2}
	case 3:
		steps = []int{1, 2, 1}
	default:
		steps = []int{1, 1, 1, 1}
	}

	cs := code[:0:len(code)]
	for i, c := range phr {
		var bcode [2]byte
		bcode, err = t.CharBriefCode(c)
		if err != nil {
			return
		}
		if n := len(steps); i < n {
			cs = append(cs, bcode[:steps[i]]...)
		}
	}
	return
}

func (t *LookupTable) Encode(s string) (code Code, err error) {
	runes := []rune(s)
	if len(runes) == 1 {
		code, err = t.CharFullCode(runes[0])
	} else {
		code, err = t.PhraseCode(runes)
	}
	return
}
*/

func (t *LookupTable) DefaultL3Code(c rune) (code Code, err error) {
	parts, err := t.ExpandChar(c)
	if err != nil {
		return
	}
	switch len(parts) {
	case 1:
		c0 := parts[0].Bytes()
		if len(c0) < 3 {
			return
		}
		copy(code[:], c0[:3])
	case 2:
		code[0] = parts[0][0]
		code[1] = parts[1][0]
		if b := parts[1][1]; b != 0 {
			code[2] = b
		} else if parts[0][1] == 0 {
			code[2] = 'V'
		} else {
			code = Code{}
		}
	default:
		code[0] = parts[0][0]
		code[1] = parts[1][0]
		code[2] = parts[2][0]
	}
	return
}
