package solitaire

type Game struct {
	startPos   Position // Start position
	maskPos    Position // Board limit
	endPos     Position // Target end position
	validMoves []Move   // List of valid moves (sorted)
	solution   []Position
}
