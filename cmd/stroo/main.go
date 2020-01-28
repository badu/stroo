package main

import (
	. "github.com/badu/stroo"
	"log"
	"os"
)

func main() {
	codeBuilder := DefaultAnalyzer()
	// set the logger
	log.SetFlags(0)
	log.SetPrefix(ToolName + ": ")
	// check flags
	if err := codeBuilder.Flags.Parse(os.Args[1:]); err != nil {
		log.Fatalf("error parsing flags: %v", err)
	}
	// create a command from our analyzer (so we don't pass parameters around)
	command := NewCommand(codeBuilder)
	if command.Serve {
		StartPlayground(command)
		return
	}
	// check if vital things are missing from the configuration
	if command.TemplateFile == "" || command.SelectedType == "" || (!command.TestMode && command.OutputFile == "") {
		codeBuilder.Flags.Usage()
		os.Exit(1)
	}

	// print the current configuration
	log.Printf("received params : %s\n", Print(codeBuilder, true))
	if loaded, err := LoadPackage("."); err != nil {
		log.Fatalf("error loading : %v", err)
	} else {
		if err := command.Analyse(codeBuilder, loaded); err != nil {
			log.Fatalf("error analysing : %v", err)
		}
	}
	if err := command.Generate(codeBuilder); err != nil {
		log.Fatalf("error generating : %v", err)
	}
	if command.TestMode {
		log.Printf("%s\n", command.Out.String())
		log.Println("file not written because test mode is set")
	}
}
