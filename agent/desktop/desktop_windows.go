package desktop

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

const swShowNormal = 1

// Open ouvre un dossier, un fichier ou une adresse avec le programme associé.
func Open(target string) error {
	verb, _ := windows.UTF16PtrFromString("open")
	file, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, file, nil, nil, swShowNormal)
}

// Reveal ouvre l'Explorateur sur le dossier du fichier, le fichier sélectionné.
func Reveal(path string) error {
	cmd := exec.Command("explorer.exe")
	// L'Explorateur veut « /select,"chemin" » tel quel : Go mettrait les
	// guillemets autour de tout l'argument.
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer.exe /select,"` + path + `"`}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() // explorer.exe renvoie 1 même quand tout va bien
	return nil
}
