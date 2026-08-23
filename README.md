# Streamline

Servidor local para recibir vídeos desde dispositivos, procesarlos automáticamente con [HandBrakeCLI](https://handbrake.fr/) y ofrecerlos para descarga desde una interfaz web.

## Funcionalidades

- Subidas de vídeo reanudables por bloques.
- Cola de procesamiento para evitar ejecuciones simultáneas innecesarias.
- Compresión con H.265 (`x265`), calidad constante 26 y preset `fast`.
- Detección de nuevos vídeos en la carpeta `videos/`.
- Dashboard web con historial, estadísticas y dispositivos.
- Descarga y limpieza de archivos procesados.
- Historial persistente en SQLite, con migración automática desde `history.json`.
- Límite de lectura por solicitud de subida: 1 GiB.

Los formatos detectados son `.mp4`, `.mkv`, `.mov`, `.avi`, `.m4v` y `.webm`.

## Requisitos

- Go `1.26.5` o compatible con la versión indicada en `go.mod`.
- HandBrakeCLI instalado y disponible.
- Windows, Linux y macOS, con vigilancia automática de carpetas mediante `fsnotify`.

## Instalación

Clona o descarga el proyecto y entra en su directorio:

```bash
git clone https://github.com/Bode512/Streamline-go.git
cd Streamline
go mod download
```

Coloca HandBrakeCLI en una de estas ubicaciones:

- Windows: `bin\HandBrakeCLI.exe`
- Linux/macOS: `bin/HandBrakeCLI`
- Una carpeta incluida en `PATH`.
- Una ruta indicada mediante `HANDBRAKE_PATH` o `HANDBRAKECLI_PATH`.

Si usas Linux o macOS, asegúrate de que el binario tenga permisos de ejecución.

## Uso

Inicia el servidor con el puerto predeterminado `8000`:

```bash
go run .
```

También puedes indicar el puerto como argumento:

```bash
go run . 8080
```

O mediante la variable de entorno `PORT`:

```bash
PORT=8080 go run .
```

En Windows PowerShell:

```powershell
$env:PORT = "8080"
go run .
```

Abre en el navegador:

- Panel de subida: `http://localhost:8000/`
- Dashboard: `http://localhost:8000/dashboard`

Al arrancar, el servidor crea automáticamente `videos/` y `convertidos/` en el directorio de trabajo.

## Configuración

| Variable | Descripción |
| --- | --- |
| `PORT` | Puerto HTTP predeterminado. Si se proporciona un puerto como argumento, tiene prioridad. |
| `HANDBRAKE_PATH` | Ruta completa al ejecutable de HandBrakeCLI. |
| `HANDBRAKECLI_PATH` | Ruta alternativa al ejecutable de HandBrakeCLI. |
| `STREAMLINE_INPUT_DIR` | Carpeta de entrada. Predeterminada: `videos`. |
| `STREAMLINE_OUTPUT_DIR` | Carpeta de salida. Predeterminada: `convertidos`. |
| `STREAMLINE_DATABASE` | Ruta de la base SQLite. Predeterminada: `streamline.db`. |

La compresión utiliza estos parámetros de HandBrakeCLI:

```text
-e x265 -q 26 --encoder-preset fast --vfr -B 96 -E av_aac
```

## API HTTP

### Subir un vídeo

`POST /upload?file=<nombre>&deviceId=<id>&deviceInfo=<info>&offset=<offset>&total=<bytes>`

El cuerpo contiene el bloque del archivo. `offset` debe coincidir con el tamaño ya recibido y `total` con el tamaño completo. La respuesta devuelve el siguiente `offset` mientras la subida esté incompleta.

### Consultar el progreso de una subida

`GET /api/upload-offset?file=<nombre>`

### Consultar el historial

`GET /api/history`

Para filtrar por dispositivo:

`GET /api/history?deviceId=<id>`

### Consultar estadísticas y dispositivos

- `GET /api/stats`
- `GET /api/devices`

### Eventos en tiempo real

`GET /api/events`

Abre una conexión Server-Sent Events (`text/event-stream`) para recibir cambios del historial mientras se suben o procesan vídeos. Cada mensaje usa el evento `streamline` y contiene JSON, por ejemplo:

```json
{"type":"history.updated","filename":"clip.mp4","deviceId":"phone-1","status":"ready"}
```

### Código QR para compartir

`GET /api/qr`

Devuelve un PNG de 512x512 con la URL local de Streamline, usando la primera IP IPv4 de red disponible. El puerto puede sobrescribirse con `?port=8080`.

### Red y configuración efectiva

- `GET /api/network` devuelve las interfaces, sus IPs, el puerto activo y la URL para compartir.
- `GET /api/config` devuelve las carpetas, el puerto y la ruta de SQLite en uso.

Estos endpoints están pensados para que un cliente Desktop pueda mostrar el estado local sin replicar la lógica de detección de red.

### Descargar un vídeo procesado

`GET /download?file=<nombre>&deviceId=<id>`

Solo se sirven archivos cuyo estado sea `ready`.

### Controlar trabajos

- `POST /api/jobs/cancel?file=<nombre>` cancela un procesamiento activo.
- `POST /api/jobs/retry?file=<nombre>` vuelve a encolar un vídeo disponible en la carpeta de entrada.

Los fallos de HandBrakeCLI se reintentan automáticamente hasta tres veces. Los estados posibles incluyen `processing`, `ready`, `failed`, `canceled` y `downloaded`.

### Limpiar archivos

`GET /api/cleanup?file=<nombre>&deviceId=<id>`

Para eliminar también la entrada del historial, añade `purge=1`.

## Estructura de datos

```text
videos/       Vídeos recibidos o pendientes de procesar.
convertidos/  Vídeos comprimidos listos para descargar.
streamline.db  Base de datos SQLite con el historial de subidas y conversiones.
history.json   Archivo legado que se importa automáticamente si la base está vacía.
```

Los archivos temporales de subida usan el patrón `.upload_<nombre>.part`. Durante la conversión se crea un archivo `<nombre>.part` y solo se publica el resultado cuando HandBrakeCLI termina correctamente.

## Desarrollo

### Desktop Wails

La aplicación Desktop está en [desktop](desktop). Necesita el Core ejecutándose en otra terminal:

```powershell
go run .
Push-Location desktop
wails dev
Pop-Location
```

Para generar el ejecutable Windows:

```powershell
Push-Location desktop
wails build
Pop-Location
```

El resultado queda en `desktop/build/bin/desktop.exe`.

También puedes regenerar el release autocontenido con `./release.ps1`. El script embebe el Core Go y `bin/HandBrakeCLI.exe` dentro de un único `desktop.exe`; el usuario final no necesita Go, Node.js ni archivos adicionales. El binario de HandBrakeCLI debe distribuirse respetando su licencia.

El Desktop inicia automáticamente el Core Go. En desarrollo busca el `go.mod` del repositorio y ejecuta `go run .`; para una distribución autocontenida, configura `STREAMLINE_CORE_PATH` apuntando a un Core compilado junto al Desktop.

Desde el Desktop puedes seleccionar vídeos del ordenador con **Convertir desde este ordenador**. Se suben por bloques al Core y siguen el mismo worker de HandBrakeCLI que las subidas móviles. Cuando la conversión termina correctamente, el original se mueve a la Papelera de Windows y el resultado queda en `convertidos/`.

Ejecuta las pruebas con:

```bash
go test ./...
```

Para compilar:

```bash
go build -o streamline.exe .
```

El proceso necesita ejecutarse desde la raíz del proyecto para encontrar las carpetas `videos/`, `convertidos/` y la instalación relativa de HandBrakeCLI.

## Notas de seguridad

Los nombres de archivo se validan para impedir traversal de rutas y caracteres peligrosos. El servidor está pensado para ejecutarse en una red local; no incluye autenticación ni autorización, por lo que no debe exponerse directamente a Internet.
