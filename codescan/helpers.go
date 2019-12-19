package codescan

// =====================================================================================================================
// Below code taken (and modified) from golang.org/x/tools
// =====================================================================================================================

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"unicode"
)

const (
	GOPATH        = "GOPATH"
	GOROOT        = "GOROOT"
	GO11MOD       = "GO111MODULE"
	GOMOD         = "GOMOD"
	PWD           = "PWD"
	ENV           = "env"
	GOARCH        = "GOARCH"
	UnsafePackage = "unsafe"
	SrcString     = "src"
	SrcPath       = "/" + SrcString
	SrcFPath      = "/" + SrcString + "/"
)

func printFlags() {
	type jsonFlag struct {
		Name  string
		Bool  bool
		Usage string
	}
	var flags []jsonFlag
	flag.VisitAll(func(f *flag.Flag) {
		boolFlag, ok := f.Value.(interface{ IsBoolFlag() bool })
		isBool := ok && boolFlag.IsBoolFlag()
		flags = append(flags, jsonFlag{f.Name, isBool, f.Usage})
	})
	data, err := json.MarshalIndent(flags, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(data)
}

func validIdent(name string) bool {
	for i, r := range name {
		if !(r == '_' || unicode.IsLetter(r) || i > 0 && unicode.IsDigit(r)) {
			return false
		}
	}
	return name != ""
}
