package codescan

// =====================================================================================================================
// Below code taken (and modified) from golang.org/x/tools
// =====================================================================================================================

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/badu/unbundle/gopathwalk"
	"go/ast"
	"go/scanner"
	"go/token"
	"go/types"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

type parseValue struct {
	file  *ast.File
	err   error
	ready chan struct{}
}

type callback func(*token.FileSet, string, []byte) (*ast.File, error)

type loader struct {
	Mode               LoadMode
	Context            context.Context
	Dir                string
	Env                []string
	SourceFiles        *token.FileSet
	ParsedFileCallback callback
	mPendingPkgs       map[string]*pendingPackage
	sizes              types.Sizes
	parseCache         map[string]*parseValue
	parseCacheMu       sync.Mutex
	stack              pendingPackages
	srcPkgs            pendingPackages
}

type deferredInfo func() *goInfo

func (ld *loader) overlays(getGoInfo deferredInfo, info *loaderInfo, pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}

	dr, err := ld.makeList(getGoInfo, pkgs...)
	if err != nil {
		return err
	}
	for _, pkg := range dr.Packages {
		info.addPackage(pkg)
	}
	_, needPkgs, err := ld.searchOverlays(getGoInfo, info)
	if err != nil {
		return err
	}
	return ld.overlays(getGoInfo, info, needPkgs)
}

func (ld *loader) contains(getGoInfo deferredInfo, info *loaderInfo, queries []string) error {
	for _, query := range queries {
		fdir := filepath.Dir(query)
		pattern, err := filepath.Abs(fdir)
		if err != nil {
			return fmt.Errorf("could not determine absolute path of file= query path %q: %v", query, err)
		}
		dirResponse, err := ld.makeList(getGoInfo, pattern)
		if err != nil {
			var queryErr error
			if dirResponse, queryErr = ld.addPackage(getGoInfo, pattern, query); queryErr != nil {
				return err
			}
		}

		if len(dirResponse.Packages) == 1 && len(dirResponse.Packages[0].Errors) == 1 {
			var queryErr error
			if dirResponse, queryErr = ld.addPackage(getGoInfo, pattern, query); queryErr != nil {
				return err // return the original error
			}
		}
		isRoot := make(map[string]bool, len(dirResponse.Roots))
		for _, root := range dirResponse.Roots {
			isRoot[root] = true
		}
		for _, pkg := range dirResponse.Packages {
			info.addPackage(pkg)
			if !isRoot[pkg.ID] {
				continue
			}
			for _, pkgFile := range pkg.GoFiles {
				if filepath.Base(query) == filepath.Base(pkgFile) {
					info.addRoot(pkg.ID)
					break
				}
			}
		}
	}
	return nil
}

func (ld *loader) addPackage(getGoInfo deferredInfo, pattern, query string) (*goInfo, error) {
	dirResponse, err := ld.makeList(getGoInfo, query)
	if err != nil {
		return nil, err
	}
	if len(dirResponse.Packages) == 0 && err == nil {
		dirResponse.Packages = append(dirResponse.Packages, &Package{
			ID:      "command-line-arguments",
			PkgPath: query,
			GoFiles: []string{query},
			Imports: Imports{},
		})
		dirResponse.Roots = append(dirResponse.Roots, "command-line-arguments")
	}
	return dirResponse, nil
}

