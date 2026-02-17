// Package main is the entry point for the iDRAC6 Manager server.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/williamzujkowski/idrac6-manager/internal/api"
	"github.com/williamzujkowski/idrac6-manager/web"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	host := flag.String("host", "", "iDRAC host (ip:port or ip)")
	user := flag.String("user", "root", "iDRAC username")
	pass := flag.String("pass", "", "iDRAC password")
	apiKey := flag.String("api-key", "", "optional API key for authentication")
	hostID := flag.String("host-id", "default", "host identifier")
	hostName := flag.String("host-name", "", "display name for the host")
	flag.Parse()

	if *host == "" {
		*host = os.Getenv("IDRAC_HOST")
	}
	if *host == "" {
		fmt.Fprintln(os.Stderr, "Error: --host or IDRAC_HOST is required")
		flag.Usage()
		os.Exit(1)
	}

	if envUser := os.Getenv("IDRAC_USER"); envUser != "" {
		*user = envUser
	}
	if envPass := os.Getenv("IDRAC_PASS"); envPass != "" {
		*pass = envPass
	}
	if *pass == "" {
		fmt.Fprintln(os.Stderr, "Error: --pass or IDRAC_PASS is required")
		flag.Usage()
		os.Exit(1)
	}
	if envKey := os.Getenv("IDRAC_API_KEY"); envKey != "" {
		*apiKey = envKey
	}

	displayName := *hostName
	if displayName == "" {
		displayName = *host
	}

	cfg := &api.Config{
		Hosts: map[string]*api.HostConfig{
			*hostID: {
				Name:     displayName,
				Host:     *host,
				Username: *user,
				Password: *pass,
			},
		},
		WebFS:  web.FS(),
		APIKey: *apiKey,
	}

	router := api.NewRouter(cfg)

	log.Printf("iDRAC6 Manager starting on %s", *addr)
	log.Printf("Managing host: %s (%s)", displayName, *host)
	if *apiKey != "" {
		log.Printf("API key authentication enabled")
	}
	log.Printf("Web UI: http://localhost%s", *addr)

	if err := http.ListenAndServe(*addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
