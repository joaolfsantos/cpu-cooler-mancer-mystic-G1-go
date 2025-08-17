package main

import (
	"fmt"
	"log"
	"time"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/sstallion/go-hid"
)

const (
	VendorID  = 0xaa88
	ProductID = 0x8666
)

// getCPUTemp busca a temperatura da CPU.
// Adapte o `sensorKey` se necessário para o seu sistema.
// Sensores comuns são "k10temp" para AMD ou "coretemp" para Intel.
func getCPUTemp() (int, error) {
	temps, err := host.SensorsTemperatures()
	if err != nil {
		return 0, fmt.Errorf("erro ao ler sensores: %w", err)
	}

	for _, temp := range temps {
		// Usando o sensor de CPU correto para este sistema (AMD Ryzen).
		if temp.SensorKey == "k10temp_tctl" {
			return int(temp.Temperature), nil
		}
	}
	return 0, fmt.Errorf("nenhum sensor de CPU conhecido (k10temp/coretemp) foi encontrado")
}

func main() {
	log.Println("Iniciando serviço de monitoramento do Mancer Mystic G1...")

	device, err := hid.Open(VendorID, ProductID, "")
	if err != nil {
		log.Fatalf("Erro ao abrir o dispositivo HID: %v. Certifique-se de que as regras udev estão aplicadas.", err)
	}
	defer device.Close()

	log.Println("Dispositivo conectado com sucesso.")

	// Loop infinito de monitoramento
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		temp, err := getCPUTemp()
		if err != nil {
			log.Printf("Aviso: não foi possível obter a temperatura da CPU: %v", err)
			continue // Pula para a próxima iteração
		}

		// Garante que a temperatura está no range de um byte (0-255)
		if temp < 0 {
			temp = 0
		}
		if temp > 255 {
			temp = 255
		}

		report := []byte{0x00, byte(temp)}
		_, err = device.Write(report)
		if err != nil {
			// Se o dispositivo for desconectado, o programa irá falhar aqui.
			// O systemd o reiniciará automaticamente.
			log.Fatalf("Erro fatal ao escrever no dispositivo (pode ter sido desconectado): %v", err)
		}
	}
}
