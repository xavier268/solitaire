package solitaire

import "math/bits"

// Ordre des coups guidé.
//
// À chaque nœud, au lieu d'essayer les coups dans l'ordre (fixe ou mélangé), on construit les positions enfants, on les
// note avec une évaluation bon marché (opérations sur les bits) et on essaie les meilleures d'abord. L'ordre de départ
// (mélangé à chaque tentative) départage les égalités : le tri est stable, donc les redémarrages restent différents.

const (
	maxGuided = 40 // nombre maximal d'enfants triés par nœud ; au-delà, le nœud retombe sur l'ordre non guidé

	colFirst = 0x0101010101010101 // colonne 0 de chaque ligne
	colLast  = 0x8080808080808080 // colonne 7 de chaque ligne
	colGE2   = 0xFCFCFCFCFCFCFCFC // colonnes 2 à 7 : arrivées possibles d'un saut vers l'est
	colLE5   = 0x3F3F3F3F3F3F3F3F // colonnes 0 à 5 : arrivées possibles d'un saut vers l'ouest
)

// cand est un enfant d'un nœud : la position obtenue, la variation de pagode du coup et sa note.
type cand struct {
	next  Position
	dPag  int32
	score int32
}

// isoPenalty : pénalité d'un pion isolé dans l'évaluation (en nombre de coups possibles). Mesuré : de 0 à 16, le
// résultat varie peu ; la mobilité est ce qui compte.
const isoPenalty = 2

// isolated renvoie le nombre de pions sans aucun pion voisin (haut, bas, gauche, droite).
func isolated(p Position) int {
	nb := (p&^colLast)<<1 | (p&^colFirst)>>1 | p<<8 | p>>8
	return bits.OnesCount64(uint64(p &^ nb))
}

// mobility renvoie le nombre de coups légaux depuis p dans le mask (un pion en saute un autre vers une case vide).
func mobility(p, mask Position) int {
	empty := uint64(mask &^ p)
	x := uint64(p)
	return bits.OnesCount64((x<<1)&(x<<2)&colGE2&empty) +
		bits.OnesCount64((x>>1)&(x>>2)&colLE5&empty) +
		bits.OnesCount64((x<<8)&(x<<16)&empty) +
		bits.OnesCount64((x>>8)&(x>>16)&empty)
}

// guideScore note une position : plus la note est grande, plus on l'essaie tôt.
// Note = mobilité (nombre de coups légaux ensuite) - isoPenalty * pions isolés. On préfère les positions qui laissent
// beaucoup de choix et peu de pions perdus. L'inverse (Warnsdorff : peu de coups d'abord) ne termine pas sur l'européen.
func (g *Game) guideScore(p Position) int32 {
	return int32(mobility(p, g.maskPos) - isoPenalty*isolated(p))
}

// sortedChildren remplit buf avec les enfants légaux de curr triés par note décroissante (tri par insertion stable,
// sans allocation). Renvoie leur nombre, ou -1 si buf est trop petit (le nœud est alors exploré sans guidage).
func (g *Game) sortedChildren(curr Position, buf []cand) int {
	n := 0
	for _, m := range g.moves {
		next := curr.DoMove(m)
		if next == 0 { // move illégal depuis cette position
			continue
		}
		if n == len(buf) {
			return -1
		}
		s := g.guideScore(next)
		j := n
		for j > 0 && buf[j-1].score < s {
			buf[j] = buf[j-1]
			j--
		}
		buf[j] = cand{next, m.dPag, s}
		n++
	}
	return n
}

// scratchFor renvoie la zone de travail du nœud de profondeur depth (allouée une seule fois, à la demande).
func (g *Game) scratchFor(depth int) []cand {
	if g.scratch == nil {
		g.scratch = make([]cand, (bits.OnesCount64(uint64(g.startPos))+1)*maxGuided)
	}
	return g.scratch[depth*maxGuided : (depth+1)*maxGuided]
}
