package solitaire

import (
	"fmt"
	"math/bits"
	"strings"
)

// Une position sur le board (8x8 max) est reprensté par un bitmap sur 64bits (uint64)
type Position uint64

// Coordonnée sur la grille
type Point struct {
	l, c uint8
}

func (pt Point) String() string {
	return fmt.Sprintf("%c%1d", pt.c+'a', pt.l+1)
}

// BitIndex renvoie l'index du bit (0 à 63) correspondant à (ligne, colonne).
// ligne et colonne vont de 0 à 7.
func BitIndex(pt Point) uint8 {
	return pt.l*8 + pt.c
}

// Return un bitmask correspondnat aux coordonnées fournies
func Bit(l, c uint8) Position {
	return 1 << BitIndex(Point{l, c})
}

// Le bitMask correspondnat à un point donné
func (pt Point) Bit() Position {
	return 1 << BitIndex(pt)
}

// Masques définissant les limites des boards (1 octet par ligne, bit 0 = colonne 0).
const (
	// MASK_8x8 : toute la grille 8x8 (64 bits).
	MASK_8x8 Position = 0xFFFFFFFFFFFFFFFF

	// MASK_5x5 : carré de 5x5 en haut à gauche (x: 0..4, y: 0..4), 25 bits.
	MASK_5x5 Position = 0x0000001F1F1F1F1F

	// MASK_ENGLISHCROSS : croix du solitaire anglais, 33 bits.
	// Carré 7x7 (x,y: 0..6) privé de ses 4 coins 2x2.
	MASK_ENGLISHCROSS Position = 0x001C1C7F7F7F1C1C

	// MASK_EUROPEAN : plateau européen (français), 37 bits.
	// Carré 7x7 (x,y: 0..6) privé de ses 4 coins (2 cases en L : 3 cases par coin),
	// soit la croix anglaise plus les 4 cases (1,1), (1,5), (5,1) et (5,5).
	MASK_EUROPEAN Position = 0x001C3E7F7F7F3E1C
)

// Positions de départ : le mask complet, privé de sa case centrale.
// (Le 8x8 n'a pas de case centrale unique, il n'a donc pas de position de départ.)
const (
	// START_5x5 : 24 pions, trou en (ligne 2, colonne 2) soit c3.
	CENTER_5x5 Position = (1 << (2*8 + 2))
	START_5x5  Position = MASK_5x5 &^ CENTER_5x5

	// START_ENGLISHCROSS : 32 pions, trou en (ligne 3, colonne 3) soit d4.
	CENTER_ENGLISHCROSS Position = (1 << (3*8 + 3))
	START_ENGLISHCROSS  Position = MASK_ENGLISHCROSS &^ CENTER_ENGLISHCROSS

	// CENTER_EUROPEAN : case centrale du plateau européen, en (ligne 3, colonne 3) soit d4.
	//
	// ATTENTION : avec le trou de départ au centre (MASK_EUROPEAN sans CENTER_EUROPEAN), le solitaire
	// européen est INSOLUBLE, quelle que soit la cible (la règle de trois le prouve, cf. VerifyRuleOfThree ;
	// c'est le résultat de Bergholt). NewGame refuse donc cette configuration.
	CENTER_EUROPEAN Position = (1 << (3*8 + 3))

	// HOLE_EUROPEAN : trou de départ choisi pour le plateau européen, en (ligne 2, colonne 3) soit d3,
	// la case juste au-dessus du centre. Il est compatible avec la règle de trois et une solution existe :
	// une partie complète en 35 coups, finissant avec une seule bille en g5, a été trouvée par recherche.
	// La seule bille finale ne peut alors se trouver que sur d2, a5, d5 ou g5.
	HOLE_EUROPEAN Position = (1 << (2*8 + 3))

	// START_EUROPEAN : 36 pions, trou en d3. Utiliser une cible 0 (une seule bille, n'importe où ;
	// voir SolveAnyEnd), ou une des cases finales ci-dessus.
	START_EUROPEAN Position = MASK_EUROPEAN &^ HOLE_EUROPEAN
)

// Affiche une position. Utilise le mask fourni pour délimiter la surface jouable.
func (p Position) Print(mask Position) {
	p &= mask
	fmt.Printf("Bits actifs : %d\n", bits.OnesCount64(uint64(p)))
	fmt.Println(p.Render(mask))
}

// Render dessine une Position en ASCII ("O" = bit actif, "." = bit inactif, " " = hors mask),
// avec les indices de colonnes et de lignes.
func (p Position) Render(mask Position) string {
	p &= mask
	var sb strings.Builder
	sb.WriteString("  a b c d e f g h\n")
	for l := range uint8(8) {
		sb.WriteByte(byte('0' + l + 1))
		sb.WriteByte(' ')
		for c := range uint8(8) {
			bit := Position(1) << BitIndex(Point{l, c})
			switch {
			case mask&bit == 0:
				sb.WriteString("  ")
			case p&bit != 0:
				sb.WriteString("O ")
			default:
				sb.WriteString(". ")
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
