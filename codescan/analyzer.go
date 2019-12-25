package codescan

// =====================================================================================================================
// Below code taken (and modified) from golang.org/x/tools
// =====================================================================================================================

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
)

type Analyzer struct {
	Name             string
	Doc              string
	Requires         Analyzers
	ResultType       reflect.Type
	Flags            flag.FlagSet
	Runnner          func(*Pass) (interface{}, error)
	RunDespiteErrors bool
	PrintDebug       bool
}

func (a *Analyzer) String() string { return a.Name }

const (
	sInit = iota
	sPending
	sDone
	sFinished
)

func (a *Analyzer) visit(state map[*Analyzer]uint8) error {
	if state[a] == sInit {
		state[a] = sPending
		if !validIdent(a.Name) {
			return fmt.Errorf("invalid analyzer name %q", a)
		}
		if a.Doc == "" {
			return fmt.Errorf("analyzer %q is undocumented", a)
		}
		for i, req := range a.Requires {
			if err := req.visit(state); err != nil {
				return fmt.Errorf("%s.Requires[%d]: %v", a.Name, i, err)
			}
		}
		state[a] = sDone
	}
	return nil
}

type Analyzers []*Analyzer

func (a Analyzers) Validate() error {
	states := make(map[*Analyzer]uint8)
	for _, analyzer := range a {
		if err := analyzer.visit(states); err != nil {
			return err
		}
	}
	for _, analyzer := range a {
		if states[analyzer] == sFinished {
			return fmt.Errorf("duplicate analyzer: %s", analyzer.Name)
		}
		states[analyzer] = sFinished
	}

	return nil
}

func (a Analyzers) ParseFlags() Analyzers {
	for _, analyzer := range a {
		var prefix string

		analyzer.Flags.VisitAll(func(f *flag.Flag) {
			if flag.Lookup(f.Name) != nil {
				log.Printf("%s flag -%s would conflict with driver; skipping", analyzer.Name, f.Name)
				return
			}

			name := prefix + f.Name
			flag.Var(f.Value, name, f.Usage)
		})
	}
	pFlags := flag.Bool("flags", false, "print analyzer flags")

	flag.Parse()

	if *pFlags {
		printFlags()
		os.Exit(0)
	}

	return a
}
