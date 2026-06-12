//go:build linux

package main

import (
	"fmt"

	"github.com/shirou/gopsutil/v3/host"
)

// getCPUTemp busca a temperatura da CPU via lm-sensors (gopsutil).
// Adapte o `SensorKey` se necessário: "k10temp" para AMD, "coretemp" para Intel.
// Use `go run check_sensors.go` para descobrir o nome correto no seu sistema.
func getCPUTemp() (int, error) {
	temps, err := host.SensorsTemperatures()
	if err != nil {
		// gopsutil retorna avisos quando alguns sensores falham, mas ainda
		// devolve os lidos com sucesso. Só abortamos se nenhum foi lido.
		if len(temps) == 0 {
			return 0, fmt.Errorf("erro ao ler sensores: %w", err)
		}
	}

	for _, temp := range temps {
		// Sensor de CPU deste sistema (AMD Ryzen).
		if temp.SensorKey == "k10temp_tctl" {
			return int(temp.Temperature), nil
		}
	}
	return 0, fmt.Errorf("nenhum sensor de CPU conhecido (k10temp/coretemp) foi encontrado")
}
