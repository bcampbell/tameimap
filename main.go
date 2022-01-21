package main

import (
	"fmt"
	"github.com/bcampbell/tameimap/store"
	//	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/server"
	"os"
)

func main() {
	// Create a memory backend
	be, err := store.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERR: %s\n", err)
		os.Exit(1)
	}

	// Create a new server
	s := server.New(be)
	s.Debug = os.Stderr
	s.Addr = ":1143"
	// Since we will use this server for testing only, we can allow plain text
	// authentication over unencrypted connections
	s.AllowInsecureAuth = true

	fmt.Println("Starting IMAP server at localhost:1143")
	if err := s.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "ERR: %s\n", err)
	}
}
