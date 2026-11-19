// Package tictactoe is the tutorial's game as a host would run it for players: chapter 7's board and score modules, unchanged,
// served by server/ out of development mode.
package tictactoe

import (
	"errors"
	"slices"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/ny4rl4th0t3p/playhold"

	"github.com/ny4rl4th0t3p/playhold-tictactoe/bonus/events"
	"github.com/ny4rl4th0t3p/playhold-tictactoe/bonus/pb"
	"github.com/ny4rl4th0t3p/playhold-tictactoe/bonus/score"
)

// #region room-type

// NewRoomType is the game: rooms of two players, each with a board and a score. The score reaches the board through
// the board's keeper, which is wired in here: the two modules never import each other.
func NewRoomType() (*playhold.RoomType, error) {
	return playhold.NewRoomType(playhold.RoomConfig{
		Name:       "tictactoe",
		MaxPlayers: 2,
		Modules:    []playhold.Module{Board{}, score.Module{Board: Keeper{}}},
	})
}

// #endregion room-type

// #region state

// MsgPlay is the board's one message, a move: its ID comes from the .proto, so clients share it.
const MsgPlay = uint32(pb.Message_MESSAGE_PLAY)

// TurnTime is how long a player has for each move. Running out of it loses the game.
const TurnTime = 30 * time.Second

// Why a move is refused. Only the player who tried it is told.
var (
	ErrWaiting     = errors.New("waiting for a second player")
	ErrGameOver    = errors.New("the game is over")
	ErrNotYourTurn = errors.New("it isn't your turn")
	ErrNoSuchCell  = errors.New("cells go from 0 to 8")
	ErrTaken       = errors.New("that cell is taken")
)

// board is the module's state: each cell's mark, how the game stands, and the turn's timer.
type board struct {
	cells  [9]pb.Mark // MARK_UNSPECIFIED is an empty cell
	moves  int        // X moves when it's even, O when it's odd
	result pb.Result  // RESULT_UNSPECIFIED while the game is on
	timer  playhold.TimerID
}

var key = playhold.NewKey[board]("board")

// #endregion state

// Board is the module.
type Board struct{}

// Name implements playhold.Module.
func (Board) Name() string { return "board" }

// #region register

// Register implements playhold.Module: the state and how it's saved, the move, the turn's timer, and the view.
func (Board) Register(r *playhold.Registry) {
	playhold.State(r, key, func() *board { return &board{} })
	playhold.Snapshot(r, key, 2, save, load)
	playhold.Migrate(r, key, 1, fromVersion1)
	playhold.Handle(r, MsgPlay, play)
	playhold.On(r, secondPlayerJoined)
	playhold.On(r, turnExpired)
	playhold.View(r, view)
}

// #endregion register

// #region snapshot

// save turns the board into the message the server saves, at version 2: the result replaces version 1's winner and
// draw, and the turn's timer is saved too. The cells are copied, as in the view, so the message never shares the
// module's state.
func save(b *board) *pb.BoardState {
	return &pb.BoardState{
		Cells:     slices.Clone(b.cells[:]),
		Moves:     uint32(b.moves), // at most 9
		Result:    b.result,
		TurnTimer: uint64(b.timer),
	}
}

// load turns a saved board back into the module's state, when the server restores the room.
func load(msg *pb.BoardState) *board {
	b := &board{moves: int(msg.GetMoves()), result: msg.GetResult(), timer: playhold.TimerID(msg.GetTurnTimer())}
	copy(b.cells[:], msg.GetCells())
	return b
}

// #endregion snapshot

// #region migration

// fromVersion1 moves a board saved at version 1, which had a winner and a draw, to version 2, which has a result. It
// works on the saved bytes, so it doesn't depend on Go code that has changed since. A game saved at version 1 has no
// timer: its next move starts one.
func fromVersion1(old []byte) ([]byte, error) {
	msg := &pb.BoardState{}
	if err := proto.Unmarshal(old, msg); err != nil {
		return nil, err
	}
	switch {
	case msg.GetWinner() == pb.Mark_MARK_X:
		msg.Result = pb.Result_RESULT_X_WON
	case msg.GetWinner() == pb.Mark_MARK_O:
		msg.Result = pb.Result_RESULT_O_WON
	case msg.GetDraw():
		msg.Result = pb.Result_RESULT_DRAW
	}
	msg.Winner, msg.Draw = pb.Mark_MARK_UNSPECIFIED, false
	return proto.Marshal(msg)
}

// #endregion migration

// #region play

// play marks a cell for the player who sent it, if it's their turn and the cell is free, and starts the next player's
// time, unless the game is over.
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
	switch line := b.lineOf(); {
	case line == pb.Mark_MARK_X:
		end(rc, b, pb.Result_RESULT_X_WON)
	case line == pb.Mark_MARK_O:
		end(rc, b, pb.Result_RESULT_O_WON)
	case b.moves == len(b.cells):
		end(rc, b, pb.Result_RESULT_DRAW)
	}
	rc.CancelTimer(b.timer)
	if !b.over() {
		startTurn(rc, b)
	}
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

