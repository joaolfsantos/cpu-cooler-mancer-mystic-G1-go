//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/shirou/gopsutil/v3/host"
)

// Script de diagnóstico: lista todos os sensores de temperatura
// disponíveis no sistema para identificar o sensor correto da CPU.
//
// Uso:
//
//	go run check_sensors.go
func main() {
	temps, err := host.SensorsTemperatures()
	if err != nil {
		// gopsutil retorna avisos (Warnings) quando alguns sensores falham,
		// mas ainda assim devolve os sensores lidos com sucesso.
		fmt.Fprintf(os.Stderr, "Aviso ao ler sensores: %v\n", err)
		if len(temps) == 0 {
			os.Exit(1)
		}
	}

	if len(temps) == 0 {
		fmt.Println("Nenhum sensor de temperatura foi encontrado.")
		fmt.Println("Verifique se o pacote 'lm-sensors' está instalado e configurado.")
		os.Exit(1)
	}

	fmt.Println("--- Sensores Encontrados ---")
	for _, temp := range temps {
		fmt.Printf("SensorKey:     %s\n", temp.SensorKey)
		fmt.Printf("  Temperatura: %.2f°C\n", temp.Temperature)
		fmt.Println("------------------------------")
	}
	fmt.Println("Procure o sensor da sua CPU (geralmente 'k10temp' para AMD ou 'coretemp' para Intel)")
	fmt.Println("e configure-o na linha do SensorKey em main.go.")
}
