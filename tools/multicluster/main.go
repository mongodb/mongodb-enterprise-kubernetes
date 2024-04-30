package main

import (
	"context"

	"github.com/10gen/ops-manager-kubernetes/multi/cmd"
)

func main() {
	ctx := context.Background()
	cmd.Execute(ctx)
}