func (ld *loader) makeList(getGoInfo deferredInfo, words ...string) (*goInfo, error) {
	buf, err := callGoTool(ld.Context, ld.Dir, ld.Env, goToolArgs(words, ld.Mode)...)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]*jsonPackage)
	var response goInfo
	for dec := json.NewDecoder(buf); dec.More(); {
		jsonPckg := new(jsonPackage)
		if err := dec.Decode(jsonPckg); err != nil {
			return nil, fmt.Errorf("JSON decoding failed: %v", err)
		}

		if jsonPckg.ImportPath == "" {
			if jsonPckg.Error != nil {
				return nil, Error{
					Pos: jsonPckg.Error.Pos,
					Msg: jsonPckg.Error.Err,
				}
			}
			return nil, fmt.Errorf("package missing import path: %+v", jsonPckg)
		}

		if filepath.IsAbs(jsonPckg.ImportPath) && jsonPckg.Error != nil {
			pkgPath, ok := getPkgPath(jsonPckg.ImportPath, getGoInfo)
			if ok {
				jsonPckg.ImportPath = pkgPath
			}
		}

		if cached, ok := seen[jsonPckg.ImportPath]; ok {
			if !reflect.DeepEqual(jsonPckg, cached) {
				return nil, fmt.Errorf("internal error: go list gives conflicting information for package %v", jsonPckg.ImportPath)
			}
			continue
		}

		seen[jsonPckg.ImportPath] = jsonPckg
		pkg := &Package{
			Name:    jsonPckg.Name,
			ID:      jsonPckg.ImportPath,
			GoFiles: absJoin(jsonPckg.Dir, jsonPckg.GoFiles, jsonPckg.CgoFiles),
		}

		if i := strings.IndexByte(pkg.ID, ' '); i >= 0 {
			pkg.PkgPath = pkg.ID[:i]
		} else {
			pkg.PkgPath = pkg.ID
		}

		if pkg.PkgPath == UnsafePackage {
			pkg.GoFiles = nil
		}

		if jsonPckg.Dir != "" && !filepath.IsAbs(jsonPckg.Dir) {
			log.Fatalf("internal error: go list returned non-absolute Package.Dir: %s", jsonPckg.Dir)
			// todo return error ?
		}

		ids := make(map[string]bool)
		for _, id := range jsonPckg.Imports {
			ids[id] = true
		}

		pkg.Imports = Imports{}
		for importPath, id := range jsonPckg.ImportMap {
			pkg.Imports[importPath] = &Package{ID: id}
			delete(ids, id)
		}
		for id := range ids {
			if id == "C" {
				continue
			}

			pkg.Imports[id] = &Package{ID: id}
		}

		if !jsonPckg.DepOnly {
			response.Roots = append(response.Roots, pkg.ID)
		}

		if jsonPckg.Error != nil {
			pkg.Errors = append(pkg.Errors, Error{
				Pos: jsonPckg.Error.Pos,
				Msg: strings.TrimSpace(jsonPckg.Error.Err),
			})
		}

		response.Packages = append(response.Packages, pkg)
	}

	return &response, nil
}

func getPkgPath(dir string, getGoInfo deferredInfo) (string, bool) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		log.Printf("error getting absolute path of %s: %v\n", dir, err)
		return "", false
	}
	for rdir, rpath := range getGoInfo().rootDirs {
		absRdir, err := filepath.Abs(rdir)
		if err != nil {
			log.Printf("error getting absolute path of %s: %v\n", rdir, err)
			continue
		}
		if !strings.HasPrefix(absDir, absRdir) {
			log.Printf("%s does not have prefix %s\n", absDir, absRdir)
			continue
		}
		r, err := filepath.Rel(rdir, dir)
		if err != nil {
			continue
		}
		if rpath != "" {
			return path.Join(rpath, filepath.ToSlash(r)), true
		}
		return filepath.ToSlash(r), true
	}
	return "", false
}

func goToolArgs(words []string, mode LoadMode) []string {
	fullargs := []string{
		"list", "-e", "-json",
		fmt.Sprintf("-compiled=%t", mode&(NeedSyntax|NeedTypesInfo|NeedTypesSizes) != 0),
		fmt.Sprintf("-export=%t", mode&NeedTypes != 0 && mode&NeedDeps == 0),
		fmt.Sprintf("-deps=%t", mode&NeedImports != 0),
		fmt.Sprintf("-find=%t", mode&(NeedImports|NeedTypes|NeedSyntax|NeedTypesInfo) == 0),
	}
	fullargs = append(fullargs, "--")
	fullargs = append(fullargs, words...)
	return fullargs
}

