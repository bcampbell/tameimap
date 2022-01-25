package main

import (
	"flag"
	"fmt"
	"github.com/bcampbell/tameimap/store"
	"github.com/emersion/go-imap/server"
	"os"
)

var opts struct {
	verbose bool
	dump    string
	port    int
}

func usage() {

	fmt.Fprintf(os.Stderr, `Usage: %s [OPTIONS] [FILE] ...
A simple IMAP server.
Options:
`, os.Args[0])

	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.StringVar(&opts.dump, "d", "", "dump all network traffic to file (\"-\" = stdout)")
	flag.BoolVar(&opts.verbose, "v", false, "output extra info to stderr")
	flag.IntVar(&opts.port, "p", 1143, "port number")
	flag.Parse()

	// Create our backend (and a test user)
	be := store.New()
	u, err := store.NewUser("./bob", "bob", "pass")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERR: %s\n", err)
		os.Exit(1)
	}
	be.Users[u.Username()] = u

	// Create a new server
	s := server.New(be)
	if opts.dump == "-" {
		s.Debug = os.Stdout
	} else if opts.dump != "" {
		f, err := os.Create(opts.dump)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERR: %s\n", err)
			os.Exit(1)
		}
		s.Debug = f
		defer f.Close()
	}

	s.Addr = fmt.Sprintf(":%d", opts.port)
	// Since we will use this server for testing only, we can allow plain text
	// authentication over unencrypted connections
	s.AllowInsecureAuth = true

	if opts.verbose {
		fmt.Fprintf(os.Stderr, "Starting IMAP server at localhost%s\n", s.Addr)
	}
	err = s.ListenAndServe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERR: %s\n", err)
	}
}
