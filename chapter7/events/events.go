// Package events holds the events the game's modules share. A module emits or hears them by importing this package,
// never another module.
package events

import "github.com/ny4rl4th0t3p/playhold"

// #region game-ended

// GameEnded is emitted by the board when a game ends: by a line, a full board or the clock. Winner is the player who
// won, or empty for a draw.
type GameEnded struct {
	Winner playhold.PlayerID
}

// #endregion game-ended
