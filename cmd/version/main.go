// Affiche le numéro de version du package (utilisé par le Taskfile).
package main

import (
	"fmt"

	"github.com/xavier268/solitaire"
)

func main() {
	fmt.Print(solitaire.VERSION)
}
