package main

import (
	"flag"
	"log"
	"os"
)

func init() {
	// MakeArvadosClient returns an error if this env var isn't
	// available as a default token (even if we explicitly set a
	// different token before doing anything with the client). We
	// set this dummy value during init so it doesn't clobber the
	// one used by "run test servers".
	if os.Getenv("ARVADOS_API_TOKEN") == "" {
		os.Setenv("ARVADOS_API_TOKEN", "xxx")
	}
}

func main() {
	flag.Parse()
	if os.Getenv("ARVADOS_API_HOST") == "" {
		log.Fatal("ARVADOS_API_HOST environment variable must be set.")
	}
	srv := &server{}
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
	log.Println("Listening at", srv.Addr)
	if err := srv.Wait(); err != nil {
		log.Fatal(err)
	}
}
