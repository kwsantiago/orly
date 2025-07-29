package main

import (
	"orly.dev/cmd/lerproxy/app"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"os"
	"os/signal"

	"github.com/alexflint/go-arg"
)

var args app.RunArgs

func main() {
	arg.MustParse(&args)
	ctx, cancel := signal.NotifyContext(context.Bg(), os.Interrupt)
	defer cancel()
	if err := app.Run(ctx, args); chk.T(err) {
		log.F.Ln(err)
	}
}
