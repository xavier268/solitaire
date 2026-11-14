// Affiche les caractéristiques de la machine et du build, en en-tête des fichiers de benchmark.
// Stdlib uniquement ; le modèle du CPU est ajouté par `go test -bench` (ligne "cpu:").
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/xavier268/solitaire"
)

func main() {
	host, _ := os.Hostname()

	fmt.Printf("# solitaire version : %s\n", solitaire.VERSION)
	fmt.Printf("# date              : %s\n", time.Now().Format("2006-01-02 15:04:05 -07:00"))
	fmt.Printf("# git               : %s\n", gitInfo())
	fmt.Printf("# machine           : %s\n", host)
	fmt.Printf("# os/arch           : %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("# os                : %s\n", osInfo())
	fmt.Printf("# cpu logiques      : %d (GOMAXPROCS=%d)\n", runtime.NumCPU(), runtime.GOMAXPROCS(0))
	fmt.Printf("# ram               : %s\n", ramInfo())
	fmt.Printf("# go                : %s\n", runtime.Version())
	fmt.Println("#")
}

// gitInfo renvoie le commit courant, avec "+dirty" si des modifications ne sont pas commitées.
func gitInfo() string {
	commit := run("git", "rev-parse", "--short", "HEAD")
	if commit == "" {
		return "inconnu"
	}
	if run("git", "status", "--porcelain") != "" {
		commit += "+dirty"
	}
	return commit
}

// run exécute une commande et renvoie sa sortie nettoyée, ou "" en cas d'erreur.
func run(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
