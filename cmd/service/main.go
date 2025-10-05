package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/e-wrobel/sync-service/internal/sync"
	"github.com/e-wrobel/sync-service/internal/validators"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	var src string
	var dst string
	var deleteMissing bool

	flag.StringVar(&src, "source", "", "Path to source folder")
	flag.StringVar(&dst, "target", "", "Path to target folder")
	flag.BoolVar(&deleteMissing, "delete-missing", false, "Remove files missing in source folder")
	flag.Parse()

	if src == "" || dst == "" {
		fmt.Fprintln(os.Stderr, "Usage: sync --source <dir> --target <dir> [--delete-missing]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	if err := validators.MustDir(src); err != nil {
		log.Fatalf("source error: %v", err)
	}
	if err := validators.MustDir(dst); err != nil {
		log.Fatalf("target error: %v", err)
	}

	rep := sync.Sync(sync.Options{
		Source:        src,
		Target:        dst,
		DeleteMissing: deleteMissing,
		Logger:        log.Default(),
	})

	log.Printf("DONE – copied=%d overwritten=%d deleted=%d skipped=%d errors=%d",
		rep.Copied, rep.Overwritten, rep.Deleted, rep.Skipped, len(rep.Errors))

	if len(rep.Errors) > 0 {
		log.Println("Encountered errors:")
		for _, e := range rep.Errors {
			log.Printf("  - %v", e)
		}
		os.Exit(1)
	}
}
