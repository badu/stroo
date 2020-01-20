package main

import (
	. "github.com/badu/stroo"
	"log"
)

func main() {
	command := Prepare()
	if command.Serve {
		StartPlayground(command)
		return
	}
	command.Check()
	log.Printf("received params : %s\n", command.Print(true))
	if loaded, err := command.Load("."); err != nil {
		log.Fatalf("error loading : %v", err)
	} else {
		if err := command.Analyse(loaded); err != nil {
			log.Fatalf("error analysing : %v", err)
		}
	}
	if err := command.Generate(); err != nil {
		log.Fatalf("error generating : %v", err)
	}
	if command.TestMode {
		log.Printf("%s\n", command.Out.String())
		log.Println("file not written because test mode is set")
	}
}
