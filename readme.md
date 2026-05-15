
# Monitor de Display para o Water Cooler Mancer Mystic G1 (Linux)

Este é um driver/serviço simples, escrito em Go, para controlar o display de temperatura do Water Cooler Mancer Mystic G1 em sistemas Linux. Ele lê a temperatura da CPU em tempo real e a exibe no display do cooler.

O programa é projetado para rodar como um serviço `systemd` em segundo plano, iniciando automaticamente com o sistema.

## Funcionalidades

*   Exibe a temperatura da CPU em tempo real no display do cooler.
*   Roda como um serviço `systemd` leve e de baixo consumo de recursos.
*   Inicia automaticamente no boot do sistema.
*   Configuração segura que não requer a execução do serviço como `root`.

## Sistemas testados

A instalação foi documentada e testada nos seguintes sistemas:

*   **Fedora KDE**
*   **Fedora GNOME**
*   **Ubuntu GNOME**

O programa funciona de forma idêntica em qualquer ambiente de desktop (KDE, GNOME, etc.) — o ambiente gráfico não influencia o funcionamento. As únicas diferenças entre os sistemas estão nos **gerenciadores de pacotes** (`dnf` no Fedora, `apt` no Ubuntu) e nos **nomes dos pacotes** de dependência.

---

## Passo 1: Instalar as Dependências

O programa usa `cgo` para compilar, portanto precisa de um compilador C, das bibliotecas de comunicação HID/USB e do `lm-sensors` para a leitura de temperatura.

Escolha o bloco correspondente ao seu sistema.

### Fedora (KDE ou GNOME)

```bash
sudo dnf install golang gcc hidapi-devel systemd-devel lm_sensors
```

| Pacote | Para que serve |
|---|---|
| `golang` | Compilador da linguagem Go |
| `gcc` | Compilador C, exigido pelo `cgo` |
| `hidapi-devel` | Biblioteca de comunicação com dispositivos HID |
| `systemd-devel` | Fornece o `libudev.h`, exigido pela `go-hid` |
| `lm_sensors` | Leitura dos sensores de temperatura |

### Ubuntu (GNOME)

```bash
sudo apt-get update
sudo apt-get install golang-go build-essential libhidapi-dev libudev-dev lm-sensors
```

| Pacote | Para que serve |
|---|---|
| `golang-go` | Compilador da linguagem Go |
| `build-essential` | Compilador C (`gcc`), exigido pelo `cgo` |
| `libhidapi-dev` | Biblioteca de comunicação com dispositivos HID |
| `libudev-dev` | Fornece o `libudev.h`, exigido pela `go-hid` |
| `lm-sensors` | Leitura dos sensores de temperatura |

> **Opcional (todos os sistemas):** rode `sudo sensors-detect` e responda `YES` às perguntas para garantir que todos os módulos de sensores estejam carregados. Não é obrigatório, mas ajuda a expor mais sensores.

---

## Passo 2: Clonar o Repositório

```bash
git clone https://github.com/joaolfsantos/cpu-cooler-mancer-mystic-G1-go.git
cd cpu-cooler-mancer-mystic-G1-go
```

---

## Passo 3: Baixar as Dependências do Go

```bash
go mod tidy
```

---

## Passo 4: Identificar o Sensor de CPU Correto

Os nomes dos sensores de temperatura variam de sistema para sistema. Para descobrir qual usar, rode o script de diagnóstico incluído:

```bash
go run check_sensors.go
```

Você verá a lista de todos os sensores disponíveis. Procure o que corresponde à sua CPU — geralmente contém `k10temp` (AMD) ou `coretemp` (Intel).

**Exemplo de saída:**
```
--- Sensores Encontrados ---
SensorKey:     nvme_composite
  Temperatura: 31.85°C
------------------------------
SensorKey:     k10temp_tctl
  Temperatura: 43.00°C
------------------------------
```

No exemplo acima, o sensor correto é `k10temp_tctl`.

> **Nota:** o `gopsutil` pode imprimir um aviso como `Aviso ao ler sensores: Number of warnings: 3`. Isso é **normal** — significa apenas que alguns sensores do sistema (ex: pentes de memória RAM) não puderam ser lidos. Os demais sensores continuam sendo listados normalmente, e o programa principal lida com isso automaticamente.

Abra o `main.go` e edite esta linha, substituindo o nome do sensor pelo que você encontrou:

```go
// Encontre esta linha no main.go
if temp.SensorKey == "k10temp_tctl" { // <-- SUBSTITUA AQUI
    return int(temp.Temperature), nil
}
```

---