const maxStackLength = 50

func getStackTrace() string {
	stackBuf := make([]uintptr, maxStackLength)
	length := runtime.Callers(3, stackBuf[:])
	stack := stackBuf[:length]

	trace := ""
	frames := runtime.CallersFrames(stack)
	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.File, "runtime/") {
			trace = trace + fmt.Sprintf("\n\tFile: %s, Line: %d. Function: %s", frame.File, frame.Line, frame.Function)
		}
		if !more {
			break
		}
	}
	return trace
}

func callGoTool(ctx context.Context, dir string, env []string, args ...string) (*bytes.Buffer, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Env = append(append([]string{}, env...), PWD+"="+dir)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.Error); ok && ee.Err == exec.ErrNotFound {
			return nil, fmt.Errorf("'go list' driver requires 'go', but %s", exec.ErrNotFound)
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return nil, fmt.Errorf("couldn't exec 'go %v': %s %T", args, err, err)
		}
		return nil, fmt.Errorf("%v\nSTDERR:%v", exitErr, stderr.String())
	}
	return stdout, nil
}

func (ld *loader) visit(lpkg *pendingPackage) bool {
	switch lpkg.state {
	case sDone:
		return lpkg.needSources
	case sPending:
		panic("internal error: sPending node")
	}
	lpkg.state = sPending
	ld.stack = append(ld.stack, lpkg)
	stubs := lpkg.Imports
	if ld.Mode&NeedImports != 0 {
		lpkg.Imports = newImports(len(stubs))
		for importPath, ipkg := range stubs {
			var importErr error
			imp := ld.mPendingPkgs[ipkg.ID]
			if imp == nil {
				importErr = fmt.Errorf("missing package: %q", ipkg.ID)
			} else if imp.state == sPending {
				importErr = fmt.Errorf("import cycle: %s", ld.stack)
			}
			if importErr != nil {
				if lpkg.importErrors == nil {
					lpkg.importErrors = make(map[string]error)
				}
				lpkg.importErrors[importPath] = importErr
				continue
			}

			if ld.visit(imp) {
				lpkg.needSources = true
			}
			lpkg.Imports[importPath] = imp.Package
		}
	}
	if lpkg.needSources {
		ld.srcPkgs = append(ld.srcPkgs, lpkg)
	}
	if ld.Mode&NeedTypesSizes != 0 {
		lpkg.TypesSizes = ld.sizes
	}
	ld.stack = ld.stack[:len(ld.stack)-1]
	lpkg.state = sDone

	return lpkg.needSources
}

