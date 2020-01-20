package stroo

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"
	"unicode/utf8"
)

func contains(args ...string) bool {
	who := args[0]
	for i := 1; i < len(args); i++ {
		if args[i] == who {
			return true
		}
	}
	return false
}

func empty(src string) bool {
	if src == "" {
		return true
	}
	return false
}

func lowerInitial(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return ""
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToTitle(r)) + s[n:]
}

func templateGoStr(input string) string {
	if len(input) > 0 && input[len(input)-1] == '\n' {
		input = input[0 : len(input)-1]
	}
	if strings.Contains(input, "`") {
		lines := strings.Split(input, "\n")
		for idx, line := range lines {
			lines[idx] = strconv.Quote(line + "\n")
		}
		return strings.Join(lines, " + \n")
	}
	return "`" + input + "`"
}

func isNil(value interface{}) bool {
	if value == nil {
		return true
	}
	return false
}

func SortFields(fields Fields) bool {
	sort.Sort(fields)
	return true
}

type TypeWithRoot struct {
	T *TypeInfo // like "current" type
	D *Code     // required as "parent" in recursion templates
}

type Code struct {
	CodeConfig
	Imports      []string
	PackageInfo  *PackageInfo
	Main         TypeWithRoot
	keeper       map[string]interface{} // template authors keeps data in here, key-value, as they need
	tmpl         *template.Template     // reference to template, so we don't pass it as parameter
	templateName string                 // set by template, used in GenerateAndStore and ListStored
}

// getters for config - to be accessible from template
func (c *Code) SelectedType() string     { return c.CodeConfig.SelectedType }
func (c *Code) TestMode() bool           { return c.CodeConfig.TestMode }
func (c *Code) DebugPrint() bool         { return c.CodeConfig.DebugPrint }
func (c *Code) Serve() bool              { return c.CodeConfig.Serve }
func (c *Code) TemplateFile() string     { return c.CodeConfig.TemplateFile }
func (c *Code) OutputFile() string       { return c.CodeConfig.OutputFile }
func (c *Code) SelectedPeerType() string { return c.CodeConfig.SelectedPeerType }

// gets a struct declaration by it's name
func (c *Code) StructByKey(key string) *TypeInfo { return c.PackageInfo.Types.Extract(key) }
func (c *Code) Tmpl() *template.Template         { return c.tmpl }
func (c *Code) Keeper() map[string]interface{}   { return c.keeper }

// returns true if the key exist and will overwrite
func (c *Code) Store(key string, value interface{}) bool {
	_, has := c.keeper[key]
	c.keeper[key] = value
	return has
}

func (c *Code) Retrieve(key string) interface{} {
	value, _ := c.keeper[key]
	return value
}

func (c *Code) HasInStore(key string) bool {
	_, has := c.keeper[key]
	if c.CodeConfig.DebugPrint {
		log.Printf("Has in store %q = %t", key, has)
	}
	return has
}

func (c *Code) AddToImports(imp string) string {
	c.Imports = append(c.Imports, imp)
	return ""
}

func (c *Code) Declare(name string) bool {
	if c.Main.T == nil {
		log.Printf("error : main type is not set - impossible...")
	}
	if name == "" {
		log.Printf("error : cannot declare empty template name (e.g.`Stringer`)")
	}
	c.templateName = name
	c.keeper[name+c.Main.T.Name] = "" // set it to empty in case of self reference, so template will exit
	return true
}

func (c *Code) GenerateAndStore(kind string) bool {
	entity := c.templateName + kind
	if c.CodeConfig.DebugPrint {
		log.Printf("Processing %q %q ", c.templateName, kind)
	}
	// already has it
	if _, has := c.keeper[entity]; has {
		if c.CodeConfig.DebugPrint {
			log.Printf("%q already stored.", kind)
		}
		return false
	}
	var buf strings.Builder
	nt := c.PackageInfo.Types.Extract(kind)
	if nt == nil {
		if c.CodeConfig.DebugPrint {
			log.Printf("%q doesn't exist.", kind)
		}
		return false
	}

	err := c.tmpl.ExecuteTemplate(&buf, c.templateName, TypeWithRoot{D: c, T: nt})
	if err != nil {
		if c.CodeConfig.DebugPrint {
			log.Printf("generate and store error : %v", err)
		}
		return false
	}
	c.keeper[entity] = buf.String()
	if c.CodeConfig.DebugPrint {
		log.Printf("%q stored.", kind)
	}
	return true
}

func (c *Code) ListStored() []string {
	var result []string
	for key, value := range c.keeper {
		if strings.HasPrefix(key, c.templateName) {
			if r, ok := value.(string); ok {
				// len(0) is default template for main (
				if len(r) > 0 {
					result = append(result, r)
				}
			} else {
				// if it's not a string, we're ignoring it
				if c.CodeConfig.DebugPrint {
					log.Printf("%q has prefix %q but it's not a string and we're ignoring it", key, c.templateName)
				}
			}
		}
	}
	return result
}

func (c *Code) Header(flagValues string) string {
	return fmt.Sprintf("// Generated on %v by Stroo [https://github.com/badu/stroo]\n"+
		"// Do NOT bother with altering it by hand : use the tool\n"+
		"// Arguments at the time of generation:\n//\t%s", time.Now().Format("Mon Jan 2 15:04:05"), flagValues)
}
