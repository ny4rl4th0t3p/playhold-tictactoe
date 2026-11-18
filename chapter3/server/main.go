// Command server serves the tutorial's game after chapter 3 on this machine, at ws://localhost:8080/ws, until Ctrl+C.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ny4rl4th0t3p/playhold"

	tictactoe "github.com/ny4rl4th0t3p/playhold-tictactoe/chapter3"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// #region server

// run serves the game until Ctrl+C or SIGTERM, then stops without losing a game.
func run() error {
	typ, err := tictactoe.NewRoomType()
	if err != nil {
		return err
	}
	// Development mode serves plain HTTP, for your own machine only: a public server needs TLS, see Run in production.
	server, err := playhold.NewServer(playhold.NewRuntime(playhold.Config{}), &playhold.ServerConfig{
		RoomType:    typ,
		Development: true,
	})
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return server.ListenAndServe(ctx, &playhold.ListenConfig{})
}

// #endregion server
