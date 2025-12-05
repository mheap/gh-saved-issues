package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mheap/gh-saved-issues/pkg/savedsearches"
)

func main() {
	ctx := context.Background()

	configFlag := flag.String("config", "", "path to config file (default: $XDG_HOME/.github-searches.yaml or $XDG_CONFIG_HOME/.github-searches.yaml)")
	recreate := flag.Bool("recreate", false, "recreate all saved searches (delete existing first)")
	reset := flag.Bool("reset", false, "delete configured saved searches without recreating them")
	flag.Parse()

	configPath, err := savedsearches.ResolveConfigPath(*configFlag)
	if err != nil {
		log.Fatalf("resolve config path: %v", err)
	}

	client, err := savedsearches.NewGraphQLClient(ctx, "")
	if err != nil {
		log.Fatalf("init client: %v", err)
	}

	syncer := savedsearches.NewSyncer(client, *recreate, *reset)
	if err := syncer.Sync(ctx, configPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
