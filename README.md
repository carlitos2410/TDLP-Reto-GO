# Supervisor de procesos en Go

Proyecto académico del curso **Taller de Lenguajes de Programación**. Implementa un supervisor capaz de iniciar, vigilar, detener y reiniciar procesos hijos definidos mediante un archivo YAML.

Su propósito es demostrar el manejo de procesos del sistema operativo, concurrencia, señales, archivos, configuración, pruebas y servicios HTTP utilizando Go.

## ¿Qué problema resuelve?

Un programa puede cerrarse inesperadamente, bloquearse o necesitar reiniciarse. El supervisor centraliza la administración de varios programas y aplica automáticamente una política de ejecución para cada uno.

El sistema permite:

- Iniciar varios procesos hijos.
- Capturar `stdout` y `stderr` en archivos de log.
- Consultar el estado de cada proceso.
- Reiniciar procesos según una política configurable.
- Esperar antes de cada reintento mediante backoff exponencial.
- Limitar la cantidad de reintentos.
- Detener los procesos de forma ordenada.
- Controlar procesos mediante una API HTTP.
- Recargar la configuración con `SIGHUP` en sistemas Unix.

## Funcionamiento general

1. El programa lee `configs/example.yaml`.
2. Valida los campos, valores y procesos configurados.
3. Crea un worker independiente para cada proceso.
4. Cada worker ejecuta y vigila su proceso hijo.
5. Cuando un proceso termina, el worker aplica su política de reinicio.
6. El estado puede consultarse o controlarse mediante la API HTTP.
7. Al recibir `Ctrl+C`, se cancela el contexto principal y se realiza un apagado ordenado.

## Arquitectura

```text
cmd/supervisor
      |
      +-- internal/config      Lee y valida el YAML
      +-- internal/supervisor  Coordina workers, estados y reinicios
      +-- internal/process     Ejecuta procesos hijos con os/exec
      +-- internal/signal      Gestiona señales del sistema operativo
      +-- internal/api         Expone la API HTTP
```

Cada proceso es administrado por un worker en una goroutine. El proyecto utiliza:

- `context.Context` para propagar cancelaciones.
- `sync.RWMutex` para proteger estados compartidos.
- `sync.WaitGroup` para esperar la finalización de goroutines.
- Canales para coordinar la terminación de procesos.
- `os/exec` para ejecutar programas externos.
- `net/http` para la API.

## Políticas de reinicio

| Política | Comportamiento |
|---|---|
| `always` | Reinicia el proceso siempre que termine. |
| `on-failure` | Reinicia únicamente cuando termina con error. |
| `never` | Ejecuta el proceso una vez y no lo reinicia. |

## Backoff exponencial

El backoff evita una tormenta de reinicios cuando un proceso falla repetidamente.

```text
espera = min(initial_seconds × factor^fallos, max_seconds)
```

Por ejemplo, con valor inicial de 1 segundo, factor 2 y máximo 30, las esperas aumentan aproximadamente así:

```text
1 s, 2 s, 4 s, 8 s, 16 s, 30 s
```

Una ejecución exitosa reinicia el contador de fallos.

## Estados de los procesos

| Estado | Significado |
|---|---|
| `running` | El proceso hijo está ejecutándose. |
| `backoff` | Está esperando antes de reiniciar. |
| `stopped` | Terminó o fue detenido y no se reiniciará. |
| `failed` | Superó el máximo de reintentos permitidos. |

## Procesos de demostración

| Proceso | Comportamiento |
|---|---|
| `ticker` | Permanece activo y escribe mensajes periódicamente. |
| `greeter` | Muestra saludos y termina correctamente. |
| `one-shot` | Ejecuta una tarea una sola vez. |
| `flaky` | Simula fallos para demostrar reinicios y backoff. |

## Requisitos

- Go 1.22 o superior.
- PowerShell para los comandos de Windows mostrados a continuación.

## Compilar los ejemplos en Windows

Desde la raíz del proyecto:

```powershell
New-Item -ItemType Directory -Force -Path bin

go build -o bin/ticker.exe ./examples/ticker
go build -o bin/greeter.exe ./examples/greeter
go build -o bin/oneshot.exe ./examples/oneshot
go build -o bin/flaky.exe ./examples/flaky
```

Los ejemplos se compilan antes de ejecutar el supervisor para que este controle directamente cada binario y no deje procesos huérfanos creados por `go run`.

## Ejecutar

```powershell
go run ./cmd/supervisor -config configs/example.yaml
```

Para detener el sistema de forma ordenada:

```text
Ctrl+C
```

## API HTTP

La dirección predeterminada es:

```text
http://localhost:8080
```

| Método | Ruta | Acción |
|---|---|---|
| `GET` | `/api/v1/processes` | Lista todos los procesos. |
| `GET` | `/api/v1/processes/{name}` | Consulta un proceso. |
| `POST` | `/api/v1/processes/{name}/start` | Inicia un proceso. |
| `POST` | `/api/v1/processes/{name}/stop` | Detiene un proceso. |
| `POST` | `/api/v1/processes/{name}/restart` | Reinicia un proceso. |

Ejemplos desde PowerShell:

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/processes

Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/api/v1/processes/ticker/stop

Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/api/v1/processes/ticker/start
```

## Logs

La salida estándar y los errores se guardan en `logs/`. Por ejemplo:

```text
logs/ticker.stdout.log
logs/ticker.stderr.log
logs/flaky.stdout.log
logs/flaky.stderr.log
```

## Pruebas y análisis

```powershell
go test ./...
go vet ./...
```

El detector de condiciones de carrera requiere CGO y GCC. En Windows se recomienda utilizar WSL:

```bash
go test -race ./...
```

## Estructura principal

```text
supervisor-procesos/
├── cmd/supervisor/       Punto de entrada
├── configs/              Configuración YAML
├── examples/             Procesos de demostración
├── internal/api/         API HTTP
├── internal/config/      Lectura y validación
├── internal/process/     Ejecución de procesos hijos
├── internal/signal/      Señales del sistema
├── internal/supervisor/  Workers, estados y reinicios
├── docs/                 Documentación técnica
├── go.mod
└── README.md
```

## Resumen para exposición

> Este proyecto implementa un supervisor de procesos en Go. Lee desde YAML qué programas debe ejecutar, crea un worker concurrente para cada uno, captura sus logs y aplica políticas de reinicio. El backoff exponencial evita reinicios excesivos, mientras que el uso de contextos, mutex y grupos de espera permite controlar la concurrencia y realizar un apagado ordenado. Además, una API HTTP permite consultar, iniciar, detener y reiniciar procesos.

## Documentación adicional

- `BITACORA_APRENDIZAJE.md`: evolución y decisiones del desarrollo.
- `CONTEXTO_PROYECTO_IA.md`: explicación detallada de arquitectura y código.
- `ESTRUCTURA_PROYECTO.md`: descripción de archivos y carpetas.
- `GUIÓN_EXPOSICION.md`: guía para la demostración.
- `docs/ESTADOS_Y_BACKOFF.md`: máquina de estados y backoff.
