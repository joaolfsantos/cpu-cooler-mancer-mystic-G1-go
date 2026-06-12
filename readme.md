# Monitor de Display para o Water Cooler Mancer Mystic G1

Driver/serviço simples, escrito em Go, que lê a temperatura da CPU em tempo real
e a exibe no display do Water Cooler **Mancer Mystic G1**. Roda em segundo plano
e inicia automaticamente com o sistema.

Funciona em **Linux** e em **Windows**. O loop principal e o protocolo de
comunicação com o cooler são os mesmos nos dois; apenas a leitura da temperatura
e a inicialização em background mudam por sistema.

## Como funciona

A cada segundo o programa lê a temperatura da CPU e envia ao cooler um relatório
HID de 2 bytes (`[0x00, temperatura]`) para o dispositivo USB `aa88:8666`.

O código específico de cada SO é separado por *build tags*, então o mesmo
projeto compila para os dois sistemas:

| Arquivo | Sistema | Responsabilidade |
|---|---|---|
| `main.go` | comum | Loop principal, limite de 0–255 e orquestração |
| `temp_linux.go` | Linux | Temperatura via `lm-sensors` (`gopsutil`) |
| `hid_linux.go` | Linux | Comunicação HID via `go-hid` (hidapi/cgo) |
| `temp_windows.go` | Windows | Temperatura via servidor web do LibreHardwareMonitor (JSON) |
| `hid_windows.go` | Windows | Comunicação HID via syscalls nativas (Go puro, sem cgo) |
| `log_windows.go` | Windows | Redireciona logs para arquivo (o app roda sem console) |
| `check_sensors.go` | Linux | Diagnóstico: lista os sensores do `lm-sensors` |
| `list_sensors_windows.go` | Windows | Diagnóstico: lista os sensores do LibreHardwareMonitor |

> Por que diferente no Windows? O Windows não tem `lm-sensors`, e ler a
> temperatura da CPU exige acesso a registradores via driver de kernel. Em vez
> de reimplementar isso, apoiamo-nos no **LibreHardwareMonitor**, que já faz esse
> trabalho e expõe os sensores no seu servidor web embutido.

## Sistemas testados

* **Linux:** Fedora (KDE/GNOME) e Ubuntu (GNOME) — o ambiente gráfico não influi.
* **Windows:** Windows 11 (AMD Ryzen).

---

# Instalação no Linux

O programa usa `cgo`, então precisa de um compilador C, das bibliotecas HID/USB
e do `lm-sensors`.

## Passo 1 — Instalar as dependências

### Fedora (KDE ou GNOME)
```bash
sudo dnf install golang gcc hidapi-devel systemd-devel lm_sensors
```

### Ubuntu (GNOME)
```bash
sudo apt-get update
sudo apt-get install golang-go build-essential libhidapi-dev libudev-dev lm-sensors
```

| Pacote (Fedora / Ubuntu) | Para que serve |
|---|---|
| `golang` / `golang-go` | Compilador Go |
| `gcc` / `build-essential` | Compilador C, exigido pelo `cgo` |
| `hidapi-devel` / `libhidapi-dev` | Comunicação com dispositivos HID |
| `systemd-devel` / `libudev-dev` | Fornece o `libudev.h`, exigido pela `go-hid` |
| `lm_sensors` / `lm-sensors` | Leitura dos sensores de temperatura |

> **Opcional:** rode `sudo sensors-detect` e responda `YES` às perguntas para
> expor o máximo de sensores.

## Passo 2 — Clonar e baixar as dependências
```bash
git clone https://github.com/joaolfsantos/cpu-cooler-mancer-mystic-G1-go.git
cd cpu-cooler-mancer-mystic-G1-go
go mod tidy
```

## Passo 3 — Identificar o sensor da CPU
```bash
go run check_sensors.go
```
Procure o sensor da sua CPU (geralmente `k10temp` para AMD ou `coretemp` para
Intel). Exemplo de saída:
```
SensorKey:     k10temp_tctl
  Temperatura: 43.00°C
```
Se for diferente de `k10temp_tctl`, edite a linha correspondente em
[`temp_linux.go`](temp_linux.go):
```go
if temp.SensorKey == "k10temp_tctl" { // <-- ajuste aqui
```

## Passo 4 — Permitir acesso ao USB sem `sudo` (regra `udev`)
```bash
sudo nano /etc/udev/rules.d/99-mancer-cooler.rules
```
Cole:
```
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="aa88", ATTRS{idProduct}=="8666", MODE="0666"
```
Aplique e **reconecte o cooler**:
```bash
sudo udevadm control --reload-rules
sudo udevadm trigger
```

## Passo 5 — Compilar e instalar o binário
```bash
go build -o mancer-cooler-monitor .
```
**(Opcional)** teste antes: `./mancer-cooler-monitor` (o display deve atualizar;
`Ctrl+C` para parar). Depois mova para `/usr/local/bin`:

