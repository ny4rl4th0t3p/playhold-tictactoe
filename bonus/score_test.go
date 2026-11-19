package tictactoe_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ny4rl4th0t3p/playhold/playholdtest"

	tictactoe "github.com/ny4rl4th0t3p/playhold-tictactoe/bonus"
	"github.com/ny4rl4th0t3p/playhold-tictactoe/bonus/pb"
	"github.com/ny4rl4th0t3p/playhold-tictactoe/bonus/score"
)

// scoreOf is the score the player sees.
func scoreOf(p *playholdtest.Player) *pb.ScoreView {
	return playholdtest.View[pb.ScoreView](p, "score")
}

// rematch asks for another game, and fails the test if it's refused.
func rematch(t *testing.T, p *playholdtest.Player) {
	t.Helper()
	require.NoError(t, p.Send("score", score.MsgRematch, &pb.Rematch{}))
}

// winTopRow plays a game X wins on the top row.
func winTopRow(t *testing.T, x, o *playholdtest.Player) {
	t.Helper()
	for i, cell := range []int32{0, 3, 1, 4, 2} {
		play(t, []*playholdtest.Player{x, o}[i%2], cell)
	}
}

// #region test

func TestTheScoreCountsEachGame(t *testing.T) {
	playholdtest.Run(t, func(t *testing.T) {
		alice, bob := newGame(t)
		winTopRow(t, alice, bob)
		assert.Equal(t, uint32(1), scoreOf(alice).GetWins())
		assert.Equal(t, uint32(1), scoreOf(bob).GetLosses(), "each player sees the score from their side")

		rematch(t, bob)

		assert.Equal(t, ".........", cells(view(alice)), "a new game, on an empty board")
		assert.Equal(t, pb.Status_STATUS_YOUR_TURN, view(alice).GetStatus(), "X moves first again")
		winTopRow(t, alice, bob)
		assert.Equal(t, uint32(2), scoreOf(alice).GetWins())
	})
}

// #endregion test

func TestARematchWaitsForTheGameToEnd(t *testing.T) {
	playholdtest.Run(t, func(t *testing.T) {
		alice, bob := newGame(t)
		play(t, alice, 4)

		err := bob.Send("score", score.MsgRematch, &pb.Rematch{})

		require.ErrorIs(t, err, score.ErrGameOn)
		assert.Equal(t, "....X....", cells(view(alice)), "the game goes on")
	})
}

func TestDrawsAndTimeoutsCount(t *testing.T) {
	playholdtest.Run(t, func(t *testing.T) {
		typ, err := tictactoe.NewRoomType()
		require.NoError(t, err)
		room := playholdtest.NewRoom(t, typ, playholdtest.Options{})
		alice, bob := room.Join("alice"), room.Join("bob")
		for i, cell := range []int32{0, 1, 2, 4, 3, 5, 7, 6, 8} { // a draw
			play(t, []*playholdtest.Player{alice, bob}[i%2], cell)
		}
		assert.Equal(t, uint32(1), scoreOf(alice).GetDraws())

		rematch(t, alice)
		room.Advance(tictactoe.TurnTime) // alice doesn't move

		assert.Equal(t, uint32(1), scoreOf(bob).GetWins())
		assert.Equal(t, uint32(1), scoreOf(alice).GetLosses())
	})
}

func TestTheScoreSurvivesARestart(t *testing.T) {
	playholdtest.Run(t, func(t *testing.T) {
		typ, err := tictactoe.NewRoomType()
		require.NoError(t, err)
		room := playholdtest.NewRoom(t, typ, playholdtest.Options{})
		alice, bob := room.Join("alice"), room.Join("bob")
		winTopRow(t, alice, bob)

		room.Restart()

		alice, bob = room.Join("alice"), room.Join("bob")
		assert.Equal(t, uint32(1), scoreOf(alice).GetWins())
		assert.Equal(t, uint32(1), scoreOf(bob).GetLosses())
	})
}

func TestNobodyHasAScoreAlone(t *testing.T) {
	playholdtest.Run(t, func(t *testing.T) {
		typ, err := tictactoe.NewRoomType()
		require.NoError(t, err)
		alice := playholdtest.NewRoom(t, typ, playholdtest.Options{}).Join("alice")

		assert.Nil(t, scoreOf(alice), "no score until the second player sits down")
	})
}
