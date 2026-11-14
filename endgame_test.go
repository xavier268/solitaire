package solitaire

import (
	"math/bits"
	"math/rand/v2"
	"testing"
)

// UndoMove est l'inverse de DoMove.
func TestUndoMoveInvertsDoMove(t *testing.T) {
	g, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, 0)
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewPCG(3, 4))
	checked := 0
	for walk := 0; walk < 100; walk++ {
		p := g.startPos
		for {
			var legal []Move
			for _, m := range g.validMoves {
				if p.DoMove(m) != 0 {
					legal = append(legal, m)
				}
			}
			if len(legal) == 0 {
				break
			}
			m := legal[rng.IntN(len(legal))]
			q := p.DoMove(m)
			if back := q.UndoMove(m); back != p {
				t.Fatalf("UndoMove(DoMove(p)) = %#x, attendu %#x", uint64(back), uint64(p))
			}
			if p.UndoMove(m) != 0 && p.UndoMove(m) == p {
				t.Fatal("UndoMove ne doit pas renvoyer la position elle-même")
			}
			p = q
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("aucun coup testé")
	}
}

func TestPosSet(t *testing.T) {
	s := newPosSet(10)
	rng := rand.New(rand.NewPCG(5, 6))
	ref := map[Position]bool{}
	for i := 0; i < 20000; i++ {
		p := Position(rng.Uint64() | 1)
		s.add(p)
		ref[p] = true
	}
	if s.n != len(ref) {
		t.Fatalf("n = %d, attendu %d", s.n, len(ref))
	}
	for p := range ref {
		if !s.has(p) {
			t.Fatalf("%#x manquant", uint64(p))
		}
	}
	for i := 0; i < 20000; i++ {
		if p := Position(rng.Uint64() | 1); s.has(p) != ref[p] {
			t.Fatalf("has(%#x) incorrect", uint64(p))
		}
	}
}

// La table de fins de partie doit être exacte : une position à K pions est dans la table si et seulement si
// une recherche exhaustive indépendante (sans table) la résout.
func TestEndgameExact(t *testing.T) {
	g, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, 0)
	if err != nil {
		t.Fatal(err)
	}
	g.buildEndgame(true)
	k := g.endgameK
	if k < 5 {
		t.Fatalf("table trop petite : K = %d", k)
	}
	t.Logf("K = %d, %d positions", k, g.endgame.n)

	solvable := func(pos Position) bool {
		h, err := NewGame(MASK_EUROPEAN, pos, 0)
		if err != nil {
			return false // règle de trois : insoluble
		}
		h.endgameOff = true
		return h.SolveAnyEnd() == nil
	}

	rng := rand.New(rand.NewPCG(7, 8))
	seen := map[Position]bool{}
	yes, no := 0, 0
	for walk := 0; walk < 3000 && len(seen) < 300; walk++ {
		p := g.startPos
		for {
			if bits.OnesCount64(uint64(p)) == k {
				if !seen[p] {
					seen[p] = true
					want := solvable(p)
					if got := g.endgame.has(p); got != want {
						t.Fatalf("position %#x (%d pions) : table = %v, recherche = %v", uint64(p), k, got, want)
					}
					if want {
						yes++
					} else {
						no++
					}
				}
				break
			}
			var legal []Move
			for _, m := range g.validMoves {
				if p.DoMove(m) != 0 {
					legal = append(legal, m)
				}
			}
			if len(legal) == 0 {
				break
			}
			p = p.DoMove(legal[rng.IntN(len(legal))])
		}
	}
	t.Logf("%d positions testées : %d résolubles, %d perdantes", len(seen), yes, no)
	if len(seen) == 0 {
		t.Fatal("aucune position testée")
	}
}

// Cible précise : la table construite à partir de endPos est exacte aussi.
func TestEndgameExactTarget(t *testing.T) {
	end := Bit(4, 6) // g5 : dernière bille de la solution connue
	g, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, end)
	if err != nil {
		t.Fatal(err)
	}
	g.buildEndgame(false)
	if g.endgameK < 5 {
		t.Fatalf("table trop petite : K = %d", g.endgameK)
	}
	k := g.endgameK
	rng := rand.New(rand.NewPCG(9, 10))
	seen := map[Position]bool{}
	for walk := 0; walk < 3000 && len(seen) < 150; walk++ {
		p := g.startPos
		for {
			if bits.OnesCount64(uint64(p)) == k {
				if !seen[p] {
					seen[p] = true
					h, err := NewGame(MASK_EUROPEAN, p, end)
					want := false
					if err == nil {
						h.endgameOff = true
						want = h.Solve() == nil
					}
					if got := g.endgame.has(p); got != want {
						t.Fatalf("position %#x : table = %v, recherche = %v", uint64(p), got, want)
					}
				}
				break
			}
			var legal []Move
			for _, m := range g.validMoves {
				if p.DoMove(m) != 0 {
					legal = append(legal, m)
				}
			}
			if len(legal) == 0 {
				break
			}
			p = p.DoMove(legal[rng.IntN(len(legal))])
		}
	}
}

// Les positions d'une vraie solution sont résolubles : elles doivent toutes être dans la table (pour le bon K).
func TestEndgameContainsSolutionPositions(t *testing.T) {
	g, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, 0)
	if err != nil {
		t.Fatal(err)
	}
	g.buildEndgame(true)
	k := g.endgameK
	g.endgameBuilt = true // la table est déjà construite
	for i := 0; i < 3; i++ {
		h, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, 0)
		if err != nil {
			t.Fatal(err)
		}
		h.endgameOff = true
		if err := h.SolveAnyEnd(); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, p := range h.solution {
			if bits.OnesCount64(uint64(p)) == k {
				found = true
				if !g.endgame.has(p) {
					t.Fatalf("position de solution %#x (%d pions) absente de la table", uint64(p), k)
				}
			}
		}
		if !found {
			t.Fatalf("aucune position à %d pions dans la solution", k)
		}
	}
}