```bash
# Fedora (usa SELinux — restaure o contexto após mover):
sudo mv mancer-cooler-monitor /usr/local/bin/
sudo restorecon -v /usr/local/bin/mancer-cooler-monitor

# Ubuntu:
sudo mv mancer-cooler-monitor /usr/local/bin/
```

## Passo 6 — Instalar o serviço `systemd`
```bash
sudo cp mancer-cooler.service /etc/systemd/system/
sudo nano /etc/systemd/system/mancer-cooler.service   # ajuste o User= se quiser
sudo systemctl daemon-reload
sudo systemctl enable --now mancer-cooler.service
```

### Gerenciando (Linux)
```bash
sudo systemctl status mancer-cooler.service     # status
sudo systemctl restart mancer-cooler.service    # reiniciar
journalctl -u mancer-cooler.service -f          # logs ao vivo
```

---

# Instalação no Windows

No Windows a temperatura é lida do **servidor web do LibreHardwareMonitor**
(JSON em `http://localhost:8085/data.json`). O binário é **Go puro** (não precisa
de compilador C / MinGW).

## Passo 1 — Instalar o Go
Baixe e instale em <https://go.dev/dl/>. Confirme com `go version`.

## Passo 2 — Instalar o LibreHardwareMonitor
Pelo `winget` (recomendado — é onde o instalador procura o LHM):
```powershell
winget install --id LibreHardwareMonitor.LibreHardwareMonitor
```

## Passo 3 — Clonar o repositório
```powershell
git clone https://github.com/joaolfsantos/cpu-cooler-mancer-mystic-G1-go.git
cd cpu-cooler-mancer-mystic-G1-go
```

## Passo 4 — Instalar (compila + cria as tarefas automáticas)
Com o cooler conectado, rode no PowerShell (**não** precisa abrir como admin — o
script pede o UAC apenas na hora de registrar as tarefas):
```powershell
.\install-windows.ps1
```

O `install-windows.ps1` faz tudo:
1. Compila o monitor **sem janela de console** (`-H=windowsgui`) e instala em
   `%LOCALAPPDATA%\MancerCooler`.
2. Configura o LibreHardwareMonitor para **subir com o servidor web já ligado**
   (grava a config dele — não é preciso clicar em nada no LHM).
3. Cria duas tarefas no Agendador, que sobem **a cada logon**, elevadas e
   silenciosas:
   * **`LibreHardwareMonitor-admin`** — o LHM como administrador (necessário para
     ler a temperatura da CPU) expondo o servidor web.
   * **`MancerCoolerMonitor`** — o monitor que lê a temperatura e escreve no
     display; reinicia sozinho se cair.

Pronto — o display passa a exibir a temperatura e tudo volta sozinho após
reiniciar o PC.

### (Opcional) Conferir os nomes dos sensores
Com o LibreHardwareMonitor rodando (após o passo 4):
```powershell
go run list_sensors_windows.go
```
O programa já escolhe automaticamente `CPU Package` (Intel) ou `Core (Tctl/Tdie)`
(AMD). Para usar outro, ajuste a lista `preferredSensors` em
[`temp_windows.go`](temp_windows.go).

### Gerenciando (Windows)
```powershell
# Status das tarefas
Get-ScheduledTask -TaskName MancerCoolerMonitor, LibreHardwareMonitor-admin

# Ver os logs do monitor (reescritos a cada início)
Get-Content "$env:LOCALAPPDATA\MancerCooler\monitor.log"

# Reinstalar após mudar o código
.\install-windows.ps1

# Desinstalar
Unregister-ScheduledTask -TaskName MancerCoolerMonitor -Confirm:$false
Unregister-ScheduledTask -TaskName LibreHardwareMonitor-admin -Confirm:$false
```

---

# Solução de problemas

### Linux
| Sintoma | Causa provável |
|---|---|
| `fatal error: libudev.h: No such file or directory` (ao compilar) | Falta `systemd-devel` (Fedora) / `libudev-dev` (Ubuntu) |
| `Permission denied` em `/dev/hidrawN` | Regra `udev` não aplicada — reveja o Passo 4 e **reconecte o cooler** |
| `Number of warnings: N` nos logs | Apenas aviso do `gopsutil` (sensores secundários) — pode ignorar |
| Nenhum sensor de CPU encontrado | Rode `go run check_sensors.go` e ajuste o `SensorKey` em `temp_linux.go` |

### Windows
| Sintoma | Causa provável |
|---|---|
| `falha ao acessar o servidor web do LibreHardwareMonitor` | LHM fechado, sem o servidor web ativo, ou não está como administrador. Rode `.\install-windows.ps1` de novo |
| `LibreHardwareMonitor nao encontrado` (no instalador) | Instale-o via `winget` (Passo 2) |
| `nenhum sensor de CPU reconhecido` | Rode `go run list_sensors_windows.go` e ajuste `preferredSensors` em `temp_windows.go` |
| `dispositivo HID ... não encontrado` | Cooler desconectado, ou VID/PID diferente de `aa88:8666` |
| A janela preta de console aparece | Recompile com o instalador (`-H=windowsgui` deixa o app sem janela) |

---

## Licença

Este projeto está licenciado sob a Licença MIT.
