package codescan

// =====================================================================================================================
// Below code taken (and modified) from golang.org/x/tools
// =====================================================================================================================

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"sync"
)

type Action struct {
	once         sync.Once
	analyzer     *Analyzer
	pkg          *Package
	pass         *Pass
	dependencies Actions
	err          error
	isRoot       bool
	inputs       map[*Analyzer]interface{}
	result       interface{}
}

func (a Action) getResult() (interface{}, error) {
	if a.err != nil {
		log.Printf("%s: %v\n", a.analyzer.Name, a.err)
		return nil, a.err
	}
	return a.result, nil
}

func (a *Action) String() string {
	return fmt.Sprintf("%s@%s", a.analyzer, a.pkg)
}

func (a *Action) execOnce() {
	a.dependencies.execAll()

	var failures []string
	for _, action := range a.dependencies {
		if action.err != nil {
			failures = append(failures, action.String())
		}
	}

	if failures != nil {
		sort.Strings(failures)
		a.err = fmt.Errorf("failed prerequisites: %s", strings.Join(failures, ", "))
		return
	}

	inputs := make(map[*Analyzer]interface{})
	for _, dependency := range a.dependencies {
		if dependency.pkg == a.pkg {
			inputs[dependency.analyzer] = dependency.result
		}
	}

	pass := Pass{
		Analyzer:   a.analyzer,
		SourceFile: a.pkg.SourceFiles,
		Files:      a.pkg.Syntax,
		Pkg:        a.pkg.Types,
		TypesInfo:  a.pkg.TypesInfo,
		TypesSizes: a.pkg.TypesSizes,
		ResultOf:   inputs,
	}

	a.pass = &pass

	if a.pkg.IllTyped && !pass.Analyzer.RunDespiteErrors {
		a.err = fmt.Errorf("analysis skipped due to errors in package")
		return
	}

	a.result, a.err = pass.Analyzer.Runnner(&pass)
	if a.err == nil {
		if got, want := reflect.TypeOf(a.result), pass.Analyzer.ResultType; got != want {
			a.err = fmt.Errorf(
				"internal error: on package %s, analyzer %s returned a result of type %v, but declared ResultType %v",
				pass.Pkg.Path(), pass.Analyzer, got, want)
		}
	}
}

type Actions []*Action

func (a Actions) visitAll(printed map[*Action]struct{}) ([]interface{}, error) {
	var results []interface{}
	for _, act := range a {
		if _, ok := printed[act]; !ok {
			printed[act] = struct{}{}
			act.dependencies.visitAll(printed)
			result, err := act.getResult()
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
	}
	return results, nil
}

func (a Actions) GatherResults() ([]interface{}, int) {
	printed := make(map[*Action]struct{})
	result, err := a.visitAll(printed)
	if err != nil {
		return nil, 1
	}
	return result, 0
}

func (a Actions) execAll() {
	var wg sync.WaitGroup
	for _, action := range a {
		wg.Add(1)
		work := func(act *Action) {
			act.once.Do(act.execOnce)
			wg.Done()
		}
		go work(action)
	}
	wg.Wait()
}
