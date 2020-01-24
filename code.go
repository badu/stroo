package stroo

import (
	"errors"
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

type Code struct {
	CodeConfig
	Imports      []string
	PackageInfo  *PackageInfo
	keeper       map[string]interface{} // template authors keeps data in here, key-value, as they need
	tmpl         *template.Template     // reference to template, so we don't pass it as parameter
	templateName string                 // set by template, used in RecurseGenerate and ListStored
	Main         *TypeInfo              // not nil if we're working with a preselected type
}

func New(info *PackageInfo, config CodeConfig, template *template.Template, selectedType string) (*Code, error) {
	result := &Code{
		PackageInfo: info,
		CodeConfig:  config,
	}
	// reset keeper
	result.ResetKeeper()
	// add imports (they get cleared by importer tool)
	for _, imprt := range info.Imports {
		result.AddToImports(imprt.Path)
	}
	// we were provided with a template
	if template != nil {
		result.tmpl = template
	}
	// we were provided with a preselected type
	if selectedType != "" {
		var err error
		result.Main, err = result.StructByKey(selectedType)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// getters for config - to be accessible from template
func (c *Code) SelectedType() string           { return c.CodeConfig.SelectedType }
func (c *Code) TestMode() bool                 { return c.CodeConfig.TestMode }
func (c *Code) DebugPrint() bool               { return c.CodeConfig.DebugPrint }
func (c *Code) Serve() bool                    { return c.CodeConfig.Serve }
func (c *Code) TemplateFile() string           { return c.CodeConfig.TemplateFile }
func (c *Code) OutputFile() string             { return c.CodeConfig.OutputFile }
func (c *Code) SelectedPeerType() string       { return c.CodeConfig.SelectedPeerType }
func (c *Code) Tmpl() *template.Template       { return c.tmpl } // TODO temporary - can't really say what's the usage
func (c *Code) Keeper() map[string]interface{} { return c.keeper }
func (c *Code) ResetKeeper()                   { c.keeper = make(map[string]interface{}) }

// gets a struct declaration by it's name
// also sets the reference to this, so it can be accessed as Root()
func (c *Code) StructByKey(key string) (*TypeInfo, error) {
	result := c.PackageInfo.Types.Extract(key)
	if result == nil {
		return nil, fmt.Errorf("error looking for %q into types", key)
	}
	// set access to root to type and it's fields
	result.root = c
	for _, field := range result.Fields {
		field.root = c
	}
	return result, nil
}

// returns true if the key exist and will overwrite
func (c *Code) Store(key string, value interface{}) bool {
	_, has := c.keeper[key]
	c.keeper[key] = value
	return has
}

// retrieves the entire "storage" at template dev disposal
func (c *Code) Retrieve(key string) (interface{}, error) {
	value, has := c.keeper[key]
	if !has {
		return nil, fmt.Errorf("error : attempt to retrieve %q - was not found", key)
	}
	return value, nil
}

func (c *Code) HasInStore(key string) bool {
	_, has := c.keeper[key]
	if c.CodeConfig.DebugPrint {
		log.Printf("Has in store %q = %t", key, has)
	}
	return has
}

func (c *Code) AddToImports(imp string) string {
	if imp == "" {
		// dummy fix : don't allow empty imports
		return ""
	}
	for _, imprt := range c.Imports {
		if imprt == imp {
			// dummy fix : already has it
			return ""
		}
	}
	c.Imports = append(c.Imports, imp)
	return ""
}

// this should be called to allow the generator to know which kind of methods we're generating
func (c *Code) Declare(name string) error {
	if name == "" {
		log.Printf("error : cannot declare empty template name (e.g.`Stringer`)")

		return errors.New("error : cannot declare empty template name (e.g.`Stringer`)")
	}
	if c.Main == nil {
		log.Printf("error : main operating type was not selected")
		return errors.New("error : main operating type was not selected - you cannot recurse")
	}
	c.templateName = name
	c.keeper[name+c.Main.Name] = "" // set it to empty in case of self reference, so template will exit
	return nil
}

// checker for recurse generated
func (c *Code) HasNotGenerated(kind string) (bool, error) {
	if c.templateName == "" {
		return false, errors.New("you haven't called Declare(methodName) to allow replacing existing generated code")
	}
	_, has := c.keeper[c.templateName+kind]
	return !has, nil
}

// uses the template name to apply the template recursively
// it's useful for replacing the code in existing generated files
func (c *Code) RecurseGenerate(kind string) error {
	if c.templateName == "" {
		return errors.New("you haven't called Declare(methodName) to allow replacing existing generated code")
	}
	entity := c.templateName + kind
	if c.CodeConfig.DebugPrint {
		log.Printf("Processing %q %q ", c.templateName, kind)
	}
	// already has it
	if _, has := c.keeper[entity]; has {
		if c.CodeConfig.DebugPrint {
			log.Printf("%q already stored.", kind)
		}
		return errors.New("`" + kind + "` already stored. you are not checking that yourself?")
	}
	var buf strings.Builder

	nt, err := c.StructByKey(kind)
	if nt == nil || err != nil {
		if c.CodeConfig.DebugPrint {
			log.Printf("probably %q doesn't exist : %v", kind, err)
		}
		return err
	}
	err = c.tmpl.ExecuteTemplate(&buf, c.templateName, nt)
	if err != nil {
		if c.CodeConfig.DebugPrint {
			log.Printf("generate and store error : %v", err)
		}
		log.Printf("generate and store error : %v", err) // TODO : remove it after finishing dev
		return err
	}
	c.keeper[entity] = buf.String()
	if c.CodeConfig.DebugPrint {
		log.Printf("%q stored.", kind)
	}
	return nil
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
