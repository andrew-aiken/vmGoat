package cmd

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// Destroy handles the destroy command
func Destroy(ctx context.Context, c *cli.Command) error {
	fmt.Println("Destroyed deployment: ", c.Args().First())
	return nil
}
