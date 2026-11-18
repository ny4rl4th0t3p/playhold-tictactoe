// Command server serves the tutorial's game after chapter 4 on this machine, at ws://localhost:8080/ws, until Ctrl+C.
// The page runs on its own, on Vite's development server, at http://localhost:5173.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ny4rl4th0t3p/playhold"

	tictactoe "github.com/ny4rl4th0t3p/playhold-tictactoe/chapter4"
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
		RoomType:       typ,
		Development:    true,
		OriginPatterns: origins(),
	})
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return server.ListenAndServe(ctx, &playhold.ListenConfig{})
}

// origins are the addresses of the pages allowed to connect: the page comes from another address than the server's, and
// browsers say which page opened a connection, so the server refuses pages it doesn't know. ORIGINS lists them, separated
// by commas, such as "192.168.1.20:5173" for Vite on another machine, or the page's domain on a public server; by
// default, Vite's development server on this machine.
func origins() []string {
	if list := os.Getenv("ORIGINS"); list != "" {
		return strings.Split(list, ",")
	}
	return []string{"localhost:5173"}
}

// #endregion server
