package solitaire

import (
	"fmt"
	"math/bits"
)

type Move struct {
	from  Point    // case de départ
	to    Point    // case d'arrivée
	mask3 Position // les 3 cases adjacentes alignées concernées
	mask2 Position // les 2 cases qui doivent être à 1 (la 3ème etant à zero).
}

func (m Move) String() string {
	return fmt.Sprintf("%s->%s", m.from, m.to)
}

// Applique le move m à la position
// si move valide, renvoie la nouvelle position
// si move invalide, renvoie 0 (position impossible a obtenir de façon naturelle)
func (p Position) DoMove(m Move) (pp Position) {

	// Vérifie que le move est légal. Si illegal, renvoie 0.
	// Sur les 3 cases alignées, exactement mask2 doit être occupé :
	// départ et case sautée occupées, arrivée libre.
	if p&m.mask3 != m.mask2 {
		return 0
	}

	// Effectue le move, renvoie la nouvelle position :
	// départ et case sautée se vident (1 -> 0), l'arrivée se remplit (0 -> 1).
	return p ^ m.mask3
}

// Renvoie la liste de tous les moves légaux à l'interieur du mask donné.
// Un move saute par-dessus une case adjacente (haut, bas, gauche, droite) :
// les 3 cases (départ, saut, arrivée) doivent toutes appartenir au mask.
// Chaque sens est un move distinct.
func (mask Position) AllMoves() []Move {
	// Borne haute : au plus 4 moves (un par direction) au départ de chaque case du mask.
	// Une seule allocation, sans réallocation par append.
	moves := make([]Move, 0, 4*bits.OnesCount64(uint64(mask)))
	dirs := [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} // (dl, dc)

	// inside indique si (l, c) est sur la grille 8x8 et dans le mask.
	// Les coordonnées sont des int pendant le calcul pour éviter le wrap-around
	// des uint8 avec les déplacements négatifs.
	inside := func(l, c int) bool {
		return l >= 0 && l < 8 && c >= 0 && c < 8 &&
			mask&Bit(uint8(l), uint8(c)) != 0
	}

	for l := range 8 {
		for c := range 8 {
			if !inside(l, c) {
				continue
			}
			for _, d := range dirs {
				ml, mc := l+d[0], c+d[1]     // case sautée
				tl, tc := l+2*d[0], c+2*d[1] // arrivée
				if !inside(ml, mc) || !inside(tl, tc) {
					continue
				}
				from := Point{uint8(l), uint8(c)}
				to := Point{uint8(tl), uint8(tc)}
				mid := Point{uint8(ml), uint8(mc)}
				moves = append(moves, Move{
					from:  from,
					to:    to,
					mask3: from.Bit() | mid.Bit() | to.Bit(),
					mask2: from.Bit() | mid.Bit(),
				})
			}
		}
	}
	return moves
}
