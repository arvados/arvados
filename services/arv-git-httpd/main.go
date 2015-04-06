package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

type config struct {
	Addr       string
	GitCommand string
	Root       string
	Pidfile    string
}

var theConfig *config

func init() {
	theConfig = &config{}
	flag.StringVar(&theConfig.Addr, "address", "0.0.0.0:80",
		"Address to listen on, \"host:port\".")
	flag.StringVar(&theConfig.GitCommand, "git-command", "/usr/bin/git",
		"Path to git executable. Each authenticated request will execute this program with a single argument, \"http-backend\".")
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln("Getwd():", err)
	}
	flag.StringVar(&theConfig.Root, "repo-root", cwd,
		"Path to git repositories.")

	flag.StringVar(
		&theConfig.Pidfile,
		"pid",
		"",
		"Path to write pid file")

	// MakeArvadosClient returns an error if token is unset (even
	// though we don't need to do anything requiring
	// authentication yet). We can't do this in newArvadosClient()
	// just before calling MakeArvadosClient(), though, because
	// that interferes with the env var needed by "run test
	// servers".
	os.Setenv("ARVADOS_API_TOKEN", "xxx")
}

func main() {
	flag.Parse()
	srv := &server{}
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}

	if theConfig.Pidfile != "" {
		f, err := os.Create(theConfig.Pidfile)
		if err != nil {
			log.Fatalf("Error writing pid file (%s): %s", theConfig.Pidfile, err.Error())
		}
		fmt.Fprint(f, os.Getpid())
		f.Close()
		defer os.Remove(theConfig.Pidfile)
	}

	log.Println("Listening at", srv.Addr)

	if err := srv.Wait(); err != nil {
		log.Fatal(err)
	}
}
