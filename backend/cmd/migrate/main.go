// Command migrate manages the database schema outside the API process.
//
//	migrate up            apply pending migrations (what MIGRATE_ON_START does)
//	migrate fresh -yes    DROP the public schema, then apply every migration
package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/config"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/migrate"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	cfg, err := config.Load()
	if err != nil {
		fail(err)
	}

	switch os.Args[1] {
	case "up":
		if err := migrate.Up(cfg.DatabaseURL); err != nil {
			fail(err)
		}
		fmt.Println("migrations applied")
	case "fresh":
		fs := flag.NewFlagSet("fresh", flag.ExitOnError)
		yes := fs.Bool("yes", false, "confirm that ALL data in the target database will be destroyed")
		_ = fs.Parse(os.Args[2:])

		target := describe(cfg.DatabaseURL)
		if !*yes {
			fmt.Fprintf(os.Stderr, "refusing to wipe %s without -yes\n", target)
			os.Exit(2)
		}
		fmt.Printf("wiping %s and re-applying all migrations\n", target)
		if err := migrate.Fresh(cfg.DatabaseURL); err != nil {
			fail(err)
		}
		fmt.Println("done")
	default:
		usage()
	}
}

// describe names the target without leaking credentials.
func describe(databaseURL string) string {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "the configured database"
	}

	return u.Host + "/" + strings.TrimPrefix(u.Path, "/")
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: migrate up | migrate fresh -yes")
	os.Exit(2)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
