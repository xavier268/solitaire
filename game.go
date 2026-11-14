package solitaire

import (
	"fmt"
	"math/bits"
)

type Game struct {
	startPos   Position // Start position
	maskPos    Position // Board limit
	endPos     Position // Target end position
	validMoves []Move   // List of valid moves (sorted)
	solution   []Position
}

// NewGame crée un jeu à partir du mask du board, de la position de départ et de la position cible.
// Renvoie une erreur si une des positions sort du mask, ou si la cible a plus de pions que le départ
// (chaque move retire exactement un pion).
func NewGame(maskPos Position, startPos Position, endPos Position) (Game, error) {

	if startPos&^maskPos != 0 {
		return Game{}, fmt.Errorf("la position de départ sort du masque (cases hors masque : %#x)", uint64(startPos&^maskPos))
	}
	if endPos&^maskPos != 0 {
		return Game{}, fmt.Errorf("la position d'arrivée sort du masque (cases hors masque : %#x)", uint64(endPos&^maskPos))
	}

	// Chaque move retire un pion : le nombre de coups entre start et end est fixé.
	maxMoves := bits.OnesCount64(uint64(startPos)) - bits.OnesCount64(uint64(endPos))
	if maxMoves < 0 {
		return Game{}, fmt.Errorf("la position d'arrivée a plus de pions (%d) que le départ (%d)",
			bits.OnesCount64(uint64(endPos)), bits.OnesCount64(uint64(startPos)))
	}

	return Game{
		startPos:   startPos,
		maskPos:    maskPos,
		endPos:     endPos,
		validMoves: maskPos.AllMoves(),
		solution:   make([]Position, 0, maxMoves), // vide, capacité = nombre max de coups
	}, nil
}

// Solve recherche une solution pour aller de la position de départ à la position cible
// (depth-first brute force, sans élagage). Erreur si impossible.
// En cas de succès, g.solution contient les positions successives après chaque coup
// (la position de départ n'est pas incluse, la dernière est endPos).
func (g *Game) Solve() error {
	g.solution = g.solution[:0]
	if !g.dfs(g.startPos) {
		return fmt.Errorf("aucune solution entre la position de départ et la position cible")
	}
	return nil
}

// Recursive depth-first exploration, with backtracking.
// Renvoie true dès que la cible est atteinte ; g.solution contient alors le chemin parcouru.
// Pas d'allocation : solution a été créée avec la capacité maximale (un pion retiré par coup).
func (g *Game) dfs(curr Position) bool {

	if curr == g.endPos {
		return true
	}

	for _, m := range g.validMoves {
		next := curr.DoMove(m)
		if next == 0 { // move illégal depuis cette position
			continue
		}
		g.solution = append(g.solution, next)
		if g.dfs(next) {
			return true
		}
		g.solution = g.solution[:len(g.solution)-1] // backtrack
	}
	return false
}
