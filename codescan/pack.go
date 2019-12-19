package codescan

// =====================================================================================================================
// Below code taken (and modified) from golang.org/x/tools
// =====================================================================================================================

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type (
	LoadMode int

	ErrorKind int

	goInfo struct {
		Sizes    *types.StdSizes
		Roots    []string
		Packages Packages
		rootDirs map[string]string
	}

	Error struct {
		Pos  string
		Msg  string
		Kind ErrorKind
	}

	pendingPackage struct {
		*Package
		importErrors map[string]error
		loadOnce     sync.Once
		state        uint8
		needSources  bool
		needTypes    bool
		initial      bool
	}

	pendingPackages []*pendingPackage

	loaderInfo struct {
		seenRoots    map[string]bool
		seenPackages Imports
		info         *goInfo
	}
)

const (
	NeedName LoadMode = 1 << iota
	NeedFiles
	NeedImports
	NeedDeps
	NeedTypes
	NeedSyntax
	NeedTypesInfo
	NeedTypesSizes
)

const (
	UnknownError ErrorKind = iota
	ListError
	ParseError
	TypeError
)

func (err Error) Error() string {
	pos := err.Pos
	if pos == "" {
		pos = "-"
	}
	return pos + ": " + err.Msg
}

func (r *loaderInfo) init(dr *goInfo) {
	r.info = dr
	r.seenRoots = map[string]bool{}
	r.seenPackages = Imports{}
	for _, pkg := range dr.Packages {
		r.seenPackages[pkg.ID] = pkg
	}
	for _, root := range dr.Roots {
		r.seenRoots[root] = true
	}
}

func (r *loaderInfo) addPackage(p *Package) {
	if r.seenPackages[p.ID] != nil {
		return
	}
	r.seenPackages[p.ID] = p
	r.info.Packages = append(r.info.Packages, p)
}

func (r *loaderInfo) addRoot(id string) {
	if r.seenRoots[id] {
		return
	}
	r.seenRoots[id] = true
	r.info.Roots = append(r.info.Roots, id)
}

func extractPackageName(filename string, contents []byte) (string, bool) {
	f, err := parser.ParseFile(token.NewFileSet(), filename, contents, parser.PackageClauseOnly)
	if err != nil {
		return "", false
	}
	return f.Name.Name, true
}

func pathMatchesQueries(path string, queries []string) bool {
	lastTwo := lastTwoComponents(path)
	for _, query := range queries {
		if strings.Contains(lastTwo, query) {
			return true
		}
		if hasHyphenOrUpperASCII(lastTwo) && !hasHyphenOrUpperASCII(query) {
			lastTwo = lowerASCIIAndRemoveHyphen(lastTwo)
			if strings.Contains(lastTwo, query) {
				return true
			}
		}
	}
	return false
}

func lastTwoComponents(v string) string {
	nslash := 0
	for i := len(v) - 1; i >= 0; i-- {
		if v[i] == '/' || v[i] == '\\' {
			nslash++
			if nslash == 2 {
				return v[i:]
			}
		}
	}
	return v
}

func hasHyphenOrUpperASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == '-' || ('A' <= b && b <= 'Z') {
			return true
		}
	}
	return false
}

func lowerASCIIAndRemoveHyphen(s string) string {
	buf := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		b := s[i]
		switch {
		case b == '-':
			continue
		case 'A' <= b && b <= 'Z':
			buf = append(buf, b+('a'-'A'))
		default:
			buf = append(buf, b)
		}
	}
	return string(buf)
}

type jsonPackage struct {
	ImportPath string
	Dir        string
	Name       string
	Export     string
	GoFiles    []string
	CgoFiles   []string
	Imports    []string
	ImportMap  map[string]string
	DepOnly    bool
	Error      *jsonPackageError
}

type jsonPackageError struct {
	ImportStack []string
	Pos         string
	Err         string
}

func absJoin(dir string, fileses ...[]string) []string {
	var res []string
	for _, files := range fileses {
		for _, file := range files {
			if !filepath.IsAbs(file) {
				file = filepath.Join(dir, file)
			}
			res = append(res, file)
		}
	}
	return res
}

func envDebug(envlist []string, args ...string) string {
	env := make(map[string]string)
	for _, kv := range envlist {
		split := strings.Split(kv, "=")
		k, v := split[0], split[1]
		env[k] = v
	}
	return fmt.Sprintf("%s=%v %s=%v %s=%v %s=%v go %v", GOROOT, env[GOROOT], GOPATH, env[GOPATH], GO11MOD, env[GO11MOD], PWD, env[PWD], args)
}

type importerFunc func(string) (*types.Package, error)

func (f importerFunc) Import(path string) (*types.Package, error) { return f(path) }

func sameFile(x, y string) bool {
	if x == y {
		return true
	}
	if strings.EqualFold(filepath.Base(x), filepath.Base(y)) {
		if xi, err := os.Stat(x); err == nil {
			if yi, err := os.Stat(y); err == nil {
				return os.SameFile(xi, yi)
			}
		}
	}
	return false
}

func shortCallGoTool(ctx context.Context, env []string, dir string, usesExportData bool, args ...string) (*bytes.Buffer, *bytes.Buffer, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Env = append(append([]string{}, env...), "PWD="+dir)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return nil, nil, fmt.Errorf("couldn't exec 'go %v': %s %T", args, err, err)
		}
		if !usesExportData {
			return nil, nil, fmt.Errorf("go %v: %s: %s", args, exitErr, stderr)
		}
	}
	return stdout, stderr, nil
}

