//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Script de diagnóstico (Windows): lista os sensores de temperatura expostos
// pelo servidor web do LibreHardwareMonitor para você identificar o nome do
// sensor da CPU.
//
// Pré-requisitos: LibreHardwareMonitor aberto como administrador e com
// "Options > Remote Web Server > Run" ativado.
//
// Uso:
//
//	go run list_sensors_windows.go
type node struct {
	Text     string `json:"Text"`
	Type     string `json:"Type"`
	Value    string `json:"Value"`
	Children []node `json:"Children"`
}

func main() {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://localhost:8085/data.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao acessar o servidor web do LibreHardwareMonitor: %v\n", err)
		fmt.Fprintln(os.Stderr, "Verifique se ele está aberto e com 'Options > Remote Web Server > Run' ativado.")
		os.Exit(1)
	}
	defer resp.Body.Close()

	var root node
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		fmt.Fprintf(os.Stderr, "Resposta inválida: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("--- Sensores de Temperatura (LibreHardwareMonitor) ---")
	found := false
	var walk func(n *node)
	walk = func(n *node) {
		if n.Type == "Temperature" {
			fmt.Printf("Name: %-22s  Value: %s\n", n.Text, n.Value)
			found = true
		}
		for i := range n.Children {
			walk(&n.Children[i])
		}
	}
	walk(&root)
	if !found {
		fmt.Println("(nenhum sensor de temperatura encontrado)")
	}
	fmt.Println("------------------------------------------------------")
	fmt.Println("Use o nome do sensor da CPU (ex: \"CPU Package\" ou \"Core (Tctl/Tdie)\")")
	fmt.Println("e ajuste a lista preferredSensors em temp_windows.go, se necessário.")
}
