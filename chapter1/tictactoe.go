// Package tictactoe is the tutorial's game after chapter 1: one module, the board, with a move and a view.
package tictactoe

import (
	"errors"
	"strings"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ny4rl4th0t3p/playhold"
)

// #region room-type

// NewRoomType is the game: rooms of two players, each with a board.
func NewRoomType() (*playhold.RoomType, error) {
	return playhold.NewRoomType(playhold.RoomConfig{
		Name:       "tictactoe",
		MaxPlayers: 2,
		Modules:    []playhold.Module{Board{}},
	})
}

// #endregion room-type

// #region state

// MsgPlay is the board's one message: a move, the cell to mark, from 0 (top left) to 8 (bottom right).
const MsgPlay uint32 = 1

// Why a move is refused. Only the player who tried it is told.
var (
	ErrWaiting     = errors.New("waiting for a second player")
	ErrGameOver    = errors.New("the game is over")
	ErrNotYourTurn = errors.New("it isn't your turn")
	ErrNoSuchCell  = errors.New("cells go from 0 to 8")
	ErrTaken       = errors.New("that cell is taken")
)

// board is the module's state: each cell's mark, "X", "O" or empty, and how the game stands.
type board struct {
	cells  [9]string
	moves  int    // X moves when it's even, O when it's odd
	winner string // "X", "O", "draw", or empty while the game goes on
}

var key = playhold.NewKey[board]("board")

// #endregion state

// Board is the module.
type Board struct{}

// Name implements playhold.Module.
func (Board) Name() string { return "board" }

// #region register

// Register implements playhold.Module: the state, the move, and the view.
func (Board) Register(r *playhold.Registry) {
	playhold.State(r, key, func() *board { return &board{} })
	playhold.Handle(r, MsgPlay, play)
	playhold.View(r, view)
}

// #endregion register

// #region play

// play marks a cell for the player who sent it, if it's their turn and the cell is free.
func play(rc *playhold.RoomContext, msg *wrapperspb.Int32Value) error {
	b := key.Get(rc)
	mark := markOf(rc, rc.Player())
	cell := msg.GetValue()
	switch {
	case len(rc.Players()) < 2:
		return ErrWaiting
	case b.winner != "":
		return ErrGameOver
	case mark != b.toMove():
		return ErrNotYourTurn
	case cell < 0 || cell > 8:
		return ErrNoSuchCell
	case b.cells[cell] != "":
		return ErrTaken
	}
	b.cells[cell] = mark
	b.moves++
	b.winner = b.result()
	return nil
}

// markOf is a player's mark: X for the first to take a seat, O for the second.
func markOf(rc *playhold.RoomContext, player playhold.PlayerID) string {
	switch players := rc.Players(); {
	case len(players) > 0 && players[0] == player:
		return "X"
	case len(players) > 1 && players[1] == player:
		return "O"
	}
	return ""
}

// #endregion play

// toMove is the mark whose turn it is.
func (b *board) toMove() string {
	if b.moves%2 == 0 {
		return "X"
	}
	return "O"
}

// lines are the board's rows, columns and diagonals, by cell.
var lines = [][3]int{
	{0, 1, 2}, {3, 4, 5}, {6, 7, 8},
	{0, 3, 6}, {1, 4, 7}, {2, 5, 8},
	{0, 4, 8}, {2, 4, 6},
}

// result is the winner's mark, "draw" once the board is full, or empty while the game goes on.
func (b *board) result() string {
	for _, line := range lines {
		if mark := b.cells[line[0]]; mark != "" && mark == b.cells[line[1]] && mark == b.cells[line[2]] {
			return mark
		}
	}
	if b.moves == len(b.cells) {
		return "draw"
	}
	return ""
}

// #region view

// view shows a player the board, a "." for each empty cell, and how the game stands for them, such as
// "X.O.X.... your turn". Each player gets their own: "you won" for one is "you lost" for the other.
func view(rc *playhold.RoomContext) *wrapperspb.StringValue {
	b := key.Get(rc)
	var cells strings.Builder
	for _, mark := range b.cells {
		if mark == "" {
			mark = "."
		}
		cells.WriteString(mark)
	}
	return wrapperspb.String(cells.String() + " " + status(rc, b))
}

// status is how the game stands for the player the view is for.
func status(rc *playhold.RoomContext, b *board) string {
	mark := markOf(rc, rc.Player())
	switch {
	case len(rc.Players()) < 2:
		return "waiting"
	case b.winner == "draw":
		return "draw"
	case b.winner == mark:
		return "you won"
	case b.winner != "":
		return "you lost"
	case b.toMove() == mark:
		return "your turn"
	default:
		return "their turn"
	}
}

// #endregion view
