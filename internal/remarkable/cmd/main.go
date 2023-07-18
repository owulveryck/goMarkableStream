package main

import (
	"context"
	"fmt"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

func main() {
	es := remarkable.NewEventScanner()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	es.Start(ctx)

	for range es.EventC {
	}
	fmt.Println("done")
	time.Sleep(2 * time.Second)
}
