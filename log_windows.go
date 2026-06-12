//go:build windows

package main

import (
	"log"
	"os"
	"path/filepath"
)

// No Windows o programa é compilado como aplicativo GUI (-H=windowsgui) para
// não abrir uma janela de console. Como nesse modo não há stderr visível,
// redirecionamos os logs para um arquivo na pasta de instalação.
//
// O arquivo é truncado a cada início, então sempre reflete a execução atual
// (útil porque a tarefa agendada reinicia o processo em caso de falha).
func init() {
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "MancerCooler")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return // mantém o comportamento padrão (stderr) se algo falhar
	}
	f, err := os.OpenFile(filepath.Join(dir, "monitor.log"),
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(f)
}
