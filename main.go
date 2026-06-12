package main

import (
	"log"
	"time"
)

const (
	VendorID  = 0xaa88
	ProductID = 0x8666
)

// Device abstrai a comunicação com o display do cooler. A implementação muda
// por SO (go-hid no Linux, syscalls nativas no Windows), mas a interface e o
// loop principal abaixo são idênticos em qualquer plataforma.
type Device interface {
	WriteTemp(temp byte) error
	Close() error
}

// openDevice e getCPUTemp são implementadas nos arquivos específicos de cada
// SO (hid_*.go e temp_*.go), selecionados via build tags.

func main() {
	log.Println("Iniciando serviço de monitoramento do Mancer Mystic G1...")

	dev, err := openDevice(VendorID, ProductID)
	if err != nil {
		log.Fatalf("Erro ao abrir o dispositivo HID: %v", err)
	}
	defer dev.Close()

	log.Println("Dispositivo conectado com sucesso.")

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

		if err := dev.WriteTemp(byte(temp)); err != nil {
			// Se o dispositivo for desconectado, falhamos aqui. O serviço
			// (systemd no Linux / Agendador de Tarefas no Windows) reinicia.
			log.Fatalf("Erro fatal ao escrever no dispositivo (pode ter sido desconectado): %v", err)
		}
	}
}
