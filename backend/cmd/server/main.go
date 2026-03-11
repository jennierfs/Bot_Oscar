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
	"bot-oscar/internal/candles"
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
	// Prioridad: Twelve Data (principal) → Yahoo Finance (respaldo) → Demo
	var provider market.Provider
	var tdProvider *market.TwelveDataProvider
	if cfg.TwelveDataAPIKey != "" {
		provider, tdProvider = market.NewProviderWithTwelveData(cfg.TwelveDataAPIKey, redisCache)
		log.Println("✅ Proveedor de datos: Twelve Data (principal) + Yahoo Finance (respaldo)")
	} else {
		provider = market.NewProvider(cfg.AlphaVantageAPIKey, redisCache)
		log.Println("✅ Proveedor de datos: Yahoo Finance (sin Twelve Data API key)")
	}

	// Crear motor de trading con lógica de trader profesional
	engine := trading.NewEngine(database, redisCache, provider, cfg)
	log.Println("✅ Motor de trading inicializado")

	// Crear descargador de velas históricas (requiere Twelve Data)
	var candleDownloader *candles.CandleDownloader
	var realtimeUpdater *candles.RealtimeUpdater
	if tdProvider != nil {
		candleDownloader = candles.NewCandleDownloader(tdProvider, database)
		realtimeUpdater = candles.NewRealtimeUpdater(tdProvider, database)
		// Asegurar que la tabla de velas existe en la BD
		if err := database.EnsureCandlesTable(context.Background()); err != nil {
			log.Printf("⚠️ Error creando tabla de velas: %v", err)
		}
		log.Println("✅ Descargador de velas históricas configurado")
		log.Println("✅ Actualizador de velas en tiempo real configurado")
	}

	// Crear cliente de DeepSeek AI para análisis inteligente
	deepseekClient := ai.NewDeepSeekClient(cfg.DeepSeekAPIKey)
	if deepseekClient.IsConfigured() {
		log.Println("✅ DeepSeek AI configurado - Análisis inteligente disponible")
	} else {
		log.Println("⚠️ DeepSeek AI no configurado - Configura DEEPSEEK_API_KEY en .env")
	}

	// Crear proveedor de sentimiento de mercado (Analyst Ratings + Short Interest)
	sentimentProvider := market.NewSentimentProvider(redisCache)
	log.Println("✅ Proveedor de sentimiento de mercado configurado (Yahoo Finance)")

	// Crear router de la API REST
	router := api.NewRouter(database, redisCache, engine, deepseekClient, candleDownloader, realtimeUpdater, sentimentProvider)

	// Configurar servidor HTTP
	// Timeouts amplios porque Yahoo Finance + cálculo de 300 velas + DeepSeek toman tiempo
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Iniciar servidor en una goroutine
	go func() {
		log.Printf("🚀 Servidor API escuchando en puerto %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Error en servidor HTTP: %v", err)
		}
	}()

	// Auto-iniciar actualizador de velas en tiempo real
	if realtimeUpdater != nil {
		if err := realtimeUpdater.Start(); err != nil {
			log.Printf("⚠️ Error iniciando actualizador en tiempo real: %v", err)
		} else {
			log.Println("🔴 LIVE — Velas en tiempo real activadas automáticamente")
		}
	}

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

	// Detener actualizador en tiempo real
	if realtimeUpdater != nil {
		realtimeUpdater.Stop()
		log.Println("✅ Actualizador en tiempo real detenido")
	}

	// Apagar servidor HTTP
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("⚠️ Error en apagado del servidor: %v", err)
	}

	log.Println("👋 Bot Oscar apagado correctamente")
}
