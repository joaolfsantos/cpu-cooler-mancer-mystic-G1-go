
# Monitor de Display para o Water Cooler Mancer Mystic G1 (Linux)

Este é um driver/serviço simples, escrito em Go, para controlar o display de temperatura do Water Cooler Mancer Mystic G1 em sistemas Linux. Ele lê a temperatura da CPU em tempo real e a exibe no display do cooler.

O programa é projetado para rodar como um serviço `systemd` em segundo plano, iniciando automaticamente com o sistema.

## Funcionalidades

*   Exibe a temperatura da CPU em tempo real no display do cooler.
*   Roda como um serviço `systemd` leve e de baixo consumo de recursos.
*   Inicia automaticamente no boot do sistema.
*   Configuração segura que não requer a execução do serviço como `root`.

## Pré-requisitos

Antes de começar, certifique-se de que você tem os seguintes pré-requisitos instalados no seu sistema (os exemplos são para Ubuntu/Debian):

1.  **Go (Linguagem de Programação):**
    ```bash
    sudo apt-get update
    sudo apt-get install golang-go
    ```
2.  **Biblioteca `libhidapi-dev`:** Essencial para a comunicação com dispositivos HID.
    ```bash
    sudo apt-get install libhidapi-dev
    ```
3.  **Um sistema Linux com `systemd`:** Padrão na maioria das distribuições modernas como Ubuntu, Fedora, Debian, etc.

## Instalação e Configuração

Siga estes passos para configurar, compilar e instalar o serviço.

### Passo 1: Clonar o Repositório

Primeiro, clone este repositório para a sua máquina local.

```bash
git clone https://github.com/joaolfsantos/cpu-cooler-mancer-mystic-G1-go.git
cd cpu-cooler-mancer-mystic-G1-go
```

### Passo 2: Instalar as Dependências do Go

Navegue até o diretório do projeto e baixe as dependências do Go necessárias.

```bash
go mod tidy
```

### Passo 3: Identificar o Sensor de CPU Correto

Os nomes dos sensores de temperatura variam de sistema para sistema. Para garantir que o programa leia a temperatura correta, execute o script de diagnóstico incluído:

```bash
go run check_sensors.go
```

Você verá uma lista de todos os sensores disponíveis. Procure o que corresponde à sua CPU (geralmente contém `k10temp` para AMD ou `coretemp` para Intel).

**Exemplo de saída:**
```
--- Sensores Encontrados ---
SensorKey:     k10temp_tctl
  Temperatura: 41.75°C
------------------------------
SensorKey:     nvme_composite
  Temperatura: 29.85°C
------------------------------
```
No exemplo acima, o sensor correto é `k10temp_tctl`.

Abra o arquivo `main.go` e edite esta linha, substituindo o nome do sensor pelo que você encontrou:

```go
// Encontre esta linha no main.go
if temp.SensorKey == "k10temp_tctl" { // <-- SUBSTITUA AQUI
    return int(temp.Temperature), nil
}
```

### Passo 4: Configurar Permissões USB (Regra `udev`)

Para permitir que o programa acesse o cooler sem precisar de `sudo`, vamos criar uma regra `udev`.

1.  Crie o arquivo de regra:
    ```bash
    sudo nano /etc/udev/rules.d/99-mancer-cooler.rules
    ```
2.  Cole o seguinte conteúdo no arquivo:
    ```
    SUBSYSTEM=="usb", ATTR{idVendor}=="aa88", ATTR{idProduct}=="8666", MODE="0666"
    ```
3.  Salve (`Ctrl+X`, `Y`, `Enter`) e aplique as regras:
    ```bash
    sudo udevadm control --reload-rules
    sudo udevadm trigger
    ```
4.  Desconecte e reconecte o cooler para que as novas permissões sejam aplicadas.

### Passo 5: Compilar e Instalar o Programa

Agora, compile o programa e mova o binário para um diretório do sistema.

1.  Compile o binário:
    ```bash
    go build -o mancer-cooler-monitor .
    ```
2.  Mova o binário para `/usr/local/bin`:
    ```bash
    sudo mv mancer-cooler-monitor /usr/local/bin/
    ```

### Passo 6: Instalar e Configurar o Serviço `systemd`

1.  Copie o arquivo de serviço para o diretório do `systemd`:
    ```bash
    sudo cp mancer-cooler.service /etc/systemd/system/
    ```

2.  **(Passo Crucial)** Edite o arquivo de serviço para definir o usuário correto.
    ```bash
    sudo nano /etc/systemd/system/mancer-cooler.service
    ```
    Encontre a linha `User=seu_usuario_normal` e **substitua `seu_usuario_normal` pelo seu nome de usuário do Linux**. Salve e feche o arquivo.

### Passo 7: Habilitar e Iniciar o Serviço

Finalmente, habilite o serviço para que ele inicie no boot e inicie-o pela primeira vez.

1.  Recarregue o `systemd` para que ele reconheça o novo serviço:
    ```bash
    sudo systemctl daemon-reload
    ```
2.  Habilite o serviço:
    ```bash
    sudo systemctl enable mancer-cooler.service
    ```
3.  Inicie o serviço:
    ```bash
    sudo systemctl start mancer-cooler.service
    ```

Pronto! Seu cooler agora deve estar exibindo a temperatura da CPU em tempo real.

## Gerenciando o Serviço

Você pode usar os seguintes comandos para gerenciar o serviço:

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

## Licença

Este projeto está licenciado sob a Licença MIT.
```