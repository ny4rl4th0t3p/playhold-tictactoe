package tictactoe_test

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ny4rl4th0t3p/playhold/playholdtest"

	tictactoe "github.com/ny4rl4th0t3p/playhold-tictactoe/chapter1"
)

// newGame starts a room of the game for a test, with alice as X and bob as O.
func newGame(t *testing.T) (alice, bob *playholdtest.Player) {
	t.Helper()
	typ, err := tictactoe.NewRoomType()
	require.NoError(t, err)
	room := playholdtest.NewRoom(t, typ, playholdtest.Options{})
	return room.Join("alice"), room.Join("bob")
}

// board is what the player sees.
func board(p *playholdtest.Player) string {
	return playholdtest.View[wrapperspb.StringValue](p, "board").GetValue()
}

// play sends a move, and fails the test if it's refused.
func play(t *testing.T, p *playholdtest.Player, cell int32) {
	t.Helper()
	require.NoError(t, p.Send("board", tictactoe.MsgPlay, wrapperspb.Int32(cell)))
}

// #region test

func TestPlayersTakeTurns(t *testing.T) {
	playholdtest.Run(t, func(t *testing.T) {
		alice, bob := newGame(t)
		assert.Equal(t, "......... your turn", board(alice), "X goes first")
		assert.Equal(t, "......... their turn", board(bob))

		play(t, alice, 4)

		assert.Equal(t, "....X.... their turn", board(alice))
		assert.Equal(t, "....X.... your turn", board(bob), "each player gets their own view")
	})
}

// #endregion test

func TestAWinEndsTheGame(t *testing.T) {
	playholdtest.Run(t, func(t *testing.T) {
		alice, bob := newGame(t)
		play(t, alice, 0)
		play(t, bob, 3)
		play(t, alice, 1)
		play(t, bob, 4)

		play(t, alice, 2) // the top row

		assert.Equal(t, "XXXOO.... you won", board(alice))
		assert.Equal(t, "XXXOO.... you lost", board(bob))
		assert.ErrorIs(t, bob.Send("board", tictactoe.MsgPlay, wrapperspb.Int32(5)), tictactoe.ErrGameOver)
	})
}

func TestAFullBoardWithNoLineIsADraw(t *testing.T) {
	playholdtest.Run(t, func(t *testing.T) {
		alice, bob := newGame(t)
		// X O X
		// X O O
		// O X X
		for i, cell := range []int32{0, 1, 2, 4, 3, 5, 7, 6, 8} {
			play(t, []*playholdtest.Player{alice, bob}[i%2], cell)
		}

		assert.Equal(t, "XOXXOOOXX draw", board(alice))
		assert.Equal(t, "XOXXOOOXX draw", board(bob))
	})
}

func TestRefusedMovesChangeNothing(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, alice, bob *playholdtest.Player)
		who   string
		cell  int32
		want  error
	}{
		{name: "out of turn", who: "bob", cell: 0, want: tictactoe.ErrNotYourTurn},
		{name: "off the board", who: "alice", cell: 9, want: tictactoe.ErrNoSuchCell},
		{name: "negative", who: "alice", cell: -1, want: tictactoe.ErrNoSuchCell},
		{
			name:  "a taken cell",
			setup: func(t *testing.T, alice, _ *playholdtest.Player) { play(t, alice, 4) },
			who:   "bob", cell: 4, want: tictactoe.ErrTaken,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			playholdtest.Run(t, func(t *testing.T) {
				alice, bob := newGame(t)
				if tt.setup != nil {
					tt.setup(t, alice, bob)
				}
				before := board(alice)
				player := map[string]*playholdtest.Player{"alice": alice, "bob": bob}[tt.who]

				err := player.Send("board", tictactoe.MsgPlay, wrapperspb.Int32(tt.cell))

				require.ErrorIs(t, err, tt.want)
				assert.Equal(t, before, board(alice))
			})
		})
	}
}

func TestNobodyPlaysAlone(t *testing.T) {
	playholdtest.Run(t, func(t *testing.T) {
		typ, err := tictactoe.NewRoomType()
		require.NoError(t, err)
		alice := playholdtest.NewRoom(t, typ, playholdtest.Options{}).Join("alice")
		assert.Equal(t, "......... waiting", board(alice))

		err = alice.Send("board", tictactoe.MsgPlay, wrapperspb.Int32(4))

		require.ErrorIs(t, err, tictactoe.ErrWaiting)
	})
}
