package solitaire

import "math/bits"

// Énumération des coups légaux par opérations sur les bits, et ordre des coups guidé.
//
// Au lieu de tester un à un les ~100 coups du plateau avec DoMove, on calcule d'un coup, pour chacune des 4 directions,
// l'ensemble des cases d'arrivée possibles : une case vide dont les deux cases précédentes dans la direction sont des
// pions. Il ne reste qu'à parcourir les bits de cet ensemble, et une table (Game.moveIdx) donne le coup à partir de
// (direction, case d'arrivée). Le coût par nœud ne dépend plus du nombre total de coups du plateau.
//
// Ordre guidé : à chaque nœud d'une tentative aléatoire, on construit les positions enfants, on les note avec une
// évaluation bon marché et on essaie les meilleures d'abord. Un rang aléatoire par coup (tiré à chaque tentative)
// départage les égalités : les redémarrages restent différents.

const (
	maxGuided = 40 // nombre maximal d'enfants triés par nœud ; au-delà, le nœud retombe sur l'ordre non guidé

	colFirst = 0x0101010101010101 // colonne 0 de chaque ligne
	colLast  = 0x8080808080808080 // colonne 7 de chaque ligne
	colGE2   = 0xFCFCFCFCFCFCFCFC // colonnes 2 à 7 : arrivées possibles d'un saut vers l'est
	colLE5   = 0x3F3F3F3F3F3F3F3F // colonnes 0 à 5 : arrivées possibles d'un saut vers l'ouest

	// Directions : indice de la ligne de Game.moveIdx. L'arrivée d'un saut vers l'est est à droite des deux pions, etc.
	dirEast, dirWest, dirSouth, dirNorth = 0, 1, 2, 3
)

// rankBits : un rang de coup tient sur rankBits bits (il y a moins de 1024 coups sur une grille 8x8, au plus 192).
const rankBits = 10

// cand est un enfant d'un nœud : la position obtenue, la variation de pagode du coup et sa clé de tri
// (note << rankBits | rang aléatoire).
type cand struct {
	next Position
	dPag int32
	key  int32
}

// isoPenalty : pénalité d'un pion isolé dans l'évaluation (en nombre de coups possibles). Mesuré : de 0 à 16, le
// résultat varie peu ; la mobilité est ce qui compte.
const isoPenalty = 2

// landing renvoie, pour chaque direction, les cases d'arrivée des coups légaux depuis p dans le mask : cases vides dont
// les deux cases précédentes (dans la direction du saut) sont des pions. Les colonnes de bord sont masquées pour que le
// décalage d'une ligne ne déborde pas sur la suivante.
func landing(p, mask Position) (land [4]uint64) {
	empty := uint64(mask &^ p)
	x := uint64(p)
	land[dirEast] = (x << 1) & (x << 2) & colGE2 & empty
	land[dirWest] = (x >> 1) & (x >> 2) & colLE5 & empty
	land[dirSouth] = (x << 8) & (x << 16) & empty
	land[dirNorth] = (x >> 8) & (x >> 16) & empty
	return land
}

// isolated renvoie le nombre de pions sans aucun pion voisin (haut, bas, gauche, droite).
func isolated(p Position) int {
	nb := (p&^colLast)<<1 | (p&^colFirst)>>1 | p<<8 | p>>8
	return bits.OnesCount64(uint64(p &^ nb))
}

// mobility renvoie le nombre de coups légaux depuis p dans le mask.
func mobility(p, mask Position) int {
	land := landing(p, mask)
	return bits.OnesCount64(land[0]) + bits.OnesCount64(land[1]) + bits.OnesCount64(land[2]) + bits.OnesCount64(land[3])
}

// guideScore note une position : plus la note est grande, plus on l'essaie tôt.
// Note = mobilité (nombre de coups légaux ensuite) - isoPenalty * pions isolés. On préfère les positions qui laissent
// beaucoup de choix et peu de pions perdus. L'inverse (Warnsdorff : peu de coups d'abord) ne termine pas sur l'européen.
func (g *Game) guideScore(p Position) int32 {
	return int32(mobility(p, g.maskPos) - isoPenalty*isolated(p))
}

// buildMoveIdx remplit la table (direction, case d'arrivée) -> indice du coup dans validMoves.
func (g *Game) buildMoveIdx() {
	for i := range g.validMoves {
		m := &g.validMoves[i]
		d := dirEast
		switch {
		case m.to.c < m.from.c:
			d = dirWest
		case m.to.l > m.from.l:
			d = dirSouth
		case m.to.l < m.from.l:
			d = dirNorth
		}
		g.moveIdx[d][BitIndex(m.to)] = uint16(i)
	}
	g.moveIdxBuilt = true
}

// legalMoves renvoie l'ensemble des coups légaux depuis curr sous forme d'un champ de bits indexé par le rang du coup
// dans validMoves (bit i du mot i/64) : le parcourir par bits croissants redonne l'ordre de validMoves.
// Un plateau 8x8 complet a 192 coups, qui tiennent dans 3 mots.
func (g *Game) legalMoves(curr Position) (legal [3]uint64) {
	land := landing(curr, g.maskPos)
	for d := range land {
		for b := land[d]; b != 0; b &= b - 1 {
			idx := g.moveIdx[d][bits.TrailingZeros64(b)]
			legal[idx/64] |= 1 << (idx % 64)
		}
	}
	return legal
}

// sortedChildren remplit buf avec les enfants légaux de curr triés par clé décroissante (tri par insertion, sans
// allocation). Renvoie leur nombre, ou -1 si buf est trop petit (le nœud est alors exploré sans guidage).
func (g *Game) sortedChildren(curr Position, buf []cand) int {
	land := landing(curr, g.maskPos)
	n := 0
	for d := range land {
		for b := land[d]; b != 0; b &= b - 1 {
			idx := g.moveIdx[d][bits.TrailingZeros64(b)]
			m := &g.validMoves[idx]
			next := curr ^ m.mask3
			if n == len(buf) {
				return -1
			}
			k := g.guideScore(next)<<rankBits | int32(g.rank[idx])
			j := n
			for j > 0 && buf[j-1].key < k {
				buf[j] = buf[j-1]
				j--
			}
			buf[j] = cand{next, m.dPag, k}
			n++
		}
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

// shuffleRanks tire un nouveau rang aléatoire pour chaque coup (permutation de 0..len(validMoves)-1).
func (g *Game) shuffleRanks() {
	if g.rank == nil {
		g.rank = make([]uint16, len(g.validMoves))
	}
	for i := range g.rank {
		g.rank[i] = uint16(i)
	}
	g.rng.Shuffle(len(g.rank), func(i, j int) { g.rank[i], g.rank[j] = g.rank[j], g.rank[i] })
}