func (ld *loader) init(roots []string, list Packages) (Packages, error) {
	rootMap := make(map[string]int, len(roots))
	for i, root := range roots {
		rootMap[root] = i
	}
	ld.mPendingPkgs = make(map[string]*pendingPackage)
	var initial = make(pendingPackages, len(roots))
	for _, pkg := range list {
		rootIndex := -1
		if i, found := rootMap[pkg.ID]; found {
			rootIndex = i
		}
		lpkg := &pendingPackage{
			Package:     pkg,
			needTypes:   (ld.Mode&(NeedTypes|NeedTypesInfo) != 0 && ld.Mode&NeedDeps != 0 && rootIndex < 0) || rootIndex >= 0,
			needSources: (ld.Mode&(NeedSyntax|NeedTypesInfo) != 0 && ld.Mode&NeedDeps != 0 && rootIndex < 0) || rootIndex >= 0 || pkg.ExportFile == "" && pkg.PkgPath != UnsafePackage,
		}
		ld.mPendingPkgs[lpkg.ID] = lpkg
		if rootIndex >= 0 {
			initial[rootIndex] = lpkg
			lpkg.initial = true
		}
	}
	for i, root := range roots {
		if initial[i] == nil {
			return nil, fmt.Errorf("root package %v is missing", root)
		}
	}

	if ld.Mode&NeedImports == 0 {
		for _, lpkg := range initial {
			lpkg.Imports = nil
		}
	} else {
		for _, lpkg := range initial {
			ld.visit(lpkg)
		}
	}
	if ld.Mode&NeedImports != 0 && ld.Mode&NeedTypes != 0 {
		for _, lpkg := range ld.srcPkgs {
			for _, ipkg := range lpkg.Imports {
				imp := ld.mPendingPkgs[ipkg.ID]
				imp.needTypes = true
			}
		}
	}
	if ld.Mode&NeedTypes != 0 || ld.Mode&NeedSyntax != 0 {
		var wg sync.WaitGroup
		for _, lpkg := range initial {
			wg.Add(1)
			go func(lpkg *pendingPackage) {
				ld.loadRecursive(lpkg)
				wg.Done()
			}(lpkg)
		}
		wg.Wait()
	}

	result := make(Packages, len(initial))
	for i, lpkg := range initial {
		result[i] = lpkg.Package
	}
	for i := range ld.mPendingPkgs {
		if ld.Mode&NeedName == 0 {
			ld.mPendingPkgs[i].Name = ""
			ld.mPendingPkgs[i].PkgPath = ""
		}
		if ld.Mode&NeedFiles == 0 {
			ld.mPendingPkgs[i].GoFiles = nil
		}
		if ld.Mode&NeedImports == 0 {
			ld.mPendingPkgs[i].Imports = nil
		}
		ld.mPendingPkgs[i].ExportFile = ""
		if ld.Mode&NeedTypes == 0 {
			ld.mPendingPkgs[i].Types = nil
			ld.mPendingPkgs[i].SourceFiles = nil
			ld.mPendingPkgs[i].IllTyped = false
		}
		if ld.Mode&NeedSyntax == 0 {
			ld.mPendingPkgs[i].Syntax = nil
		}
		if ld.Mode&NeedTypesInfo == 0 {
			ld.mPendingPkgs[i].TypesInfo = nil
		}
		if ld.Mode&NeedTypesSizes == 0 {
			ld.mPendingPkgs[i].TypesSizes = nil
		}
	}

	return result, nil
}

func (ld *loader) loadRecursive(target *pendingPackage) {
	target.loadOnce.Do(func() {
		var wg sync.WaitGroup
		for _, ipkg := range target.Imports {
			importedPkg := ld.mPendingPkgs[ipkg.ID]
			wg.Add(1)
			go func(imp *pendingPackage) {
				ld.loadRecursive(imp)
				wg.Done()
			}(importedPkg)
		}
		wg.Wait()
		ld.loadPackage(target)
	})
}

