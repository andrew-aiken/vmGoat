package cmd

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// Purge handles the purge command
func Purge(ctx context.Context, c *cli.Command) error {
	fmt.Println("Destroying all deployed infrastructure: ", c.Args().First())
	return nil
}
