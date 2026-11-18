package tictactoe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/ny4rl4th0t3p/playhold-tictactoe/chapter6/pb"
)

// #region migration-test

// TestBoardsSavedAtVersion1ComeBackAtVersion2 migrates boards as chapter 5's save wrote them, at version 1, with a winner
// and a draw, then loads them as the server would.
func TestBoardsSavedAtVersion1ComeBackAtVersion2(t *testing.T) {
	x, o, empty := pb.Mark_MARK_X, pb.Mark_MARK_O, pb.Mark_MARK_UNSPECIFIED
	tests := []struct {
		name  string
		saved *pb.BoardState
		want  pb.Result
	}{
		{
			name:  "in progress",
			saved: &pb.BoardState{Cells: []pb.Mark{x, o, empty, empty, x, empty, empty, empty, empty}, Moves: 3},
			want:  pb.Result_RESULT_UNSPECIFIED,
		},
		{
			name:  "X won",
			saved: &pb.BoardState{Cells: []pb.Mark{x, x, x, o, o, empty, empty, empty, empty}, Moves: 5, Winner: x},
			want:  pb.Result_RESULT_X_WON,
		},
		{
			name:  "O won",
			saved: &pb.BoardState{Cells: []pb.Mark{x, x, empty, o, o, o, x, empty, empty}, Moves: 6, Winner: o},
			want:  pb.Result_RESULT_O_WON,
		},
		{
			name:  "a draw",
			saved: &pb.BoardState{Cells: []pb.Mark{x, o, x, x, o, o, o, x, x}, Moves: 9, Draw: true},
			want:  pb.Result_RESULT_DRAW,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old, err := proto.Marshal(tt.saved)
			require.NoError(t, err)

			migrated, err := fromVersion1(old)
			require.NoError(t, err)

			msg := &pb.BoardState{}
			require.NoError(t, proto.Unmarshal(migrated, msg))
			assert.Equal(t, pb.Mark_MARK_UNSPECIFIED, msg.GetWinner(), "version 2 doesn't keep version 1's fields")
			assert.False(t, msg.GetDraw())
			b := load(msg)
			assert.Equal(t, tt.want, b.result)
			assert.Equal(t, tt.saved.GetCells(), b.cells[:], "the same board")
			assert.Equal(t, int(tt.saved.GetMoves()), b.moves)
		})
	}
}

// #endregion migration-test
