// Package tictactoe is the tutorial's game after chapter 5: the board module, saved so a game survives a restart.
package tictactoe

import (
	"errors"
	"slices"

	"github.com/ny4rl4th0t3p/playhold"

	"github.com/ny4rl4th0t3p/playhold-tictactoe/chapter5/pb"
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

// MsgPlay is the board's one message, a move: its ID comes from the .proto, so clients share it.
const MsgPlay = uint32(pb.Message_MESSAGE_PLAY)

// Why a move is refused. Only the player who tried it is told.
var (
	ErrWaiting     = errors.New("waiting for a second player")
	ErrGameOver    = errors.New("the game is over")
	ErrNotYourTurn = errors.New("it isn't your turn")
	ErrNoSuchCell  = errors.New("cells go from 0 to 8")
	ErrTaken       = errors.New("that cell is taken")
)

// board is the module's state: each cell's mark, and how the game stands.
type board struct {
	cells  [9]pb.Mark // MARK_UNSPECIFIED is an empty cell
	moves  int        // X moves when it's even, O when it's odd
	winner pb.Mark    // MARK_UNSPECIFIED while nobody has won
	draw   bool
}

var key = playhold.NewKey[board]("board")

// #endregion state

// Board is the module.
type Board struct{}

// Name implements playhold.Module.
func (Board) Name() string { return "board" }

// #region register

// Register implements playhold.Module: the state and how it's saved, the move, and the view.
func (Board) Register(r *playhold.Registry) {
	playhold.State(r, key, func() *board { return &board{} })
	playhold.Snapshot(r, key, 1, save, load)
	playhold.Handle(r, MsgPlay, play)
	playhold.View(r, view)
}

// #endregion register

// #region snapshot

// save turns the board into the message the server saves, at version 1. The cells are copied, as in the view, so the
// message never shares the module's state.
func save(b *board) *pb.BoardState {
	return &pb.BoardState{
		Cells:  slices.Clone(b.cells[:]),
		Moves:  uint32(b.moves), // at most 9
		Winner: b.winner,
		Draw:   b.draw,
	}
}

// load turns a saved board back into the module's state, when the server restores the room.
func load(msg *pb.BoardState) *board {
	b := &board{moves: int(msg.GetMoves()), winner: msg.GetWinner(), draw: msg.GetDraw()}
	copy(b.cells[:], msg.GetCells())
	return b
}

// #endregion snapshot

// #region play

// play marks a cell for the player who sent it, if it's their turn and the cell is free.
func play(rc *playhold.RoomContext, msg *pb.Play) error {
	b := key.Get(rc)
	mark := markOf(rc, rc.Player())
	cell := msg.GetCell()
	switch {
	case len(rc.Players()) < 2:
		return ErrWaiting
	case b.over():
		return ErrGameOver
	case mark != b.toMove():
		return ErrNotYourTurn
	case cell < 0 || cell > 8:
		return ErrNoSuchCell
	case b.cells[cell] != pb.Mark_MARK_UNSPECIFIED:
		return ErrTaken
	}
	b.cells[cell] = mark
	b.moves++
	b.winner = b.lineOf()
	b.draw = b.winner == pb.Mark_MARK_UNSPECIFIED && b.moves == len(b.cells)
	return nil
}

// markOf is a player's mark: X for the first to take a seat, O for the second.
func markOf(rc *playhold.RoomContext, player playhold.PlayerID) pb.Mark {
	switch players := rc.Players(); {
	case len(players) > 0 && players[0] == player:
		return pb.Mark_MARK_X
	case len(players) > 1 && players[1] == player:
		return pb.Mark_MARK_O
	}
	return pb.Mark_MARK_UNSPECIFIED
}

// #endregion play

// toMove is the mark whose turn it is.
func (b *board) toMove() pb.Mark {
	if b.moves%2 == 0 {
		return pb.Mark_MARK_X
	}
	return pb.Mark_MARK_O
}

// over reports whether the game has ended, with a winner or a draw.
func (b *board) over() bool { return b.winner != pb.Mark_MARK_UNSPECIFIED || b.draw }

// lines are the board's rows, columns and diagonals, by cell.
var lines = [][3]int{
	{0, 1, 2}, {3, 4, 5}, {6, 7, 8},
	{0, 3, 6}, {1, 4, 7}, {2, 5, 8},
	{0, 4, 8}, {2, 4, 6},
}

// lineOf is the mark that fills a line, or MARK_UNSPECIFIED if none does.
func (b *board) lineOf() pb.Mark {
	for _, line := range lines {
		if mark := b.cells[line[0]]; mark != pb.Mark_MARK_UNSPECIFIED &&
			mark == b.cells[line[1]] && mark == b.cells[line[2]] {
			return mark
		}
	}
	return pb.Mark_MARK_UNSPECIFIED
}

// #region view

// view shows a player the board, their own mark, and how the game stands for them. Each player gets their own: a win
// for one is a loss for the other. The cells are copied, so the view never shares the module's state.
func view(rc *playhold.RoomContext) *pb.BoardView {
	b := key.Get(rc)
	mark := markOf(rc, rc.Player())
	return &pb.BoardView{Cells: slices.Clone(b.cells[:]), You: mark, Status: status(rc, b, mark)}
}

// status is how the game stands for the player with this mark.
func status(rc *playhold.RoomContext, b *board, mark pb.Mark) pb.Status {
	switch {
	case len(rc.Players()) < 2:
		return pb.Status_STATUS_WAITING
	case b.draw:
		return pb.Status_STATUS_DRAW
	case b.winner == mark:
		return pb.Status_STATUS_WON
	case b.winner != pb.Mark_MARK_UNSPECIFIED:
		return pb.Status_STATUS_LOST
	case b.toMove() == mark:
		return pb.Status_STATUS_YOUR_TURN
	default:
		return pb.Status_STATUS_THEIR_TURN
	}
}

// #endregion view
