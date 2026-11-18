package tictactoe_test

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ny4rl4th0t3p/playhold"
	sdk "github.com/ny4rl4th0t3p/playhold-sdk-go"

	tictactoe "github.com/ny4rl4th0t3p/playhold-tictactoe/chapter7"
	"github.com/ny4rl4th0t3p/playhold-tictactoe/chapter7/pb"
)

// These tests run the game on a real server, over real connections, through the Go SDK, as any client would.

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// startServer runs the game as server/ does, on a port of its own, and returns its address.
func startServer(t *testing.T) string {
	t.Helper()
	typ, err := tictactoe.NewRoomType()
	require.NoError(t, err)
	quiet := slog.New(slog.DiscardHandler)
	server, err := playhold.NewServer(playhold.NewRuntime(playhold.Config{Logger: quiet}), &playhold.ServerConfig{
		RoomType:    typ,
		Development: true,
		MemoryLimit: -1, // leave Go's memory limit alone in a test
		Logger:      quiet,
	})
	require.NoError(t, err)
	httpServer := httptest.NewServer(server)
	t.Cleanup(func() {
		server.Close()
		httpServer.Close()
	})
	return "ws" + strings.TrimPrefix(httpServer.URL, "http")
}

// connect says hello as a new guest.
func connect(t *testing.T, url string) *sdk.Client {
	t.Helper()
	c, err := sdk.Connect(testContext(t), &sdk.Options{URL: url})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	return c
}

// waitFor reads the player's views until the board's matches, and returns it.
func waitFor(t *testing.T, c *sdk.Client, match func(*pb.BoardView) bool) *pb.BoardView {
	t.Helper()
	for {
		views, err := c.Next(testContext(t))
		require.NoError(t, err)
		for _, v := range views {
			if v.Module != "board" {
				continue
			}
			board := &pb.BoardView{}
			require.NoError(t, proto.Unmarshal(v.View, board))
			if match(board) {
				return board
			}
		}
	}
}

func status(want pb.Status) func(*pb.BoardView) bool {
	return func(v *pb.BoardView) bool { return v.GetStatus() == want }
}

// #region test

func TestTwoPlayersPlayOverTheNetwork(t *testing.T) {
	url := startServer(t)
	alice, bob := connect(t, url), connect(t, url)

	code, err := alice.CreateRoom(testContext(t)) // a new room, with a code the server makes
	require.NoError(t, err)
	require.NoError(t, bob.Join(testContext(t), code)) // bob joins with the code alice shares
	waitFor(t, alice, status(pb.Status_STATUS_YOUR_TURN))

	_, err = alice.Send(testContext(t), "board", tictactoe.MsgPlay, &pb.Play{Cell: 4})
	require.NoError(t, err)

	board := waitFor(t, bob, status(pb.Status_STATUS_YOUR_TURN))
	assert.Equal(t, pb.Mark_MARK_X, board.GetCells()[4])
	assert.Equal(t, pb.Mark_MARK_O, board.GetYou())
}

// #endregion test

func TestARefusedMoveComesBackAsAnError(t *testing.T) {
	url := startServer(t)
	alice, bob := connect(t, url), connect(t, url)
	code, err := alice.CreateRoom(testContext(t))
	require.NoError(t, err)
	require.NoError(t, bob.Join(testContext(t), code))
	waitFor(t, bob, status(pb.Status_STATUS_THEIR_TURN))

	seq, err := bob.Send(testContext(t), "board", tictactoe.MsgPlay, &pb.Play{Cell: 0}) // it's alice's turn
	require.NoError(t, err)

	var refused *sdk.Error
	for refused == nil { // the next frame for bob is the error, after any views already on their way
		_, err := bob.Next(testContext(t))
		if err != nil {
			require.ErrorAs(t, err, &refused)
		}
	}
	assert.Equal(t, sdk.ErrorRejected, refused.Code)
	assert.Equal(t, seq, refused.Seq, "the error names the move it's about")
	assert.Equal(t, tictactoe.ErrNotYourTurn.Error(), refused.Message)
}
