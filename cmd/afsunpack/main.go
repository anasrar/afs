package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	flag.StringVar(&afsPath, "afspath", "", "Path to AFS file")
}

func main() {
	flag.Parse()

	if afsPath != "" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		if err := unpack(
			ctx,
			afsPath,
			func(total, current uint32, name string) {
				log.Printf("% 8d/%d(%s): start\n", current, total, name)
			},
			func(total, current uint32, name string) {
				log.Printf("% 8d/%d(%s): done\n", current, total, name)
			},
		); err != nil {
			log.Fatalln(err)
		}
	} else {
		gui()
	}
}
