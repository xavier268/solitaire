package solitaire

import (
	"math/rand/v2"
	"os"
	"reflect"
	"testing"
)

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
	// Cible que la règle de trois ne rejette pas (parités compatibles) mais qui est inatteignable :
	// deux pions isolés, sans aucun coup possible. C'est donc la recherche, et non NewGame, qui échoue.
	g, err := NewGame(MASK_5x5, Bit(0, 0)|Bit(4, 3), Bit(2, 0))
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
// Solve réutilise la capacité de solution et la map des impasses : on s'attend à 0 allocation par résolution.
// Une première résolution hors mesure crée la map (allocation unique), les suivantes la réutilisent.
// Lancer avec : go test -run '^$' -bench SolveEnglishCross -benchmem
func BenchmarkSolveEnglishCross(b *testing.B) {
	g, err := NewGame(MASK_ENGLISHCROSS, START_ENGLISHCROSS, CENTER_ENGLISHCROSS)
	if err != nil {
		b.Fatal(err)
	}
	if err := g.Solve(); err != nil { // échauffement : alloue la map des impasses
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

func TestSolveAnyEnd(t *testing.T) {
	mask := Bit(3, 3) | Bit(3, 4) | Bit(3, 5)

	// Cible 0 : un seul pion restant, n'importe où.
	g, err := NewGame(mask, Bit(3, 3)|Bit(3, 4), 0)
	if err != nil {
		t.Fatal(err)
	}
	if cap(g.solution) != 1 {
		t.Errorf("cap(solution) = %d, attendu 1 (2 pions -> 1 pion)", cap(g.solution))
	}
	if err := g.SolveAnyEnd(); err != nil {
		t.Fatalf("solution attendue : %v", err)
	}
	if len(g.solution) != 1 || g.solution[0] != Bit(3, 5) {
		t.Errorf("solution = %v, attendu un seul coup vers (3,5)", g.solution)
	}

	// Cible non nulle : même comportement que Solve (cible exacte). Ici la cible est atteignable.
	g, err = NewGame(mask, Bit(3, 3)|Bit(3, 4), Bit(3, 5))
	if err != nil {
		t.Fatal(err)
	}
	if err := g.SolveAnyEnd(); err != nil || len(g.solution) != 1 {
		t.Errorf("cible exacte atteignable : résolu en 1 coup attendu (err=%v, len=%d)", err, len(g.solution))
	}

	// Aucun move possible et plus d'un pion : pas de solution, mais la règle de trois ne le voit pas
	// (une case finale compatible existe), donc NewGame accepte et c'est la recherche qui échoue.
	g, err = NewGame(MASK_5x5, Bit(0, 0)|Bit(4, 3), 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.SolveAnyEnd(); err == nil {
		t.Error("deux pions isolés : erreur attendue")
	}
	if len(g.solution) != 0 {
		t.Errorf("solution doit être vide après échec, len=%d", len(g.solution))
	}

	// Un seul pion au départ : déjà résolu, sans aucun coup.
	g, err = NewGame(mask, Bit(3, 3), 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.SolveAnyEnd(); err != nil || len(g.solution) != 0 {
		t.Errorf("un seul pion : résolu sans coup attendu (err=%v, len=%d)", err, len(g.solution))
	}
}

// BenchmarkNewGameEuropean mesure la création du jeu européen (calcul de validMoves compris).
func BenchmarkNewGameEuropean(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, 0); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSolveEuropean mesure la résolution du plateau européen avec SolveAnyEnd
// (cible 0 : une seule bille restante, n'importe où).
func BenchmarkSolveEuropean(b *testing.B) {
	// Une solution existe (trou en d3), mais le DFS brute force sans élagage ni mémoire ne la trouve pas
	// en un temps raisonnable (plus de 90 s mesurées) : désactivé par défaut pour ne pas bloquer `task bench`.
	// Activer avec la variable d'environnement BENCH_EUROPEAN_SOLVE=1.
	if os.Getenv("BENCH_EUROPEAN_SOLVE") == "" {
		b.Skip("résolution européenne trop longue sans élagage ; définir BENCH_EUROPEAN_SOLVE=1")
	}

	g, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, 0)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := g.SolveAnyEnd(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestVerifyRuleOfThree(t *testing.T) {
	tests := []struct {
		name             string
		mask, start, end Position
		wantErr          bool
	}{
		// Solubles (la règle ne doit jamais rejeter une partie qui a une solution).
		{"croix anglaise, centre vers centre", MASK_ENGLISHCROSS, START_ENGLISHCROSS, CENTER_ENGLISHCROSS, false},
		{"croix anglaise, un pion n'importe où", MASK_ENGLISHCROSS, START_ENGLISHCROSS, 0, false},
		{"européen, trou en c1, un pion n'importe où", MASK_EUROPEAN, MASK_EUROPEAN &^ Bit(0, 2), 0, false},
		{"européen, trou en d3, un pion n'importe où", MASK_EUROPEAN, START_EUROPEAN, 0, false},
		{"européen, trou en d3, dernière bille en g5", MASK_EUROPEAN, START_EUROPEAN, Bit(4, 6), false},
		{"trois alignés", Bit(3, 3) | Bit(3, 4) | Bit(3, 5), Bit(3, 3) | Bit(3, 4), Bit(3, 5), false},

		// Insolubles, détectées par la parité.
		{"européen, trou au centre, un pion n'importe où (insoluble)", MASK_EUROPEAN, MASK_EUROPEAN &^ CENTER_EUROPEAN, 0, true},
		{"européen, centre vers centre (insoluble)", MASK_EUROPEAN, MASK_EUROPEAN &^ CENTER_EUROPEAN, CENTER_EUROPEAN, true},
		{"européen, trou en d3, dernière bille au centre", MASK_EUROPEAN, START_EUROPEAN, CENTER_EUROPEAN, true},
		{"trois alignés, mauvaise case", Bit(3, 3) | Bit(3, 4) | Bit(3, 5), Bit(3, 3) | Bit(3, 4), Bit(3, 4), true},
		{"un pion en trop vers un coin", MASK_ENGLISHCROSS, START_ENGLISHCROSS, Bit(0, 2), true},
	}
	for _, tc := range tests {
		// VerifyRuleOfThree est testée directement sur un Game construit à la main :
		// NewGame appelle elle-même la règle et refuserait de construire les cas insolubles.
		g := Game{maskPos: tc.mask, startPos: tc.start, endPos: tc.end}
		err := g.VerifyRuleOfThree()
		if (err != nil) != tc.wantErr {
			t.Errorf("%s : erreur = %v, attendu erreur = %v", tc.name, err, tc.wantErr)
		} else if err != nil {
			t.Logf("%s : %v", tc.name, err)
		}

		// NewGame doit refuser exactement les mêmes configurations (la règle y est intégrée).
		if _, err := NewGame(tc.mask, tc.start, tc.end); (err != nil) != tc.wantErr {
			t.Errorf("%s : NewGame erreur = %v, attendu erreur = %v", tc.name, err, tc.wantErr)
		}
	}
}

// Vérifie que la règle ne rejette jamais une position réellement atteignable :
// on joue des parties aléatoires (graine fixe) et chaque position rencontrée doit passer le test.
func TestVerifyRuleOfThreeSound(t *testing.T) {
	for _, board := range []struct {
		name        string
		mask, start Position
	}{
		{"croix anglaise", MASK_ENGLISHCROSS, START_ENGLISHCROSS},
		{"européen", MASK_EUROPEAN, START_EUROPEAN},
		{"5x5", MASK_5x5, START_5x5},
	} {
		moves := board.mask.AllMoves()
		rng := rand.New(rand.NewPCG(1, 2))
		for game := 0; game < 200; game++ {
			p := board.start
			for {
				var legal []Position
				for _, m := range moves {
					if pp := p.DoMove(m); pp != 0 {
						legal = append(legal, pp)
					}
				}
				if len(legal) == 0 {
					break
				}
				p = legal[rng.IntN(len(legal))]
				g, err := NewGame(board.mask, board.start, p)
				if err != nil {
					t.Fatal(err)
				}
				if err := g.VerifyRuleOfThree(); err != nil {
					t.Fatalf("%s : position atteignable rejetée : %v\n%s", board.name, err, p.Render(board.mask))
				}
			}
		}
	}
}

// La map des impasses n'est allouée qu'une fois : les recherches suivantes réutilisent sa mémoire (clear).
func TestDeadMapAllocatedOnce(t *testing.T) {
	g, err := NewGame(MASK_ENGLISHCROSS, START_ENGLISHCROSS, CENTER_ENGLISHCROSS)
	if err != nil {
		t.Fatal(err)
	}
	if g.dead != nil {
		t.Error("la map ne doit pas être allouée avant la première recherche")
	}

	// Première recherche : la map est créée puis remplie d'impasses.
	if err := g.Solve(); err != nil {
		t.Fatal(err)
	}
	if len(g.dead) == 0 {
		t.Fatal("des impasses auraient dû être mémorisées")
	}
	first := reflect.ValueOf(g.dead).Pointer()

	// Recherches suivantes : même map, aucune allocation.
	allocs := testing.AllocsPerRun(5, func() {
		if err := g.Solve(); err != nil {
			t.Fatal(err)
		}
	})
	if allocs != 0 {
		t.Errorf("Solve alloue %v fois après la première recherche, attendu 0", allocs)
	}
	if reflect.ValueOf(g.dead).Pointer() != first {
		t.Error("la map des impasses a été recréée")
	}
	checkSolution(t, g)
}