func goToolListSizes(ctx context.Context, env []string, dir string) (types.Sizes, error) {
	usesExportData := false // forced false
	args := []string{"list", "-f", "{{context.GOARCH}} {{context.Compiler}}"}
	args = append(args, "--", UnsafePackage)
	stdout, stderr, err := shortCallGoTool(ctx, env, dir, usesExportData, args...)
	var goarch, compiler string
	if err != nil {
		if strings.Contains(err.Error(), "cannot find main module") {
			envout, _, enverr := shortCallGoTool(ctx, env, dir, usesExportData, ENV, GOARCH)
			if enverr != nil {
				return nil, err
			}
			goarch = strings.TrimSpace(envout.String())
			compiler = "gc"
		} else {
			return nil, err
		}
	} else {
		fields := strings.Fields(stdout.String())
		if len(fields) < 2 {
			return nil, fmt.Errorf("could not parse GOARCH and Go compiler in format \"<GOARCH> <compiler>\" from stdout of go command:\n%s\ndir: %s\nstdout: <<%s>>\nstderr: <<%s>>",
				envDebug(env, args...), dir, stdout.String(), stderr.String())
		}
		goarch = fields[0]
		compiler = fields[1]
	}
	return types.SizesFor(compiler, goarch), nil
}

type Imports map[string]*Package

func newImports(size int) Imports {
	return make(map[string]*Package, size)
}

func (i Imports) visit(ld *loader, seen map[*pendingPackage]bool, view map[string]*types.Package) {
	for _, p := range i {
		pkg := ld.mPendingPkgs[p.ID]
		if !seen[pkg] {
			seen[pkg] = true
			view[pkg.PkgPath] = pkg.Types
			pkg.Imports.visit(ld, seen, view)
		}
	}
}

type Package struct {
	ID          string
	Name        string
	PkgPath     string
	Errors      []Error
	GoFiles     []string
	ExportFile  string
	Imports     Imports
	Types       *types.Package
	SourceFiles *token.FileSet
	Syntax      []*ast.File
	TypesInfo   *types.Info
	TypesSizes  types.Sizes
	IllTyped    bool
}

type Packages []*Package

type SeenKey struct {
	*Analyzer
	*Package
}

func (p Packages) PrintErrors() int {
	var n int
	p.Visit(func(pkg *Package) {
		for _, err := range pkg.Errors {
			fmt.Fprintln(os.Stderr, err)
			n++
		}
	})
	return n
}

func (p Packages) Visit(after func(*Package)) {
	seenPackages := make(map[*Package]struct{})
	for _, pkg := range p {
		pkg.Visit(after, seenPackages)
	}
}

func (p Packages) Analyze(analyzers Analyzers) Actions {
	actions := make(map[SeenKey]*Action)
	var mkAction func(a *Analyzer, pkg *Package) *Action
	mkAction = func(a *Analyzer, pkg *Package) *Action {
		k := SeenKey{a, pkg}
		act, ok := actions[k]
		if !ok {
			act = &Action{analyzer: a, pkg: pkg}
			for _, req := range a.Requires {
				act.dependencies = append(act.dependencies, mkAction(req, pkg))
			}
			actions[k] = act
		}
		return act
	}
	var roots Actions
	for _, a := range analyzers {
		for _, pkg := range p {
			root := mkAction(a, pkg)
			root.isRoot = true
			roots = append(roots, root)
		}
	}
	roots.execAll()
	return roots
}

func (p *Package) Visit(after func(*Package), seenPackages map[*Package]struct{}) {
	if _, ok := seenPackages[p]; !ok {
		seenPackages[p] = struct{}{}
		paths := make([]string, 0, len(p.Imports))
		for importPath := range p.Imports {
			paths = append(paths, importPath)
		}
		sort.Strings(paths)
		for _, aPath := range paths {
			p.Imports[aPath].Visit(after, seenPackages)
		}
		if after != nil {
			after(p)
		}
	}
}
func (p *Package) reclaimPackage(id, filename string, contents []byte) bool {
	if p.ID != id {
		return false
	}
	if len(p.Errors) != 1 {
		return false
	}
	if p.Name != "" || p.ExportFile != "" {
		return false
	}
	if len(p.GoFiles) > 0 {
		return false
	}
	if len(p.Imports) > 0 {
		return false
	}
	pkgName, ok := extractPackageName(filename, contents)
	if !ok {
		return false
	}
	p.Name = pkgName
	p.Errors = nil
	return true
}

func (p *Package) String() string { return p.ID }

func Load(patterns []string) (Packages, error) {
	wkDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	ld := &loader{
		parseCache:  map[string]*parseValue{},
		Mode:        NeedName | NeedFiles | NeedImports | NeedTypes | NeedTypesSizes | NeedSyntax | NeedTypesInfo,
		Env:         os.Environ(),
		Context:     context.Background(),
		Dir:         wkDir,
		SourceFiles: token.NewFileSet(),
		ParsedFileCallback: func(sourceFiles *token.FileSet, filename string, src []byte) (*ast.File, error) {
			return parser.ParseFile(sourceFiles, filename, src, parser.AllErrors|parser.ParseComments)
		},
	}

	response, err := ld.createList(patterns...)
	if err != nil {
		return nil, err
	}
	ld.sizes = response.Sizes

	initial, err := ld.init(response.Roots, response.Packages)
	if err == nil {
		if n := initial.PrintErrors(); n > 1 {
			err = fmt.Errorf("%d errors during loading", n)
		} else if n == 1 {
			err = fmt.Errorf("error during loading")
		} else if len(initial) == 0 {
			err = fmt.Errorf("%s matched no packages", strings.Join(patterns, " "))
		}
	}
	return initial, err
}
