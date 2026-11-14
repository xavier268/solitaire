package solitaire

import (
	"math/bits"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var moveFormat = regexp.MustCompile(`^[a-h][1-8]->[a-h][1-8]$`)

// checkExport vérifie l'export d'une solution : Moves() reconstruit des coups au bon format, qui rejoués depuis le
// départ redonnent exactement Solution(). Renvoie les coups.
func checkExport(t *testing.T, g *Game) []Move {
	t.Helper()
	sol := g.Solution()
	moves, err := g.Moves()
	if err != nil {
		t.Fatal(err)
	}
	if len(moves) != len(sol) {
		t.Fatalf("%d coups pour %d positions", len(moves), len(sol))
	}
	prev := g.startPos
	for i, m := range moves {
		if !moveFormat.MatchString(m.String()) {
			t.Fatalf("coup %d : format inattendu %q", i, m.String())
		}
		if prev = prev.DoMove(m); prev != sol[i] {
			t.Fatalf("coup %d (%s) : la position obtenue diffère de Solution()[%d]", i, m, i)
		}
	}
	return moves
}

// printSolution imprime la solution : le plateau de départ, puis pour chaque étape le coup et le plateau obtenu.
func printSolution(t *testing.T, title string, g *Game, moves []Move) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString(title + "\n")
	sb.WriteString("départ :\n" + g.startPos.Render(g.maskPos))
	for i, p := range g.Solution() {
		sb.WriteString("\n" + strconv.Itoa(i+1) + ". " + moves[i].String() + "\n")
		sb.WriteString(p.Render(g.maskPos))
	}
	t.Log("\n" + sb.String())
	var list []string
	for _, m := range moves {
		list = append(list, m.String())
	}
	t.Logf("%s : %d coups : %s", title, len(moves), strings.Join(list, " "))
}

// Un seul coup possible : d4 saute e4 et arrive en f4.
func TestExportSimple(t *testing.T) {
	mask := Bit(3, 3) | Bit(3, 4) | Bit(3, 5)
	g, err := NewGame(mask, Bit(3, 3)|Bit(3, 4), Bit(3, 5))
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Solution()) != 0 {
		t.Error("avant toute recherche, la solution doit être vide")
	}
	if err := g.Solve(); err != nil {
		t.Fatal(err)
	}
	moves := checkExport(t, &g)
	if len(moves) != 1 || moves[0].String() != "d4->f4" {
		t.Fatalf("coups = %v, attendu [d4->f4]", moves)
	}
}

// Solution() renvoie une copie : la modifier ne change pas l'état interne du jeu.
func TestSolutionIsACopy(t *testing.T) {
	g, err := NewGame(MASK_ENGLISHCROSS, START_ENGLISHCROSS, CENTER_ENGLISHCROSS)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Solve(); err != nil {
		t.Fatal(err)
	}
	s := g.Solution()
	s[0] = 0
	if g.Solution()[0] == 0 {
		t.Fatal("Solution() doit renvoyer une copie")
	}
	checkExport(t, &g)
}

// Exporte et imprime les solutions de la croix anglaise et du plateau européen, avec les coups entre les positions
// (go test -run TestPrintSolutions -v).
func TestPrintSolutions(t *testing.T) {
	cross, err := NewGame(MASK_ENGLISHCROSS, START_ENGLISHCROSS, CENTER_ENGLISHCROSS)
	if err != nil {
		t.Fatal(err)
	}
	if err := cross.Solve(); err != nil {
		t.Fatal(err)
	}
	moves := checkExport(t, &cross)
	if len(moves) != 31 {
		t.Fatalf("croix : %d coups, attendu 31", len(moves))
	}
	printSolution(t, "Croix anglaise, de d4 vide à d4 seul", &cross, moves)

	eu, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, 0)
	if err != nil {
		t.Fatal(err)
	}
	eu.rng = rand.New(rand.NewPCG(1, 2)) // graine fixée : même solution à chaque exécution
	if err := eu.SolveAnyEnd(); err != nil {
		t.Fatal(err)
	}
	moves = checkExport(t, &eu)
	if len(moves) != 35 {
		t.Fatalf("européen : %d coups, attendu 35", len(moves))
	}
	if last := eu.Solution()[len(moves)-1]; bits.OnesCount64(uint64(last)) != 1 {
		t.Fatalf("européen : %d billes à la fin", bits.OnesCount64(uint64(last)))
	}
	printSolution(t, "Plateau européen, trou en d3, une seule bille à la fin", &eu, moves)
}