func (ld *loader) loadPackage(target *pendingPackage) {
	if target.PkgPath == UnsafePackage {
		target.Types = types.Unsafe
		target.SourceFiles = ld.SourceFiles
		target.Syntax = []*ast.File{}
		target.TypesInfo = new(types.Info)
		target.TypesSizes = ld.sizes
		return
	}

	target.Types = types.NewPackage(target.PkgPath, target.Name)
	target.SourceFiles = ld.SourceFiles

	if !target.needTypes {
		return
	}

	appendError := func(err error) {
		var errs []Error
		switch err := err.(type) {
		case Error:
			errs = append(errs, err)
		case *os.PathError:
			errs = append(errs, Error{
				Pos:  err.Path + ":1",
				Msg:  err.Err.Error(),
				Kind: ParseError,
			})
		case scanner.ErrorList:
			for _, err := range err {
				errs = append(errs, Error{
					Pos:  err.Pos.String(),
					Msg:  err.Msg,
					Kind: ParseError,
				})
			}
		case types.Error:
			errs = append(errs, Error{
				Pos:  err.Fset.Position(err.Pos).String(),
				Msg:  err.Msg,
				Kind: TypeError,
			})
		default:
			errs = append(errs, Error{
				Pos:  "-",
				Msg:  err.Error(),
				Kind: UnknownError,
			})
			log.Printf("internal error: error %q (%T) without position", err, err)
			// todo return error
		}
		target.Errors = append(target.Errors, errs...)
	}

	files, errs := ld.parseFiles(target.GoFiles)
	for _, err := range errs {
		appendError(err)
	}

	target.Syntax = files
	if ld.Mode&NeedTypes == 0 {
		return
	}

	target.TypesInfo = &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Scopes:     make(map[ast.Node]*types.Scope),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	target.TypesSizes = ld.sizes

	importer := importerFunc(func(path string) (*types.Package, error) {
		if path == UnsafePackage {
			return types.Unsafe, nil
		}
		ipkg := target.Imports[path]
		if ipkg == nil {
			if err := target.importErrors[path]; err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("no metadata for %s", path)
		}

		if ipkg.Types != nil && ipkg.Types.Complete() {
			return ipkg.Types, nil
		}
		log.Fatalf("internal error: package %q without types was imported from %q", path, target)
		panic("unreachable")
	})

	tc := &types.Config{
		Importer:         importer,
		IgnoreFuncBodies: ld.Mode&NeedDeps == 0 && !target.initial,
		Error:            appendError,
		Sizes:            ld.sizes,
	}

	checkErr := types.NewChecker(tc, ld.SourceFiles, target.Types, target.TypesInfo).Files(target.Syntax)
	if checkErr != nil {
		log.Printf("Error checking : %v\n", checkErr)
	}
	target.importErrors = nil

	if tc.FakeImportC {
	outer:
		for _, f := range target.Syntax {
			for _, imp := range f.Imports {
				if imp.Path.Value == `"C"` {
					err := types.Error{Fset: ld.SourceFiles, Pos: imp.Pos(), Msg: `import "C" ignored`}
					appendError(err)
					break outer
				}
			}
		}
	}
	illTyped := len(target.Errors) > 0
	if !illTyped {
		for _, imp := range target.Imports {
			if imp.IllTyped {
				illTyped = true
				break
			}
		}
	}
	target.IllTyped = illTyped
}

func (ld *loader) searchOverlays(getGoInfo deferredInfo, info *loaderInfo) ([]string, []string, error) {
	var (
		modifiedPkgs []string
		needPkgs     []string
		err          error
	)
	havePkgs := make(map[string]string)
	needPkgsSet := make(map[string]bool)
	modifiedPkgsSet := make(map[string]bool)

	for _, pkg := range info.info.Packages {
		havePkgs[pkg.PkgPath] = pkg.ID
	}

	toPkgPath := func(id string) string {
		i := strings.IndexByte(id, ' ')
		if i >= 0 {
			return id[:i]
		}
		return id
	}

	for _, pkg := range info.info.Packages {
		for _, imp := range pkg.Imports {
			pkgPath := toPkgPath(imp.ID)
			if _, ok := havePkgs[pkgPath]; !ok {
				needPkgsSet[pkgPath] = true
			}
		}
	}

	modifiedPkgs = make([]string, 0, len(modifiedPkgsSet))
	for pkg := range modifiedPkgsSet {
		modifiedPkgs = append(modifiedPkgs, pkg)
	}
	return modifiedPkgs, needPkgs, err
}

func (ld *loader) determineRootDirs() map[string]string {
	out, err := callGoTool(ld.Context, ld.Dir, ld.Env, "list", "-m", "-json", "all")
	if err != nil {
		return ld.GoPath()
	}
	m := map[string]string{}
	type jsonMod struct{ Path, Dir string }
	for dec := json.NewDecoder(out); dec.More(); {
		mod := new(jsonMod)
		if err := dec.Decode(mod); err != nil {
			return m
		}
		if mod.Dir != "" && mod.Path != "" {
			m[mod.Dir] = mod.Path
		}
	}
	return m
}

func (ld *loader) GoPath() map[string]string {
	m := map[string]string{}
	out, err := callGoTool(ld.Context, ld.Dir, ld.Env, ENV, GOPATH)
	if err != nil {
		return m
	}
	for _, p := range filepath.SplitList(string(bytes.TrimSpace(out.Bytes()))) {
		m[filepath.Join(p, SrcString)] = ""
	}
	return m
}