## Passo 5: Configurar Permissões USB (Regra `udev`)

Para permitir que o programa acesse o cooler **sem `sudo`**, criamos uma regra `udev`. O dispositivo é acessado pelo nó `/dev/hidrawN`, então a regra precisa mirar o subsistema `hidraw`.

1.  Crie o arquivo de regra:
    ```bash
    sudo nano /etc/udev/rules.d/99-mancer-cooler.rules
    ```
2.  Cole o seguinte conteúdo:
    ```
    KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="aa88", ATTRS{idProduct}=="8666", MODE="0666"
    ```
3.  Salve (`Ctrl+X`, `Y`, `Enter`) e aplique as regras:
    ```bash
    sudo udevadm control --reload-rules
    sudo udevadm trigger
    ```
4.  Desconecte e reconecte o cooler para que as novas permissões sejam aplicadas.

> Esta regra é a mesma para Fedora e Ubuntu.

---

## Passo 6: Compilar e Instalar o Programa

1.  Compile o binário:
    ```bash
    go build -o mancer-cooler-monitor .
    ```

2.  **(Teste rápido — opcional)** Antes de instalar como serviço, você pode testar a execução:
    ```bash
    ./mancer-cooler-monitor
    ```
    Se a regra `udev` do Passo 5 foi aplicada corretamente, o display do cooler deve passar a exibir a temperatura. Pressione `Ctrl+C` para parar. (Se ainda der "Permission denied", confira o Passo 5 e reconecte o cooler.)

3.  Mova o binário para `/usr/local/bin`:

    **Fedora (KDE ou GNOME):** o Fedora usa SELinux, então é necessário restaurar o contexto de segurança do arquivo após movê-lo:
    ```bash
    sudo mv mancer-cooler-monitor /usr/local/bin/
    sudo restorecon -v /usr/local/bin/mancer-cooler-monitor
    ```

    **Ubuntu (GNOME):** o Ubuntu não usa SELinux por padrão, então basta mover:
    ```bash
    sudo mv mancer-cooler-monitor /usr/local/bin/
    ```

---

## Passo 7: Instalar e Configurar o Serviço `systemd`

1.  Copie o arquivo de serviço para o diretório do `systemd`:
    ```bash
    sudo cp mancer-cooler.service /etc/systemd/system/
    ```

2.  **(Passo Crucial)** Edite o arquivo de serviço para definir o usuário correto:
    ```bash
    sudo nano /etc/systemd/system/mancer-cooler.service
    ```
    Encontre a linha `User=seu_usuario_normal` e **substitua `seu_usuario_normal` pelo seu nome de usuário do Linux** (você pode descobri-lo com o comando `whoami`). Salve e feche o arquivo.

---

## Passo 8: Habilitar e Iniciar o Serviço

```bash
sudo systemctl daemon-reload
sudo systemctl enable mancer-cooler.service
sudo systemctl start mancer-cooler.service
```

Pronto! Seu cooler agora deve estar exibindo a temperatura da CPU em tempo real, iniciando automaticamente a cada boot.

---

## Gerenciando o Serviço

*   **Verificar o status:**
    ```bash
    sudo systemctl status mancer-cooler.service
    ```
*   **Parar o serviço:**
    ```bash
    sudo systemctl stop mancer-cooler.service
    ```
*   **Reiniciar o serviço:**
    ```bash
    sudo systemctl restart mancer-cooler.service
    ```
*   **Ver os logs do serviço:**
    ```bash
    journalctl -u mancer-cooler.service -f
    ```

---

## Solução de Problemas

**`fatal error: libudev.h: No such file or directory` ao compilar**
Faltam as dependências de desenvolvimento. Reveja o Passo 1 (`systemd-devel` no Fedora, `libudev-dev` no Ubuntu).

**`Permission denied` em `/dev/hidrawN` ao executar**
A regra `udev` não foi aplicada. Reveja o Passo 5 e **reconecte o cooler** fisicamente. Para um teste imediato você pode rodar com `sudo ./mancer-cooler-monitor`, mas o ideal é a regra `udev` para que o serviço rode sem privilégios.

**`Number of warnings: N` nos logs**
Isso é apenas um aviso do `gopsutil` — alguns sensores secundários (ex: memória RAM) não puderam ser lidos. O programa ignora esses sensores e continua usando o da CPU normalmente. Não é um erro.

**O display não atualiza / nenhum sensor de CPU encontrado**
Rode `go run check_sensors.go` novamente e confirme que o nome do sensor configurado em `main.go` corresponde exatamente a um sensor da lista.

---

## Licença

Este projeto está licenciado sob a Licença MIT.
