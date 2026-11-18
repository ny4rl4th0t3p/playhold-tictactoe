// Package score is the score module: the wins and draws of the games played in a room, and a rematch once a game is
// over. It never imports the board: it reaches it through the Board keeper, and hears results through events.
package score

import (
	"errors"
	"slices"

	"github.com/ny4rl4th0t3p/playhold"

	"github.com/ny4rl4th0t3p/playhold-tictactoe/chapter7/events"
	"github.com/ny4rl4th0t3p/playhold-tictactoe/chapter7/pb"
)

// #region keeper

// Board is what the score needs from the board module. The room type wires in the board's keeper.
type Board interface {
	// Over reports whether the room's game has ended.
	Over(rc *playhold.RoomContext) bool
	// NewGame starts another game between the same players.
	NewGame(rc *playhold.RoomContext)
}

// #endregion keeper

// MsgRematch is the score's one message: another game, once this one is over.
const MsgRematch = uint32(pb.ScoreMessage_SCORE_MESSAGE_REMATCH)

// ErrGameOn refuses a rematch while a game is still being played.
var ErrGameOn = errors.New("finish this game first")

// state is the module's state: the wins by seat, in the order the players sat down, and the draws.
type state struct {
	wins  [2]int
	draws int
}

var key = playhold.NewKey[state]("score")

// #region module

// Module is the score module. Board is the board's keeper.
type Module struct {
	Board Board
}

// Name implements playhold.Module.
func (Module) Name() string { return "score" }

// Register implements playhold.Module. Needs fails the room type if the board's keeper wasn't wired in, or the board
// isn't one of its modules.
func (m Module) Register(r *playhold.Registry) {
	r.Needs("board", m.Board)
	playhold.State(r, key, func() *state { return &state{} })
	playhold.Snapshot(r, key, 1, save, load)
	playhold.Handle(r, MsgRematch, m.rematch)
	playhold.On(r, gameEnded)
	playhold.View(r, view)
}

// #endregion module

// #region rematch

// rematch asks the board for another game, through its keeper, once this one is over.
func (m Module) rematch(rc *playhold.RoomContext, _ *pb.Rematch) error {
	if !m.Board.Over(rc) {
		return ErrGameOn
	}
	m.Board.NewGame(rc)
	return nil
}

// #endregion rematch

// #region game-ended

// gameEnded counts the board's result: a win for the winner's seat, or a draw.
func gameEnded(rc *playhold.RoomContext, event events.GameEnded) {
	s := key.Get(rc)
	if seat := slices.Index(rc.Players(), event.Winner); seat >= 0 && seat < len(s.wins) {
		s.wins[seat]++
	} else {
		s.draws++
	}
}

// #endregion game-ended

// #region view

// view shows a player their wins, the other player's as their losses, and the draws, once both have sat down.
func view(rc *playhold.RoomContext) *pb.ScoreView {
	s := key.Get(rc)
	players := rc.Players()
	seat := slices.Index(players, rc.Player())
	if len(players) < 2 || seat < 0 || seat >= len(s.wins) {
		return nil // nothing to show yet
	}
	return &pb.ScoreView{
		Wins:   uint32(s.wins[seat]),   // games in one room: far below uint32's range
		Losses: uint32(s.wins[1-seat]), // the other seat
		Draws:  uint32(s.draws),
	}
}

// #endregion view

// #region snapshot

func save(s *state) *pb.ScoreState {
	return &pb.ScoreState{Wins: []uint32{uint32(s.wins[0]), uint32(s.wins[1])}, Draws: uint32(s.draws)}
}

func load(msg *pb.ScoreState) *state {
	s := &state{draws: int(msg.GetDraws())}
	for seat, wins := range msg.GetWins() {
		if seat < len(s.wins) {
			s.wins[seat] = int(wins)
		}
	}
	return s
}

// #endregion snapshot
