// Command server serves the tutorial's game after chapter 7 on this machine, at ws://localhost:8080/ws, until Ctrl+C.
// With DATABASE_URL set, it saves games in PostgreSQL, so they survive it stopping. The page runs on its own, on Vite's
// development server, at http://localhost:5173.
package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ny4rl4th0t3p/playhold"
	"github.com/ny4rl4th0t3p/playhold/store/postgres"

	tictactoe "github.com/ny4rl4th0t3p/playhold-tictactoe/chapter7"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// #region server

// run serves the game until Ctrl+C or SIGTERM, then saves every game and stops.
func run() error {
	typ, err := tictactoe.NewRoomType()
	if err != nil {
		return err
	}
	// Development mode serves plain HTTP, for your own machine only: a public server needs TLS, see Run in production.
	cfg := &playhold.ServerConfig{
		RoomType:       typ,
		Development:    true,
		OriginPatterns: origins(),
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// Without a store, games live only in memory, and a restart loses them.
	if url := os.Getenv("DATABASE_URL"); url != "" {
		records, err := postgres.Open(ctx, url)
		if err != nil {
			return err
		}
		defer records.Close() // after ListenAndServe has saved every game
		cfg.Store = records
		slog.Info("saving games in PostgreSQL")
	} else {
		slog.Warn("no DATABASE_URL: games live in memory only, and a restart loses them")
	}
	server, err := playhold.NewServer(playhold.NewRuntime(playhold.Config{}), cfg)
	if err != nil {
		return err
	}
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
