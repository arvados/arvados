package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"regexp"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/config"
)

// Server configuration
type Config struct {
	Client     arvados.Client
	Listen     string
	GitCommand string
	Root       string
}

var theConfig = defaultConfig()

func defaultConfig() *Config {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln("Getwd():", err)
	}
	return &Config{
		Listen:     ":80",
		GitCommand: "/usr/bin/git",
		Root:       cwd,
	}
}

func init() {
	const defaultCfgPath = "/etc/arvados/arv-git-httpd/config.json"
	const deprecated = " (DEPRECATED -- use config file instead)"
	flag.StringVar(&theConfig.Listen, "address", theConfig.Listen,
		"Address to listen on, \"host:port\" or \":port\"."+deprecated)
	flag.StringVar(&theConfig.GitCommand, "git-command", theConfig.GitCommand,
		"Path to git or gitolite-shell executable. Each authenticated request will execute this program with a single argument, \"http-backend\"."+deprecated)
	flag.StringVar(&theConfig.Root, "repo-root", theConfig.Root,
		"Path to git repositories."+deprecated)

	cfgPath := flag.String("config", defaultCfgPath, "Configuration file `path`.")
	flag.Usage = usage
	flag.Parse()

	err := config.LoadFile(theConfig, *cfgPath)
	if err != nil {
		h := os.Getenv("ARVADOS_API_HOST")
		if h == "" || !os.IsNotExist(err) || *cfgPath != defaultCfgPath {
			log.Fatal(err)
		}
		log.Print("DEPRECATED: No config file found, but ARVADOS_API_HOST environment variable is set. Please use a config file instead.")
		theConfig.Client.APIHost = h
		if regexp.MustCompile("^(?i:1|yes|true)$").MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE")) {
			theConfig.Client.Insecure = true
		}
		if j, err := json.MarshalIndent(theConfig, "", "    "); err == nil {
			log.Print("Current configuration:\n", string(j))
		}
	}

	// MakeArvadosClient returns an error if token is unset (even
	// though we don't need to do anything requiring
	// authentication yet). We can't do this in newArvadosClient()
	// just before calling MakeArvadosClient(), though, because
	// that interferes with the env var needed by "run test
	// servers".
	os.Setenv("ARVADOS_API_TOKEN", "xxx")
}

func main() {
	srv := &server{}
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
	log.Println("Listening at", srv.Addr)
	log.Println("Repository root", theConfig.Root)
	if err := srv.Wait(); err != nil {
		log.Fatal(err)
	}
}
