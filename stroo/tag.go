package stroo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Tags []*Tag

type Tag struct {
	Key     string   // i.e: `json:"fieldName,omitempty". Here key is: "json"
	Name    string   // i.e: `json:"fieldName,omitempty". Here name is: "fieldName"
	Options []string // `json:"fieldName,omitempty". Here options is: ["omitempty"]
}

var (
	errTagSyntax      = errors.New("bad syntax for struct tag pair")
	errTagKeySyntax   = errors.New("bad syntax for struct tag key")
	errTagValueSyntax = errors.New("bad syntax for struct tag value")
	errTagNotExist    = errors.New("tag does not exist")
)

func ParseTags(tag string) (*Tags, error) {
	if tag == "" {
		return nil, nil
	}

	if tag[0] == '`' && tag[len(tag)-1] == '`' {
		tag = tag[1 : len(tag)-1]
	}

	var tags Tags
	i := 0
	for i < len(tag) && tag[i] == ' ' {
		i++
	}
	tag = tag[i:]
	if tag == "" {
		return nil, nil
	}

	i = 0
	for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
		i++
	}

	if i == 0 {
		return nil, errTagKeySyntax
	}
	if i+1 >= len(tag) || tag[i] != ':' {
		return nil, errTagSyntax
	}
	if tag[i+1] != '"' {
		return nil, errTagValueSyntax
	}

	key := string(tag[:i])
	tag = tag[i+1:]

	i = 1
	for i < len(tag) && tag[i] != '"' {
		if tag[i] == '\\' {
			i++
		}
		i++
	}
	if i >= len(tag) {
		return nil, errTagValueSyntax
	}

	qValue := string(tag[:i+1])
	tag = tag[i+1:]

	value, err := strconv.Unquote(qValue)
	if err != nil {
		return nil, errTagValueSyntax
	}

	res := strings.Split(value, ",")
	name := res[0]
	options := res[1:]
	if len(options) == 0 {
		options = nil
	}

	tags = append(tags, &Tag{
		Key:     key,
		Name:    name,
		Options: options,
	})

	return &tags, nil
}

func (t *Tag) Value() string {
	options := strings.Join(t.Options, ",")
	if options != "" {
		return fmt.Sprintf(`%s,%s`, t.Name, options)
	}
	return t.Name
}

// cannot implement Stringer due to tests
func (t Tag) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
		tno++
	}
	sb.WriteString(tabs + "&Tag{")
	sb.WriteString("Key:\"" + t.Key + "\",")
	sb.WriteString("Name:\"" + t.Name + "\"")
	opts := ""
	for idx, option := range t.Options {
		if idx > 0 {
			opts += ","
		}
		opts += "\"" + option + "\""
	}
	if len(opts) > 0 {
		sb.WriteString(",Options:[]string{" + opts + "}")
	}
	sb.WriteString("},\n")
}

// cannot implement Stringer due to tests
func (t Tags) Debug(sb *strings.Builder, args ...int) {
	var tabs string
	var tno int
	if len(args) > 0 {
		tno = args[0]
		tabs = strings.Repeat("\t", tno)
		tno++
	}
	if len(t) > 0 {
		sb.WriteString(tabs + "Tags:&Tags{\n")
	}
	for _, tag := range t {
		if tno > 0 {
			tag.Debug(sb, tno)
		} else {
			tag.Debug(sb)
		}
	}
	if len(t) > 0 {
		sb.WriteString(tabs + "},\n")
	}
}

func (t Tags) Get(key string) (*Tag, error) {
	for _, tag := range t {
		if tag.Key == key {
			return tag, nil
		}
	}

	return nil, errTagNotExist
}