func (ld *loader) createList(patterns ...string) (*goInfo, error) {
	var (
		sizes    types.Sizes
		sizesErr error
		sizesWg  sync.WaitGroup
	)
	if ld.Mode&NeedTypesSizes != 0 || ld.Mode&NeedTypes != 0 {
		sizesWg.Add(1)
		go func() {
			sizes, sizesErr = goToolListSizes(ld.Context, ld.Env, ld.Dir)
			sizesWg.Done()
		}()
	}
	defer sizesWg.Wait()

	var (
		goInf         goInfo
		rootDirsReady = make(chan struct{})
	)

	go func() {
		goInf.rootDirs = ld.determineRootDirs()
		close(rootDirsReady)
	}()

	var getGoInfo deferredInfo = func() *goInfo {
		<-rootDirsReady
		return &goInf
	}
	defer getGoInfo()

	var (
		containFiles  []string
		packagesNamed []string
	)
	restPatterns := make([]string, 0, len(patterns))

extractQueries:
	for _, pattern := range patterns {
		eqidx := strings.Index(pattern, "=")
		if eqidx < 0 {
			restPatterns = append(restPatterns, pattern)
		} else {
			query, value := pattern[:eqidx], pattern[eqidx+len("="):]
			switch query {
			case "file":
				containFiles = append(containFiles, value)
			case "pattern":
				restPatterns = append(restPatterns, value)
			case "":
				restPatterns = append(restPatterns, pattern)
			default:
				for _, letter := range query {
					if letter < 'a' || letter > 'z' {
						restPatterns = append(restPatterns, pattern)
						continue extractQueries
					}
				}
				return nil, fmt.Errorf("invalid query type %q in query pattern %q", query, pattern)
			}
		}
	}

	info := &loaderInfo{}
	var err error

	if len(restPatterns) > 0 || len(patterns) == 0 {
		pList, err := ld.makeList(getGoInfo, restPatterns...)
		if err != nil {
			return nil, err
		}
		info.init(pList)
	} else {
		info.init(&goInfo{})
	}

	sizesWg.Wait()
	if sizesErr != nil {
		return nil, sizesErr
	}
	info.info.Sizes, _ = sizes.(*types.StdSizes)

	var containsCandidates []string

	if len(containFiles) != 0 {
		if err := ld.contains(getGoInfo, info, containFiles); err != nil {
			return nil, err
		}
	}

	if len(packagesNamed) != 0 {
		if err := ld.searchNamed(getGoInfo, info, packagesNamed); err != nil {
			return nil, err
		}
	}

	modifiedPkgs, needPkgs, err := ld.searchOverlays(getGoInfo, info)
	if err != nil {
		return nil, err
	}
	if len(containFiles) > 0 {
		containsCandidates = append(containsCandidates, modifiedPkgs...)
		containsCandidates = append(containsCandidates, needPkgs...)
	}
	if err := ld.overlays(getGoInfo, info, needPkgs); err != nil {
		return nil, err
	}

	if len(containFiles) > 0 {
		for _, id := range containsCandidates {
			pkg, ok := info.seenPackages[id]
			if !ok {
				info.addPackage(&Package{
					ID: id,
					Errors: []Error{
						{
							Kind: ListError,
							Msg:  fmt.Sprintf("package %s expected but not seen", id),
						},
					},
				})
				continue
			}
			for _, f := range containFiles {
				for _, g := range pkg.GoFiles {
					if sameFile(f, g) {
						info.addRoot(id)
					}
				}
			}
		}
	}

	return info.info, nil
}

