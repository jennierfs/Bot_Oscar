// ============================================
// Bot Oscar - Punto de entrada principal
// Inicializa todos los servicios y arranca el servidor HTTP
// ============================================
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bot-oscar/internal/ai"
	"bot-oscar/internal/api"
	"bot-oscar/internal/cache"
	"bot-oscar/internal/config"
	"bot-oscar/internal/db"
	"bot-oscar/internal/market"
	"bot-oscar/internal/trading"
)

func main() {
	log.Println("🤖 Bot Oscar - Iniciando...")

	// Cargar configuración desde variables de entorno
	cfg := config.Load()
	log.Println("✅ Configuración cargada")

	// Conectar a PostgreSQL
	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("❌ Error conectando a PostgreSQL: %v", err)
	}
	defer database.Close()
	log.Println("✅ Conectado a PostgreSQL")

	// Conectar a Redis
	redisCache, err := cache.Connect(cfg)
	if err != nil {
		log.Fatalf("❌ Error conectando a Redis: %v", err)
	}
	defer redisCache.Close()
	log.Println("✅ Conectado a Redis")

	// Crear proveedor de datos de mercado
	// Prioridad: Yahoo Finance (gratis) → Alpha Vantage (si hay key) → Demo
	provider := market.NewProvider(cfg.AlphaVantageAPIKey, redisCache)
	log.Println("✅ Proveedor de datos de mercado inicializado (Yahoo Finance + fallback)")

	// Crear motor de trading con lógica de trader profesional
	engine := trading.NewEngine(database, redisCache, provider, cfg)
	log.Println("✅ Motor de trading inicializado")

	// Crear cliente de DeepSeek AI para análisis inteligente
	deepseekClient := ai.NewDeepSeekClient(cfg.DeepSeekAPIKey)
	if deepseekClient.IsConfigured() {
		log.Println("✅ DeepSeek AI configurado - Análisis inteligente disponible")
	} else {
		log.Println("⚠️ DeepSeek AI no configurado - Configura DEEPSEEK_API_KEY en .env")
	}

	// Crear router de la API REST
	router := api.NewRouter(database, redisCache, engine, deepseekClient)

	// Configurar servidor HTTP
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Iniciar servidor en una goroutine
	go func() {
		log.Printf("🚀 Servidor API escuchando en puerto %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Error en servidor HTTP: %v", err)
		}
	}()

	// Esperar señal de cierre (Ctrl+C o SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("⏳ Apagando Bot Oscar...")

	// Contexto con timeout para apagado graceful
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Detener motor de trading
	engine.Stop()
	log.Println("✅ Motor de trading detenido")

	// Apagar servidor HTTP
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("⚠️ Error en apagado del servidor: %v", err)
	}

	log.Println("👋 Bot Oscar apagado correctamente")
}
