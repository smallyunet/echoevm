package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/smallyunet/echoevm/tests/official/internal/fixturestore"
)

func main() {
	manifestPath := flag.String("manifest", "tests/official/manifest.json", "pinned fixture release manifest")
	destination := flag.String("destination", "tests/official/fixtures", "installed fixture directory")
	archive := flag.String("archive", "tests/official/.cache/fixtures.tar.gz", "download cache path")
	flag.Parse()

	manifest, err := fixturestore.LoadManifest(*manifestPath)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("official fixtures: %s (%d bytes)\n", manifest.Release, manifest.Size)
	client := &http.Client{Timeout: 30 * time.Minute}
	if err := fixturestore.Fetch(context.Background(), client, manifest, *archive, *destination); err != nil {
		fatal(err)
	}
	fmt.Printf("installed and verified: %s\n", *destination)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "fetch official fixtures:", err)
	os.Exit(1)
}
