// ============================================
// Bot Oscar - Router de la API REST
// Define todas las rutas y middleware
// ============================================
package api

import (
	"net/http"

	"bot-oscar/internal/ai"
	"bot-oscar/internal/cache"
	"bot-oscar/internal/candles"
	"bot-oscar/internal/db"
	"bot-oscar/internal/market"
	"bot-oscar/internal/trading"
)

// Server contiene las dependencias compartidas por los handlers
type Server struct {
	db         *db.Database
	cache      *cache.Cache
	engine     *trading.Engine
	deepseek   *ai.DeepSeekClient
	downloader *candles.CandleDownloader
	realtime   *candles.RealtimeUpdater
	sentiment  *market.SentimentProvider
}

// NewRouter crea el router HTTP con todas las rutas de la API
func NewRouter(database *db.Database, redisCache *cache.Cache, engine *trading.Engine, deepseekClient *ai.DeepSeekClient, candleDownloader *candles.CandleDownloader, realtimeUpdater *candles.RealtimeUpdater, sentimentProvider *market.SentimentProvider) http.Handler {
	mux := http.NewServeMux()

	server := &Server{
		db:         database,
		cache:      redisCache,
		engine:     engine,
		deepseek:   deepseekClient,
		downloader: candleDownloader,
		realtime:   realtimeUpdater,
		sentiment:  sentimentProvider,
	}

	// --- Rutas de salud ---
	mux.HandleFunc("GET /api/salud", server.handleHealth)

	// --- Rutas de activos ---
	mux.HandleFunc("GET /api/activos", server.handleGetAssets)
	mux.HandleFunc("GET /api/activos/{id}/precios", server.handleGetPrices)

	// --- Rutas de señales ---
	mux.HandleFunc("GET /api/senales", server.handleGetSignals)

	// --- Rutas de operaciones ---
	mux.HandleFunc("GET /api/operaciones", server.handleGetOperations)

	// --- Rutas del portafolio ---
	mux.HandleFunc("GET /api/portafolio", server.handleGetPortfolio)

	// --- Rutas de configuración ---
	mux.HandleFunc("GET /api/configuracion", server.handleGetConfig)
	mux.HandleFunc("POST /api/configuracion", server.handleUpdateConfig)

	// --- Rutas del bot ---
	mux.HandleFunc("GET /api/bot/estado", server.handleGetBotStatus)
	mux.HandleFunc("POST /api/bot/iniciar", server.handleStartBot)
	mux.HandleFunc("POST /api/bot/detener", server.handleStopBot)

	// --- Rutas de indicadores ---
	mux.HandleFunc("GET /api/indicadores/{simbolo}", server.handleGetIndicators)

	// --- Ruta de Índice de Miedo & Codicia por activo ---
	mux.HandleFunc("GET /api/feargreed/{simbolo}", server.handleGetFearGreed)

	// --- Ruta de Sentimiento de Mercado (Analyst Ratings + Short Interest) ---
	mux.HandleFunc("GET /api/sentimiento/{simbolo}", server.handleGetSentiment)

	// --- Rutas de IA (DeepSeek) ---
	mux.HandleFunc("POST /api/ia/senal/{simbolo}", server.handleGenerateAISignal)

	// --- Rutas de velas históricas ---
	mux.HandleFunc("POST /api/velas/descargar", server.handleDownloadCandles)
	mux.HandleFunc("GET /api/velas/estado", server.handleGetCandleStatus)
	mux.HandleFunc("GET /api/velas/stats", server.handleGetCandleStats)
	mux.HandleFunc("GET /api/velas/{simbolo}/{timeframe}", server.handleGetCandles)

	// --- Rutas de actualización en tiempo real ---
	mux.HandleFunc("POST /api/velas/realtime/iniciar", server.handleStartRealtime)
	mux.HandleFunc("POST /api/velas/realtime/detener", server.handleStopRealtime)
	mux.HandleFunc("GET /api/velas/realtime/estado", server.handleGetRealtimeStatus)

	// Aplicar middleware CORS para desarrollo
	return corsMiddleware(mux)
}

// corsMiddleware permite peticiones cross-origin (necesario para desarrollo local)
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Responder inmediatamente a preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
