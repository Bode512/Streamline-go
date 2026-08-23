# Streamline Desktop

## About

Aplicación de escritorio Wails v2 con Svelte + TypeScript para operar Streamline.

Requiere Go, Wails CLI v2, Node.js y WebView2 en Windows.

## Live Development

Desde esta carpeta, instala y ejecuta:

```powershell
npm --prefix frontend install
wails dev
```

Arranca el Core desde la raíz del repositorio en otra terminal:

```powershell
go run .
```

La aplicación usa `http://127.0.0.1:8000` por defecto. También puedes conectar una URL LAN desde el campo de servidor.

Al iniciar la aplicación, Wails arranca automáticamente el Core Go y espera su health check. Al cerrar la ventana, también detiene el proceso del Core. Para usar un Core ya compilado, define `STREAMLINE_CORE_PATH` con la ruta del ejecutable.

El panel muestra miniaturas de los vídeos listos, peso original, duración detectada por el reproductor, estado, historial, QR para móviles y actividad en tiempo real.

## Building

Para construir el paquete distribuible:

```powershell
wails build
```

El ejecutable Windows se genera en `build/bin/desktop.exe`.

La consola incluye importación local: selecciona vídeos desde el ordenador y pulsa **Convertir**. El archivo se envía al Core por bloques, se procesa con HandBrakeCLI y el original se mueve a la Papelera al finalizar correctamente.

La interfaz consume `/api/stats`, `/api/history`, `/api/events`, `/api/qr`, `/api/jobs/cancel` y `/api/jobs/retry`.
