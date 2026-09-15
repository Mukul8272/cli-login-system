package main

import (
	"context"
	"fmt"
	"os"

	"github.com/example/containerized-cli-login/internal/auth"
	"github.com/example/containerized-cli-login/internal/cli"
	"github.com/example/containerized-cli-login/internal/config"
	dbpkg "github.com/example/containerized-cli-login/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	ctx := context.Background()
	database, err := dbpkg.Open(ctx, cfg)
	if err != nil {
		fatal(err)
	}
	defer database.Close()
	if err := dbpkg.RunMigrations(database); err != nil {
		fatal(fmt.Errorf("run migrations: %w", err))
	}
	service := auth.NewService(database, cfg)
	if err := cli.New(service).Run(); err != nil {
		if err.Error() == "exit requested" {
			fmt.Println("Goodbye!")
			return
		}
		fatal(err)
	}
}
func fatal(err error) { fmt.Fprintln(os.Stderr, "Fatal:", err); os.Exit(1) }
