package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"Streamline/internal/config"
)

func runApplication() {
	runtimeConfig = config.Load()
	if err := runtimeConfig.Validate(); err != nil {
		log.Fatalf("configuracion invalida: %v", err)
	}
	puerto := runtimeConfig.Port
	if len(os.Args) > 1 {
		puerto = os.Args[1]
	}

	configurarConsola()

	if err := crearDirectorio(runtimeConfig.InputDir); err != nil {
		log.Fatalf("error creando directorio videos: %v", err)
	}
	if err := crearDirectorio(runtimeConfig.OutputDir); err != nil {
		log.Fatalf("error creando directorio convertidos: %v", err)
	}

	miCola = NuevaCola(50)
	EnqueueFunc = miCola.Enqueue

	directorioSalida = runtimeConfig.OutputDir
	if err := RecorrerDirectorios(runtimeConfig.InputDir); err != nil {
		log.Printf("aviso durante el escaneo inicial de videos: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		if err := VigilarDirectorio(ctx, runtimeConfig.InputDir, miCola); err != nil && ctx.Err() == nil {
			log.Printf("aviso del watcher: %v", err)
		}
	}()
	go ProcesarCola(ctx, runtimeConfig.OutputDir)

	if err := IniciarServidorMongoose(puerto); err != nil {
		log.Fatalf("error del servidor: %v", err)
	}

	if historyStore != nil {
		_ = historyStore.Close()
	}
	miCola.Liberar()
}
