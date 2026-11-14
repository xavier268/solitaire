package solitaire

import (
	"math/bits"
	"math/rand/v2"
	"testing"
)

// Références lentes, case par case.
func refIsolated(p Position) (n int) {
	has := func(l, c int) bool { return l >= 0 && l < 8 && c >= 0 && c < 8 && p&Bit(uint8(l), uint8(c)) != 0 }
	for i := 0; i < 64; i++ {
		l, c := i/8, i%8
		if has(l, c) && !has(l-1, c) && !has(l+1, c) && !has(l, c-1) && !has(l, c+1) {
			n++
		}
	}
	return n
}

func refMobility(p, mask Position) (n int) {
	for _, m := range mask.AllMoves() {
		if p.DoMove(m) != 0 {
			n++
		}
	}
	return n
}

// isolated et mobility (opérations sur les bits) égalent leurs références, y compris sur une grille 8x8 complète
// (les colonnes 0 et 7 ne doivent pas se contaminer d'une ligne à l'autre).
func TestGuideFeatures(t *testing.T) {
	rng := rand.New(rand.NewPCG(21, 22))
	for name, mask := range map[string]Position{"8x8": MASK_8x8, "européen": MASK_EUROPEAN, "croix": MASK_ENGLISHCROSS, "5x5": MASK_5x5} {
		for i := 0; i < 2000; i++ {
			p := Position(rng.Uint64()&rng.Uint64()) & mask // positions plus ou moins clairsemées
			if got, want := isolated(p), refIsolated(p); got != want {
				t.Fatalf("%s : isolated(%#x) = %d, référence %d", name, uint64(p), got, want)
			}
			if got, want := mobility(p, mask), refMobility(p, mask); got != want {
				t.Fatalf("%s : mobility(%#x) = %d, référence %d", name, uint64(p), got, want)
			}
		}
	}
}

// Le tri des enfants est décroissant, stable, et contient exactement les coups légaux.
func TestSortedChildren(t *testing.T) {
	g, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, 0)
	if err != nil {
		t.Fatal(err)
	}
	g.shuffleRanks()
	rng := rand.New(rand.NewPCG(23, 24))
	p := g.startPos
	buf := make([]cand, maxGuided)
	for step := 0; step < 30; step++ {
		n := g.sortedChildren(p, buf)
		legal := 0
		for _, m := range g.validMoves {
			if p.DoMove(m) != 0 {
				legal++
			}
		}
		if n != legal {
			t.Fatalf("%d enfants triés, %d coups légaux", n, legal)
		}
		for i := 1; i < n; i++ {
			if buf[i-1].key < buf[i].key {
				t.Fatal("tri non décroissant")
			}
		}
		if n == 0 {
			break
		}
		p = buf[rng.IntN(n)].next
	}
}

// Une recherche guidée (tentatives aléatoires) donne une suite de coups légaux jusqu'à une seule bille.
func TestSolveEuropeanGuided(t *testing.T) {
	for seed := uint64(0); seed < 5; seed++ {
		g, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, 0)
		if err != nil {
			t.Fatal(err)
		}
		g.rng = rand.New(rand.NewPCG(seed, 1))
		if err := g.SolveAnyEnd(); err != nil {
			t.Fatal(err)
		}
		prev := g.startPos
		for i, pos := range g.solution {
			ok := false
			for _, m := range g.validMoves {
				if prev.DoMove(m) == pos {
					ok = true
					break
				}
			}
			if !ok {
				t.Fatalf("graine %d, étape %d : coup illégal", seed, i)
			}
			prev = pos
		}
		if bits.OnesCount64(uint64(prev)) != 1 || len(g.solution) != 35 {
			t.Fatalf("graine %d : %d billes, %d coups", seed, bits.OnesCount64(uint64(prev)), len(g.solution))
		}
	}
}

// L'énumération par bits (landing + moveIdx) donne exactement les mêmes coups légaux que le test de chaque coup
// avec DoMove, y compris sur la grille 8x8 complète (colonnes de bord).
func TestLandingMatchesDoMove(t *testing.T) {
	rng := rand.New(rand.NewPCG(31, 32))
	for name, mask := range map[string]Position{"8x8": MASK_8x8, "européen": MASK_EUROPEAN, "croix": MASK_ENGLISHCROSS, "5x5": MASK_5x5} {
		g := Game{maskPos: mask, validMoves: mask.AllMoves()}
		g.buildMoveIdx()
		for range 3000 {
			p := Position(rng.Uint64()&rng.Uint64()|rng.Uint64()&rng.Uint64()) & mask
			want := map[Position]uint16{}
			for idx, m := range g.validMoves {
				if next := p.DoMove(m); next != 0 {
					want[next] = uint16(idx)
				}
			}
			got := map[Position]uint16{}
			land := landing(p, mask)
			for d := range land {
				for b := land[d]; b != 0; b &= b - 1 {
					idx := g.moveIdx[d][bits.TrailingZeros64(b)]
					got[p^g.validMoves[idx].mask3] = idx
				}
			}
			if len(got) != len(want) {
				t.Fatalf("%s, %#x : %d coups énumérés, %d légaux", name, uint64(p), len(got), len(want))
			}
			for next, idx := range want {
				if got[next] != idx {
					t.Fatalf("%s, %#x : coup %d absent ou différent", name, uint64(p), idx)
				}
			}
		}
	}
}

// legalMoves, parcouru par bits croissants, redonne les coups légaux dans l'ordre de validMoves : c'est l'ordre
// historique de la 1re tentative, dont dépend le nombre de nœuds de la croix anglaise.
func TestLegalMovesOrder(t *testing.T) {
	rng := rand.New(rand.NewPCG(41, 42))
	for name, mask := range map[string]Position{"8x8": MASK_8x8, "européen": MASK_EUROPEAN, "croix": MASK_ENGLISHCROSS} {
		g := Game{maskPos: mask, validMoves: mask.AllMoves()}
		g.buildMoveIdx()
		for range 1000 {
			p := Position(rng.Uint64()&rng.Uint64()|rng.Uint64()&rng.Uint64()) & mask
			var want []int
			for idx, m := range g.validMoves {
				if p.DoMove(m) != 0 {
					want = append(want, idx)
				}
			}
			var got []int
			legal := g.legalMoves(p)
			for w := range legal {
				for b := legal[w]; b != 0; b &= b - 1 {
					got = append(got, w*64+bits.TrailingZeros64(b))
				}
			}
			if len(got) != len(want) {
				t.Fatalf("%s : %d coups, attendu %d", name, len(got), len(want))
			}
			for k := range got {
				if got[k] != want[k] {
					t.Fatalf("%s : ordre différent (%v / %v)", name, got, want)
				}
			}
		}
	}
	if n := len(MASK_8x8.AllMoves()); n > 192 {
		t.Fatalf("%d coups sur 8x8 : legalMoves (3 mots) ne suffit plus", n)
	}
}
