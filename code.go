package stroo

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"text/template"
	"time"
)

type Code struct {
	CodeConfig
	Imports     []string
	PackageInfo *PackageInfo
	keeper      map[string]interface{} // template authors keeps data in here, key-value, as they need
	tmpl        *template.Template     // reference to template, so we don't pass it as parameter
}

var Root *Code

func New(
	info *PackageInfo,
	config CodeConfig,
	tmpl *template.Template,
) (*Code, error) {
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
	if tmpl != nil {
		result.tmpl = tmpl
	}
	Root = result // set global, there is no other way to make template aware of the "common"
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
func (c *Code) Tmpl() *template.Template       { return c.tmpl } // can't really say what's the usage, but we're open
func (c *Code) Keeper() map[string]interface{} { return c.keeper }
func (c *Code) ResetKeeper()                   { c.keeper = make(map[string]interface{}) }
func (c *Code) PackageName() string            { return c.PackageInfo.Name }

// gets a struct declaration by it's name
func (c *Code) StructByKey(key string) (*TypeInfo, error) {
	result := c.PackageInfo.Types.Extract(key)
	if result == nil {
		log.Printf("error looking for %q into types", key)
		return nil, fmt.Errorf("error looking for %q into types", key)
	}
	return result, nil
}

// returns true if the key exist and will overwrite
func (c *Code) Store(key string, value interface{}) error {
	_, has := c.keeper[key]
	if has {
		log.Printf("%q was overwritten in store", key)
	}
	c.keeper[key] = value
	return nil
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

// check if a kind has a method called the same as the template being declared
func (c *Code) Implements(fieldInfo TypeInfo) (bool, error) {
	if c.CodeConfig.TemplateName == "" {
		return false, errors.New("you haven't called Declare(methodName) to allow replacing existing generated code")
	}
	if fieldInfo.Package == "" {
		nt, err := c.StructByKey(fieldInfo.Kind)
		if nt == nil || err != nil {
			log.Printf("lookup error : %v", err)
			return false, err
		}
	} else {
		if fieldInfo.PackagePath != "" {
			log.Printf("package path : %q", fieldInfo.PackagePath)
		}
		log.Printf("we should load : %q and lookup for %q", fieldInfo.Package, fieldInfo.Kind)
		// we have to load the package
		loadedPackage, err := LoadPackage(fieldInfo.Package)
		if err != nil {
			log.Printf("load error : %v", err)
			return false, err
		}
		codeBuilder := DefaultAnalyzer()
		command := NewCommand(codeBuilder)
		if err := command.Analyse(codeBuilder, loadedPackage); err != nil {
			return false, fmt.Errorf("error analyzing package : %v", err)
		}
		//log.Printf("package loaded :\n %#v\n", command.Result)
	}
	// by default : assume it doens't implements
	return false, nil
}

// this should be called to allow the generator to know which kind of methods we're generating
func (c *Code) Declare(name string) error {
	if name == "" {
		//log.Printf("error : cannot declare empty template name (e.g.`String` for Stringer interface implementation)")
		return errors.New("error : cannot declare empty template name (e.g.`String` for Stringer interface implementation)")
	}
	if c.CodeConfig.SelectedType == "" {
		return errors.New("error : selected type is empty")
	}
	c.CodeConfig.TemplateName = name
	c.keeper[name+c.CodeConfig.SelectedType] = "" // set it to empty in case of self reference, so template will exit
	return nil
}

// checker for recurse generated
func (c *Code) HasNotGenerated(pkg, kind string) (bool, error) {
	if c == nil {
		return false, errors.New("impossible : code was not passed here (.Root missing ???)")
	}
	if c.CodeConfig.TemplateName == "" {
		//log.Printf("you haven't called Declare(methodName) to allow replacing existing generated code")
		return false, errors.New("you haven't called Declare(methodName) to allow replacing existing generated code")
	}
	if pkg == "" {
		//log.Printf("HasNotGenerated : empty package on kind %q", kind)
		return false, nil
	}
	if pkg != c.PackageInfo.Name {
		//log.Printf("HasNotGenerated : different packages %q != %q", pkg, c.PackageInfo.Name)
		return false, nil
	}
	// check if we're going to allow call to RecurseGenerate (optim calls)
	nt, err := c.StructByKey(kind)
	if nt == nil || err != nil {
		return false, err
	}
	if nt.IsImported {
		//log.Printf("HasNotGenerated : %q in package %q it's maked imported", kind, pkg)
		return false, nil
	}
	if nt.Package != pkg {
		//log.Printf("HasNotGenerated : %q in package %q it's a different package %q", kind, pkg, nt.Package)
		return false, nil
	}
	if IsBasic(nt.Kind) {
		//log.Printf("HasNotGenerated : call for basic kind %q", nt.Kind)
		return false, nil
	}
	// finally we deliver result
	entity := c.CodeConfig.TemplateName + kind
	_, has := c.keeper[entity]
	//log.Printf("HasNotGenerated : %q %t", entity, has)
	return !has, nil
}

// uses the template name to apply the template recursively
// it's useful for replacing the code in existing generated files
func (c *Code) RecurseGenerate(pkg, kind string) error {
	if c.CodeConfig.TemplateName == "" {
		return errors.New("you haven't called Declare(methodName) to allow replacing existing generated code")
	}
	if pkg == "" {
		log.Printf("RecurseGenerate : empty package on kind %q", kind)
		return nil
	}
	if pkg != c.PackageInfo.Name {
		log.Printf("RecurseGenerate : different packages %q != %q", pkg, c.PackageInfo.Name)
		return nil
	}
	entity := c.CodeConfig.TemplateName + kind
	if c.CodeConfig.DebugPrint {
		log.Printf("RecurseGenerate : processing %q %q ", c.CodeConfig.TemplateName, kind)
	}
	// already has it
	if _, has := c.keeper[entity]; has {
		if c.CodeConfig.DebugPrint {
			log.Printf("RecurseGenerate : %q already stored.", kind)
		}
		log.Printf("RecurseGenerate : %q ALREADY stored.", kind)
		return errors.New("RecurseGenerate : `" + kind + "` already stored. you are not checking that yourself?")
	}

	nt, err := c.StructByKey(kind)
	if nt == nil || err != nil {
		return err
	}
	if nt.IsImported {
		log.Printf("RecurseGenerate : %q in package %q it's imported", kind, pkg)
		return fmt.Errorf("RecurseGenerate : %q in package %q it's imported", kind, pkg)
	}
	if nt.Package != pkg {
		log.Printf("RecurseGenerate : %q in package %q it's a different pacakge %q", kind, pkg, nt.Package)
		return fmt.Errorf("RecurseGenerate : %q in package %q it's a different package %q", kind, pkg, nt.Package)
	}
	if IsBasic(nt.Kind) {
		log.Printf("RecurseGenerate : call for basic kind %q", nt.Kind)
		return errors.New("RecurseGenerate : don't recurse call for basic kind")
	}
	//log.Printf("RecurseGenerate : %q for kind %q generating in package %q", entity, nt.Kind, pkg)
	var buf strings.Builder
	c.keeper[entity] = "" // mark it empty
	err = c.tmpl.ExecuteTemplate(&buf, c.CodeConfig.TemplateName, nt)
	if err != nil {
		c.keeper[entity] = "//" + err.Error() // put a comment with that error
		log.Printf("RecurseGenerate template error : %v", err)
		return err
	}
	// everything went fine
	c.keeper[entity] = buf.String()
	//log.Printf("RecurseGenerate : %q for kind %q in package %q STORED", entity, nt.Kind, pkg)
	return nil
}

func (c *Code) ListStored() []string {
	var result []string
	for key, value := range c.keeper {
		if strings.HasPrefix(key, c.CodeConfig.TemplateName) {
			if r, ok := value.(string); ok {
				// len(0) is default template for main (
				if len(r) > 0 {
					result = append(result, r)
				}
			} else {
				// if it's not a string, we're ignoring it
				if c.CodeConfig.DebugPrint {
					log.Printf("%q has prefix %q but it's not a string and we're ignoring it", key, c.CodeConfig.TemplateName)
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
