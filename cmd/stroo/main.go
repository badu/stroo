package main

import (
	. "github.com/badu/stroo"
	"log"
)

func main() {
	command := Prepare()
	if command.Serve {
		StartPlayground()
		return
	}
	command.Check()
	command.Print()
	command.Path = "."
	if err := command.Do(); err != nil {
		log.Fatalf("error analysing : %v", err)
	}
	log.Println("done.")
}