// #region timer

// startTurn gives the player to move TurnTime. The timer is kept in the state, so a move can cancel it, and the view
// can show how long is left.
func startTurn(rc *playhold.RoomContext, b *board) {
	b.timer = rc.After(TurnTime, &pb.TurnExpired{})
}

// secondPlayerJoined starts X's time once both seats are taken. PlayerJoined is only for a new seat, so a player coming
// back doesn't restart it.
func secondPlayerJoined(rc *playhold.RoomContext, _ playhold.PlayerJoined) {
	if len(rc.Players()) == 2 {
		startTurn(rc, key.Get(rc))
	}
}

// turnExpired ends the game when the player to move runs out of time: they lose. A move cancels the timer, so it only
// fires on a turn nobody played.
func turnExpired(rc *playhold.RoomContext, _ *pb.TurnExpired) {
	b := key.Get(rc)
	if b.toMove() == pb.Mark_MARK_X {
		end(rc, b, pb.Result_RESULT_X_OUT_OF_TIME)
	} else {
		end(rc, b, pb.Result_RESULT_O_OUT_OF_TIME)
	}
}

// #endregion timer

// #region end

// end ends the game with its result, and tells the other modules who won: the board doesn't know who's listening.
func end(rc *playhold.RoomContext, b *board, result pb.Result) {
	b.result = result
	var winner playhold.PlayerID // empty for a draw
	switch players := rc.Players(); b.winner() {
	case pb.Mark_MARK_X:
		winner = players[0]
	case pb.Mark_MARK_O:
		winner = players[1]
	}
	rc.Emit(events.GameEnded{Winner: winner})
}

// #endregion end

// #region keeper

// Keeper is what the board offers other modules, which reach it only through this.
type Keeper struct{}

// Over reports whether the room's game has ended.
func (Keeper) Over(rc *playhold.RoomContext) bool { return key.Get(rc).over() }

// NewGame clears the board for another game between the same players: X moves first again, and X's time starts.
func (Keeper) NewGame(rc *playhold.RoomContext) {
	b := key.Get(rc)
	rc.CancelTimer(b.timer)
	*b = board{}
	startTurn(rc, b)
}

// #endregion keeper

// toMove is the mark whose turn it is.
func (b *board) toMove() pb.Mark {
	if b.moves%2 == 0 {
		return pb.Mark_MARK_X
	}
	return pb.Mark_MARK_O
}

// over reports whether the game has ended.
func (b *board) over() bool { return b.result != pb.Result_RESULT_UNSPECIFIED }

// winner is the mark that won, or MARK_UNSPECIFIED while the game is on or after a draw.
func (b *board) winner() pb.Mark {
	switch b.result {
	case pb.Result_RESULT_X_WON, pb.Result_RESULT_O_OUT_OF_TIME:
		return pb.Mark_MARK_X
	case pb.Result_RESULT_O_WON, pb.Result_RESULT_X_OUT_OF_TIME:
		return pb.Mark_MARK_O
	default:
		return pb.Mark_MARK_UNSPECIFIED
	}
}

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

// view shows a player the board, their own mark, how the game stands for them, and the time left to move. Each player
// gets their own: a win for one is a loss for the other. The cells are copied, so the view never shares the module's
// state.
func view(rc *playhold.RoomContext) *pb.BoardView {
	b := key.Get(rc)
	mark := markOf(rc, rc.Player())
	v := &pb.BoardView{
		Cells:     slices.Clone(b.cells[:]),
		You:       mark,
		Status:    status(rc, b, mark),
		OutOfTime: b.result == pb.Result_RESULT_X_OUT_OF_TIME || b.result == pb.Result_RESULT_O_OUT_OF_TIME,
	}
	if left, pending := rc.TimerLeft(b.timer); pending && !b.over() {
		v.TurnMillisLeft = left.Milliseconds()
	}
	return v
}

// status is how the game stands for the player with this mark.
func status(rc *playhold.RoomContext, b *board, mark pb.Mark) pb.Status {
	switch {
	case len(rc.Players()) < 2:
		return pb.Status_STATUS_WAITING
	case b.result == pb.Result_RESULT_DRAW:
		return pb.Status_STATUS_DRAW
	case b.winner() == mark:
		return pb.Status_STATUS_WON
	case b.winner() != pb.Mark_MARK_UNSPECIFIED:
		return pb.Status_STATUS_LOST
	case b.toMove() == mark:
		return pb.Status_STATUS_YOUR_TURN
	default:
		return pb.Status_STATUS_THEIR_TURN
	}
}

// #endregion view
