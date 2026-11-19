// Command server serves the tutorial's game as players would reach it: out of development mode, behind a proxy that ends
// TLS, saving games in PostgreSQL. compose.prod.yaml runs it that way. DEVELOPMENT=1 turns development mode on, for
// `go run ./server` on your own machine as in the chapters.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ny4rl4th0t3p/playhold"
	"github.com/ny4rl4th0t3p/playhold/store/postgres"

	tictactoe "github.com/ny4rl4th0t3p/playhold-tictactoe/bonus"
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
	proxies, err := trustedProxies(os.Getenv("TRUSTED_PROXIES"))
	if err != nil {
		return err
	}
	// Development mode is off unless asked for: forgetting a setting never leaves a public server unprotected.
	development := os.Getenv("DEVELOPMENT") == "1"
	cfg := &playhold.ServerConfig{
		RoomType:       typ,
		Development:    development,
		OriginPatterns: origins(development),
		// Out of development mode, the server only takes plain HTTP from these addresses: the proxy that ends TLS.
		TrustedProxies: proxies,
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

// origins are the addresses of the pages allowed to connect, from ORIGINS, separated by commas: the page's domain, such
// as "page.example". In development mode, it defaults to Vite's development server on this machine; out of it, to
// none, so only a page served from the server's own host could connect, and this one's isn't.
func origins(development bool) []string {
	if list := os.Getenv("ORIGINS"); list != "" {
		return strings.Split(list, ",")
	}
	if development {
		return []string{"localhost:5173"}
	}
	return nil
}

// trustedProxies reads TRUSTED_PROXIES: address ranges separated by commas, such as "172.30.0.2/32". Empty trusts no
// proxy. A wrong range stops the server, rather than leaving it refusing every player.
func trustedProxies(list string) ([]netip.Prefix, error) {
	if list == "" {
		return nil, nil
	}
	var proxies []netip.Prefix
	for item := range strings.SplitSeq(list, ",") {
		prefix, err := netip.ParsePrefix(item)
		if err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXIES: %w", err)
		}
		proxies = append(proxies, prefix)
	}
	return proxies, nil
}

// #endregion server
