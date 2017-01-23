package main

import (
	"log"
)

func runTasks(tasks []task) {
	for _, t := range tasks {
		err := t.Check()
		if err == nil {
			log.Printf("%s: OK", t)
			continue
		}
		log.Printf("%s: %s", t, err)
		if !t.CanFix() {
			log.Printf("%s: can't fix")
			continue
		}
		if err = t.Fix(); err != nil {
			log.Printf("%s: can't fix: %s", t, err)
			continue
		}
		if err = t.Check(); err != nil {
			log.Printf("%s: fixed, but still broken?!: %s", t, err)
			continue
		}
		log.Printf("%s: OK", t)
	}
}

type task interface {
	String() string
	Check() error
	CanFix() bool
	Fix() error
}

