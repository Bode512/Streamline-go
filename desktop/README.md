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

## Building

Para construir el paquete distribuible:

```powershell
wails build
```

El ejecutable Windows se genera en `build/bin/desktop.exe`.

La interfaz consume `/api/stats`, `/api/history`, `/api/events`, `/api/qr`, `/api/jobs/cancel` y `/api/jobs/retry`.
