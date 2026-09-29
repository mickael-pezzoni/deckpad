package desktop

import (
	"os/exec"
	"syscall"
)

const swShowNormal = 1

// Open ouvre un dossier, un fichier ou une adresse avec le programme associé,
// au premier plan.
func Open(target string) error {
	before := shownWindows()
	unlockForeground()
	proc, err := shellOpen(target)
	if err != nil {
		return err
	}
	go raise(before, proc)
	return nil
}

// Launch lance un programme avec start, puis passe sa fenêtre au premier plan.
func Launch(start func() error) error {
	before := shownWindows()
	unlockForeground()
	if err := start(); err != nil {
		return err
	}
	go raise(before, 0)
	return nil
}

// Reveal ouvre l'Explorateur sur le dossier du fichier, le fichier sélectionné,
// au premier plan.
func Reveal(path string) error {
	before := shownWindows()
	unlockForeground()
	cmd := exec.Command("explorer.exe")
	// L'Explorateur veut « /select,"chemin" » tel quel : Go mettrait les
	// guillemets autour de tout l'argument.
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer.exe /select,"` + path + `"`}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() // explorer.exe renvoie 1 même quand tout va bien
	// La fenêtre appartient à l'Explorateur déjà lancé, pas à ce processus.
	go raise(before, 0)
	return nil
}
