package solitaire

import "math/bits"

// Table de fins de partie.
//
// Chaque coup retire exactement un pion : toute partie passe par une position à K pions, quel que soit le chemin.
// On construit une fois pour toutes l'ensemble EXACT des positions à K pions qui peuvent atteindre la cible, par une
// recherche arrière (UndoMove) à partir des positions finales, couche par couche (1 pion, 2 pions, ...).
// Pendant la recherche, toute position à K pions absente de la table est certainement perdante : on la coupe en O(1),
// ce qui évite d'explorer ses sous-arbres morts. La table est exacte : aucune solution n'est jamais perdue.

// maxEndgame borne le nombre de positions d'une couche (donc K) : la couche suivante n'est construite que si
// elle tient dans cette limite. 2^18 positions (K = 8 sur l'européen) : environ 4 Mo. Mesuré sur l'européen (12 graines) :
// K = 8, 9 et 10 donnent ≈ 17 s, 17 s et 21 s au total (contre ≈ 38 s sans table) ; K = 11 coûte ≈ 63 s car la
// construction (≈ 2 s) dépasse le gain. Au-delà de K = 8 le gain ne progresse plus.
const maxEndgame = 1 << 18

// posSet est un ensemble de positions non nulles : table de hachage à adressage ouvert, sans allocation
// à la consultation. 0 marque une case vide (une position résoluble n'est jamais vide).
type posSet struct {
	slots []Position // taille : puissance de 2, charge <= 1/2
	n     int        // nombre de positions stockées
}

func newPosSet() posSet { return posSet{slots: make([]Position, 1<<10)} }

// index renvoie la case de départ de la sonde pour p (hachage multiplicatif de Fibonacci).
func (s *posSet) index(p Position) int {
	return int((uint64(p) * 0x9E3779B97F4A7C15) >> (64 - bits.Len(uint(len(s.slots)-1))))
}

// has renvoie true si p est dans l'ensemble.
func (s *posSet) has(p Position) bool {
	mask := len(s.slots) - 1
	for i := s.index(p); ; i = (i + 1) & mask {
		switch s.slots[i] {
		case p:
			return true
		case 0:
			return false
		}
	}
}

// add insère p (non nul) dans l'ensemble ; l'ensemble double de taille quand sa charge dépasse 1/2.
func (s *posSet) add(p Position) {
	if 2*(s.n+1) > len(s.slots) {
		s.grow()
	}
	mask := len(s.slots) - 1
	for i := s.index(p); ; i = (i + 1) & mask {
		switch s.slots[i] {
		case p:
			return
		case 0:
			s.slots[i] = p
			s.n++
			return
		}
	}
}

func (s *posSet) grow() {
	old := s.slots
	s.slots = make([]Position, 2*len(old))
	s.n = 0
	for _, p := range old {
		if p != 0 {
			s.add(p)
		}
	}
}

// buildEndgame construit la table de fins de partie (g.endgame, g.endgameK). anyEnd indique si la cible 0
// signifie « une seule bille, n'importe où » (SolveAnyEnd) ; avec endPos == 0 et anyEnd faux, le but est
// inatteignable et on ne construit rien.
//
// Couche de départ : la cible (e pions) si elle est précise ; sinon une bille sur chaque case finale que la règle de
// trois ne rejette pas. On construit ensuite des couches de plus en plus grosses, tant qu'elles ont au plus
// maxEndgame positions et moins de pions que le départ ; la dernière couche conservée est la table.
func (g *Game) buildEndgame(anyEnd bool) {
	g.endgame, g.endgameK = posSet{}, 0
	if g.endPos == 0 && !anyEnd {
		return
	}
	startPegs := bits.OnesCount64(uint64(g.startPos))

	cur := newPosSet()
	k := 1
	if g.endPos != 0 {
		cur.add(g.endPos)
		k = bits.OnesCount64(uint64(g.endPos))
	} else {
		sig := colorParities(g.startPos)
		for p := g.maskPos; p != 0; p &= p - 1 {
			cell := Position(1) << bits.TrailingZeros64(uint64(p))
			if _, ok := ruleOfThreeMismatch(sig, cell, startPegs-1); ok {
				cur.add(cell)
			}
		}
	}

	for k+1 < startPegs { // la table ne sert que si le départ a plus de pions qu'elle
		next := newPosSet()
		overflow := false
		for _, q := range cur.slots {
			if q == 0 {
				continue
			}
			for _, m := range g.validMoves {
				if prev := q.UndoMove(m); prev != 0 {
					next.add(prev)
				}
			}
			if next.n > maxEndgame {
				overflow = true
				break
			}
		}
		if overflow {
			break
		}
		cur, k = next, k+1
	}
	if k+1 >= startPegs && cur.n == 0 {
		return
	}
	g.endgame, g.endgameK = cur, k
}
