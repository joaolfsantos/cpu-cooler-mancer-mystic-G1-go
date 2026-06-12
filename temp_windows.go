//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// No Windows a temperatura da CPU é lida do servidor web embutido do
// LibreHardwareMonitor, que entrega todos os sensores em JSON. Essa é a
// interface confiável do LHM (o WMI não é publicado nesta versão).
//
// Pré-requisito: LibreHardwareMonitor aberto como administrador e com
// "Options > Remote Web Server > Run" ativado.
const lhmURL = "http://localhost:8085/data.json"

// lhmNode é um nó da árvore de sensores retornada pelo /data.json.
// A estrutura é recursiva: Sensor -> Computador -> Hardware -> Categoria -> Sensor.
type lhmNode struct {
	Text     string    `json:"Text"`
	Type     string    `json:"Type"`  // "Temperature", "Load", "Clock", etc.
	Value    string    `json:"Value"` // ex: "45.0 °C"
	Children []lhmNode `json:"Children"`
}

// nomes de sensor preferidos, em ordem, que melhor representam a temperatura
// "geral" da CPU. O primeiro que existir é usado.
var preferredSensors = []string{
	"CPU Package",      // Intel
	"Core (Tctl/Tdie)", // AMD Ryzen
	"CPU Cores",        // AMD (média)
	"Core Max",
	"Core Average",
}

func getCPUTemp() (int, error) {
	root, err := fetchLHM()
	if err != nil {
		return 0, err
	}

	temps := collectTemps(root)
	if len(temps) == 0 {
		return 0, fmt.Errorf("nenhum sensor de temperatura encontrado no LibreHardwareMonitor")
	}

	// 1) nomes preferidos, na ordem.
	for _, want := range preferredSensors {
		if v, ok := temps[want]; ok {
			return int(v + 0.5), nil
		}
	}
	// 2) fallback: qualquer sensor cujo nome contenha "CPU".
	for name, v := range temps {
		if strings.Contains(name, "CPU") {
			return int(v + 0.5), nil
		}
	}
	return 0, fmt.Errorf("nenhum sensor de CPU reconhecido (rode list_sensors_windows.go para ver os nomes disponíveis)")
}

// collectTemps percorre a árvore e devolve um mapa nome -> °C com todos os
// sensores de temperatura encontrados (mantém a primeira ocorrência de cada nome).
func collectTemps(root *lhmNode) map[string]float64 {
	temps := map[string]float64{}
	var walk func(n *lhmNode)
	walk = func(n *lhmNode) {
		if n.Type == "Temperature" {
			if v, ok := parseValue(n.Value); ok {
				if _, exists := temps[n.Text]; !exists {
					temps[n.Text] = v
				}
			}
		}
		for i := range n.Children {
			walk(&n.Children[i])
		}
	}
	walk(root)
	return temps
}

func fetchLHM() (*lhmNode, error) {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(lhmURL)
	if err != nil {
		return nil, fmt.Errorf("falha ao acessar o servidor web do LibreHardwareMonitor em %s "+
			"(ele está aberto e com 'Remote Web Server' ativo?): %w", lhmURL, err)
	}
	defer resp.Body.Close()

	var root lhmNode
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return nil, fmt.Errorf("resposta inválida do LibreHardwareMonitor: %w", err)
	}
	return &root, nil
}

// parseValue extrai o número de uma string como "45.0 °C" (suporta vírgula decimal).
func parseValue(s string) (float64, bool) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0, false
	}
	tok := strings.Replace(fields[0], ",", ".", 1)
	f, err := strconv.ParseFloat(tok, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}
