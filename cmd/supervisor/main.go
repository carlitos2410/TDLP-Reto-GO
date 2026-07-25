package main

import (
	"context"
	"flag"
	"log"

	"supervisor-procesos/internal/api"
	"supervisor-procesos/internal/config"
	"supervisor-procesos/internal/signal"
	"supervisor-procesos/internal/supervisor"
)

func main() {
	configPath := flag.String("config", "configs/example.yaml", "ruta al archivo de configuración YAML")
	apiListen := flag.String("api-listen", "", "dirección HTTP del API (override de api_listen en YAML)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("error cargando configuración: %v", err)
	}

	listenAddr := cfg.APIListenAddress()
	if *apiListen != "" {
		listenAddr = *apiListen
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sup := supervisor.New(cfg)
	notifier := signal.NewNotifier()

	go notifier.Run(ctx, cancel, func() {
		reloaded, err := config.Load(*configPath)
		if err != nil {
			log.Printf("recarga fallida: %v", err)
			return
		}
		if err := sup.Reload(reloaded); err != nil {
			log.Printf("aplicar recarga fallida: %v", err)
			return
		}
		log.Printf("configuración recargada correctamente")
	})

	go func() {
		if err := api.NewServer(sup, listenAddr).Run(ctx); err != nil {
			log.Printf("servidor API terminó con error: %v", err)
		}
	}()

	log.Printf("supervisor iniciado con %d proceso(s), grace_period=%s, api=%s",
		len(cfg.Processes), cfg.GracePeriod(), listenAddr)
	sup.Run(ctx)
	log.Printf("supervisor finalizado: apagado limpio completado")
}