func (ld *loader) roots() ([]gopathwalk.Root, error) {
	stdout, err := callGoTool(ld.Context, ld.Dir, ld.Env, ENV, GOROOT, GOPATH, GOMOD)
	if err != nil {
		return nil, err
	}

	fields := strings.Split(stdout.String(), "\n")
	if len(fields) != 4 || len(fields[3]) != 0 {
		return nil, fmt.Errorf("unexpected go env output: %q", stdout.String())
	}

	goRoot, goPath, _ := fields[0], filepath.SplitList(fields[1]), fields[2]

	var roots []gopathwalk.Root
	roots = append(roots, gopathwalk.Root{
		Path: filepath.Join(goRoot, SrcPath),
		Type: gopathwalk.RootGOROOT,
	})

	for _, p := range goPath {
		roots = append(roots, gopathwalk.Root{
			Path: filepath.Join(p, SrcPath),
			Type: gopathwalk.RootGOPATH,
		})
	}

	return roots, nil
}

func (ld *loader) searchNamed(getGoInfo deferredInfo, info *loaderInfo, packagesNamed []string) error {
	roots, err := ld.roots()
	if err != nil {
		return err
	}
	var (
		matchesMu     sync.Mutex
		simpleMatches []string
	)
	gopathwalk.Walk(
		roots,
		func(root gopathwalk.Root, dir string) {
			matchesMu.Lock()
			defer matchesMu.Unlock()
			currentPath := dir
			if dir != root.Path {
				currentPath = dir[len(root.Path)+1:]
			}
			if pathMatchesQueries(currentPath, packagesNamed) {
				switch root.Type {
				case gopathwalk.RootCurrentModule:
					rel, err := filepath.Rel(ld.Dir, dir)
					if err != nil {
						panic(err)
					}
					simpleMatches = append(simpleMatches, "./"+rel)
				case gopathwalk.RootGOPATH, gopathwalk.RootGOROOT:
					simpleMatches = append(simpleMatches, currentPath)
				}
			}
		},
		gopathwalk.Options{ModulesEnabled: false},
	)

	if len(simpleMatches) != 0 {
		resp, err := ld.makeList(getGoInfo, simpleMatches...)
		if err != nil {
			return err
		}
		for _, pkg := range resp.Packages {
			info.addPackage(pkg)
			for _, name := range packagesNamed {
				if pkg.Name == name {
					info.addRoot(pkg.ID)
					break
				}
			}
		}
	}

	return nil
}

func (ld *loader) parseFiles(fileNames []string) ([]*ast.File, []error) {
	var wg sync.WaitGroup
	numFiles := len(fileNames)
	parsed := make([]*ast.File, numFiles)
	errs := make([]error, numFiles)
	for idx, file := range fileNames {
		if ld.Context.Err() != nil {
			parsed[idx] = nil
			errs[idx] = ld.Context.Err()
			continue
		}
		wg.Add(1)
		go func(index int, filename string) {
			parsed[index], errs[index] = ld.parseFile(filename)
			wg.Done()
		}(idx, file)
	}
	wg.Wait()

	var where int
	for _, file := range parsed {
		if file != nil {
			parsed[where] = file
			where++
		}
	}
	parsed = parsed[:where]
	where = 0
	for _, err := range errs {
		if err != nil {
			errs[where] = err
			where++
		}
	}
	errs = errs[:where]
	return parsed, errs
}

func (ld *loader) parseFile(filename string) (*ast.File, error) {
	ld.parseCacheMu.Lock()
	cached, ok := ld.parseCache[filename]
	if ok {
		ld.parseCacheMu.Unlock()
		<-cached.ready
		return cached.file, cached.err
	}

	cached = &parseValue{ready: make(chan struct{})}
	ld.parseCache[filename] = cached
	ld.parseCacheMu.Unlock()

	var src []byte

	var ioLimit = make(chan bool, 20)
	var err error
	if src == nil {
		ioLimit <- true
		src, err = ioutil.ReadFile(filename)
		<-ioLimit
	}

	if err != nil {
		cached.err = err
	} else {
		cached.file, cached.err = ld.ParsedFileCallback(ld.SourceFiles, filename, src)
	}

	close(cached.ready)
	return cached.file, cached.err
}
