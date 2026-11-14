package solitaire

import "testing"

func TestNewGame(t *testing.T) {
	// Solitaire anglais classique : départ = croix sans le centre, arrivée = seulement le centre.
	g, err := NewGame(MASK_ENGLISHCROSS, START_ENGLISHCROSS, CENTER_ENGLISHCROSS)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if g.startPos != START_ENGLISHCROSS || g.endPos != CENTER_ENGLISHCROSS || g.maskPos != MASK_ENGLISHCROSS {
		t.Error("champs mal initialisés")
	}
	if n := len(g.validMoves); n != 76 {
		t.Errorf("%d moves valides, attendu 76", n)
	}
	if len(g.solution) != 0 || cap(g.solution) != 31 {
		t.Errorf("solution : len=%d cap=%d, attendu len=0 cap=31", len(g.solution), cap(g.solution))
	}
}

func TestNewGameErrors(t *testing.T) {
	outside := Bit(7, 7) // hors de la croix et du 5x5
	tests := []struct {
		name             string
		mask, start, end Position
	}{
		{"départ hors masque", MASK_ENGLISHCROSS, START_ENGLISHCROSS | outside, CENTER_ENGLISHCROSS},
		{"arrivée hors masque", MASK_ENGLISHCROSS, START_ENGLISHCROSS, outside},
		{"arrivée avec plus de pions que le départ", MASK_5x5, CENTER_5x5, START_5x5},
	}
	for _, tc := range tests {
		if _, err := NewGame(tc.mask, tc.start, tc.end); err == nil {
			t.Errorf("%s : erreur attendue", tc.name)
		}
	}
}

// checkSolution rejoue la solution depuis le départ : chaque étape doit être un move légal.
func checkSolution(t *testing.T, g Game) {
	t.Helper()
	if len(g.solution) == 0 || g.solution[len(g.solution)-1] != g.endPos {
		t.Fatalf("la solution doit se terminer sur endPos")
	}
	if len(g.solution) != cap(g.solution) {
		t.Errorf("solution de %d coups, attendu %d", len(g.solution), cap(g.solution))
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
			t.Fatalf("coup %d : %v n'est pas un move légal depuis la position précédente", i+1, pos)
		}
		prev = pos
	}
}

func TestSolveSimple(t *testing.T) {
	// 3 cases alignées : un seul coup possible.
	mask := Bit(3, 3) | Bit(3, 4) | Bit(3, 5)
	g, err := NewGame(mask, Bit(3, 3)|Bit(3, 4), Bit(3, 5))
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Solve(); err != nil {
		t.Fatalf("solution attendue : %v", err)
	}
	checkSolution(t, g)
	if len(g.solution) != 1 {
		t.Errorf("%d coups, attendu 1", len(g.solution))
	}
}

func TestSolveImpossible(t *testing.T) {
	// Cible inatteignable : pion en (3,4) alors que le départ ne permet que d'aller en (3,5).
	mask := Bit(3, 3) | Bit(3, 4) | Bit(3, 5)
	g, err := NewGame(mask, Bit(3, 3)|Bit(3, 4), Bit(3, 4))
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Solve(); err == nil {
		t.Error("erreur attendue : pas de solution")
	}
	if len(g.solution) != 0 {
		t.Errorf("solution doit être vide après échec, len=%d", len(g.solution))
	}
}

func TestSolveEnglishCross(t *testing.T) {
	if testing.Short() {
		t.Skip("brute force potentiellement long")
	}
	g, err := NewGame(MASK_ENGLISHCROSS, START_ENGLISHCROSS, CENTER_ENGLISHCROSS)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Solve(); err != nil {
		t.Fatalf("solution attendue : %v", err)
	}
	checkSolution(t, g)
	t.Logf("solution en %d coups, dernière position :\n%s", len(g.solution), g.solution[len(g.solution)-1].Render(g.maskPos))
}

// BenchmarkSolveEnglishCross mesure la résolution seule (NewGame est hors mesure).
// Solve réutilise la capacité de solution : on s'attend à 0 allocation par résolution.
// Lancer avec : go test -run '^$' -bench SolveEnglishCross -benchmem
func BenchmarkSolveEnglishCross(b *testing.B) {
	g, err := NewGame(MASK_ENGLISHCROSS, START_ENGLISHCROSS, CENTER_ENGLISHCROSS)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := g.Solve(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNewGameEnglishCross mesure la création du jeu (calcul de validMoves compris).
func BenchmarkNewGameEnglishCross(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := NewGame(MASK_ENGLISHCROSS, START_ENGLISHCROSS, CENTER_ENGLISHCROSS); err != nil {
			b.Fatal(err)
		}
	}
}
