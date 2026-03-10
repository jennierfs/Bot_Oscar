// ============================================
// Bot Oscar - Handlers de la API REST
// Cada handler procesa una petición HTTP específica
// ============================================
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// ============================================
// Helpers para respuestas JSON
// ============================================

// jsonResponse escribe una respuesta JSON con el código de estado indicado
func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error codificando respuesta JSON: %v", err)
	}
}

// jsonError escribe una respuesta de error en formato JSON
func jsonError(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]string{"error": message})
}

// ============================================
// Handler: Salud del sistema
// ============================================

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "Bot Oscar API",
	})
}

// ============================================
// Handlers: Activos
// ============================================

// handleGetAssets devuelve la lista de activos activos
func (s *Server) handleGetAssets(w http.ResponseWriter, r *http.Request) {
	assets, err := s.db.GetActiveAssets(r.Context())
	if err != nil {
		log.Printf("Error obteniendo activos: %v", err)
		jsonError(w, http.StatusInternalServerError, "Error obteniendo activos")
		return
	}
	jsonResponse(w, http.StatusOK, assets)
}

// handleGetPrices devuelve los precios históricos de un activo
func (s *Server) handleGetPrices(w http.ResponseWriter, r *http.Request) {
	// Obtener ID del activo desde la URL
	idStr := r.PathValue("id")
	assetID, err := strconv.Atoi(idStr)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "ID de activo inválido")
		return
	}

	// Obtener límite opcional (por defecto 200 velas)
	limit := 200
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	prices, err := s.db.GetPrices(r.Context(), assetID, limit)
	if err != nil {
		log.Printf("Error obteniendo precios del activo %d: %v", assetID, err)
		jsonError(w, http.StatusInternalServerError, "Error obteniendo precios")
		return
	}

	jsonResponse(w, http.StatusOK, prices)
}

// ============================================
// Handlers: Señales
// ============================================

// handleGetSignals devuelve las últimas señales generadas
func (s *Server) handleGetSignals(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	signals, err := s.db.GetLatestSignals(r.Context(), limit)
	if err != nil {
		log.Printf("Error obteniendo señales: %v", err)
		jsonError(w, http.StatusInternalServerError, "Error obteniendo señales")
		return
	}
	jsonResponse(w, http.StatusOK, signals)
}

// ============================================
// Handlers: Operaciones
// ============================================

// handleGetOperations devuelve las operaciones, filtradas opcionalmente por estado
func (s *Server) handleGetOperations(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("estado")

	operations, err := s.db.GetOperations(r.Context(), status)
	if err != nil {
		log.Printf("Error obteniendo operaciones: %v", err)
		jsonError(w, http.StatusInternalServerError, "Error obteniendo operaciones")
		return
	}
	jsonResponse(w, http.StatusOK, operations)
}

// ============================================
// Handlers: Portafolio
// ============================================

// handleGetPortfolio devuelve el resumen del portafolio
func (s *Server) handleGetPortfolio(w http.ResponseWriter, r *http.Request) {
	portfolio, err := s.db.GetPortfolioSummary(r.Context())
	if err != nil {
		log.Printf("Error obteniendo portafolio: %v", err)
		jsonError(w, http.StatusInternalServerError, "Error obteniendo portafolio")
		return
	}
	jsonResponse(w, http.StatusOK, portfolio)
}

// ============================================
// Handlers: Configuración
// ============================================

// handleGetConfig devuelve toda la configuración del bot
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.db.GetAllConfig(r.Context())
	if err != nil {
		log.Printf("Error obteniendo configuración: %v", err)
		jsonError(w, http.StatusInternalServerError, "Error obteniendo configuración")
		return
	}
	jsonResponse(w, http.StatusOK, cfg)
}

// handleUpdateConfig actualiza un valor de configuración
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key   string `json:"clave"`
		Value string `json:"valor"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "Cuerpo de petición inválido")
		return
	}

	if body.Key == "" || body.Value == "" {
		jsonError(w, http.StatusBadRequest, "Clave y valor son obligatorios")
		return
	}

	if err := s.db.UpdateConfig(r.Context(), body.Key, body.Value); err != nil {
		log.Printf("Error actualizando configuración: %v", err)
		jsonError(w, http.StatusInternalServerError, "Error actualizando configuración")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ============================================
// Handlers: Bot (Iniciar/Detener/Estado)
// ============================================

// handleGetBotStatus devuelve el estado actual del bot
func (s *Server) handleGetBotStatus(w http.ResponseWriter, r *http.Request) {
	status := s.engine.GetStatus()
	jsonResponse(w, http.StatusOK, status)
}

// handleStartBot inicia el motor de trading
func (s *Server) handleStartBot(w http.ResponseWriter, r *http.Request) {
	s.engine.Start()
	jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "Bot iniciado correctamente",
	})
}

// handleStopBot detiene el motor de trading
func (s *Server) handleStopBot(w http.ResponseWriter, r *http.Request) {
	s.engine.Stop()
	jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "Bot detenido correctamente",
	})
}

// ============================================
// Handlers: Indicadores
// ============================================

// handleGetIndicators calcula y devuelve los indicadores de un activo
func (s *Server) handleGetIndicators(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("simbolo")
	if symbol == "" {
		jsonError(w, http.StatusBadRequest, "Símbolo requerido")
		return
	}

	// Buscar el activo por símbolo
	assets, err := s.db.GetActiveAssets(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Error buscando activo")
		return
	}

	var found bool
	for _, asset := range assets {
		if asset.Symbol == symbol {
			found = true
			// Analizar el activo y obtener indicadores
			_, indicators, err := s.engine.AnalyzeAsset(r.Context(), asset)
			if err != nil {
				log.Printf("Error analizando %s: %v", symbol, err)
				jsonError(w, http.StatusInternalServerError, "Error calculando indicadores")
				return
			}
			jsonResponse(w, http.StatusOK, indicators)
			return
		}
	}

	if !found {
		jsonError(w, http.StatusNotFound, "Activo no encontrado: "+symbol)
	}
}

// ============================================
// Handlers: Señales IA (DeepSeek)
// ============================================

// handleGenerateAISignal genera una señal de trading usando DeepSeek AI
// Flujo: Obtener indicadores reales → Construir prompt → DeepSeek analiza → Respuesta estructurada
func (s *Server) handleGenerateAISignal(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("simbolo")
	if symbol == "" {
		jsonError(w, http.StatusBadRequest, "Símbolo requerido")
		return
	}

	// Verificar que DeepSeek esté configurado
	if s.deepseek == nil || !s.deepseek.IsConfigured() {
		jsonError(w, http.StatusServiceUnavailable, "DeepSeek AI no está configurado. Añade DEEPSEEK_API_KEY al archivo .env")
		return
	}

	// Buscar el activo por símbolo
	assets, err := s.db.GetActiveAssets(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Error buscando activo")
		return
	}

	var found bool
	for _, asset := range assets {
		if asset.Symbol == symbol {
			found = true

			// 1. Calcular indicadores reales del activo
			_, indicators, err := s.engine.AnalyzeAsset(r.Context(), asset)
			if err != nil {
				log.Printf("Error analizando %s para IA: %v", symbol, err)
				jsonError(w, http.StatusInternalServerError, "Error calculando indicadores para análisis IA")
				return
			}

			if indicators == nil {
				jsonError(w, http.StatusInternalServerError, "No se pudieron calcular indicadores para "+symbol)
				return
			}

			// 2. Obtener precios para contexto (últimas 30 velas)
			prices, err := s.engine.GetPrices(r.Context(), asset.Symbol, 30)
			if err != nil || len(prices) < 5 {
				jsonError(w, http.StatusInternalServerError, "Datos de precios insuficientes para análisis IA")
				return
			}

			// 3. Enviar a DeepSeek para análisis inteligente
			aiSignal, err := s.deepseek.GenerateSignal(r.Context(), asset, indicators, prices)
			if err != nil {
				log.Printf("Error generando señal IA para %s: %v", symbol, err)
				jsonError(w, http.StatusInternalServerError, "Error generando señal IA: "+err.Error())
				return
			}

			jsonResponse(w, http.StatusOK, aiSignal)
			return
		}
	}

	if !found {
		jsonError(w, http.StatusNotFound, "Activo no encontrado: "+symbol)
	}
}

// ============================================
// Handlers: Velas Históricas
// ============================================

// handleDownloadCandles inicia la descarga de velas históricas en segundo plano
func (s *Server) handleDownloadCandles(w http.ResponseWriter, r *http.Request) {
	if s.downloader == nil {
		jsonError(w, http.StatusServiceUnavailable, "Descargador no configurado. Necesita TWELVE_DATA_API_KEY")
		return
	}

	if s.downloader.IsRunning() {
		jsonError(w, http.StatusConflict, "Ya hay una descarga en progreso")
		return
	}

	if err := s.downloader.Start(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Descarga de velas iniciada en segundo plano",
	})
}

// handleGetCandleStatus devuelve el estado actual de la descarga de velas
func (s *Server) handleGetCandleStatus(w http.ResponseWriter, r *http.Request) {
	if s.downloader == nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"configured": false,
			"message":    "Descargador no configurado",
		})
		return
	}

	progress := s.downloader.GetProgress()
	jsonResponse(w, http.StatusOK, progress)
}

// handleGetCandleStats devuelve estadísticas de velas almacenadas por activo y timeframe
func (s *Server) handleGetCandleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.GetCandleStats(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Error obteniendo estadísticas: "+err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, stats)
}

// handleGetCandles obtiene velas almacenadas de un activo y timeframe específico
func (s *Server) handleGetCandles(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("simbolo")
	timeframe := r.PathValue("timeframe")

	// Validar timeframe
	validTF := map[string]bool{"5min": true, "15min": true, "30min": true, "1h": true, "4h": true, "1day": true}
	if !validTF[timeframe] {
		jsonError(w, http.StatusBadRequest, "Timeframe inválido. Válidos: 5min, 15min, 30min, 1h, 4h, 1day")
		return
	}

	// Buscar activo por símbolo
	assets, err := s.db.GetActiveAssets(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Error buscando activo")
		return
	}

	for _, asset := range assets {
		if asset.Symbol == symbol {
			// Leer parámetro limit (default 500)
			limit := 500
			if l := r.URL.Query().Get("limit"); l != "" {
				if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
					limit = parsed
				}
			}

			candles, err := s.db.GetCandles(r.Context(), asset.ID, timeframe, limit)
			if err != nil {
				jsonError(w, http.StatusInternalServerError, "Error obteniendo velas")
				return
			}

			jsonResponse(w, http.StatusOK, candles)
			return
		}
	}

	jsonError(w, http.StatusNotFound, "Activo no encontrado: "+symbol)
}

// ============================================
// Handlers: Actualización en Tiempo Real
// ============================================

// handleStartRealtime inicia el actualizador de velas en tiempo real
func (s *Server) handleStartRealtime(w http.ResponseWriter, r *http.Request) {
	if s.realtime == nil {
		jsonError(w, http.StatusServiceUnavailable, "Actualizador no configurado. Necesita TWELVE_DATA_API_KEY")
		return
	}

	if s.realtime.IsRunning() {
		jsonError(w, http.StatusConflict, "Actualizador en tiempo real ya está activo")
		return
	}

	if err := s.realtime.Start(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "🔴 Actualizador en tiempo real INICIADO",
	})
}

// handleStopRealtime detiene el actualizador en tiempo real
func (s *Server) handleStopRealtime(w http.ResponseWriter, r *http.Request) {
	if s.realtime == nil {
		jsonError(w, http.StatusServiceUnavailable, "Actualizador no configurado")
		return
	}

	if !s.realtime.IsRunning() {
		jsonError(w, http.StatusConflict, "Actualizador no está activo")
		return
	}

	s.realtime.Stop()

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "⏹️ Actualizador en tiempo real DETENIDO",
	})
}

// handleGetRealtimeStatus devuelve el estado del actualizador en tiempo real
func (s *Server) handleGetRealtimeStatus(w http.ResponseWriter, r *http.Request) {
	if s.realtime == nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"configured": false,
			"message":    "Actualizador no configurado",
		})
		return
	}

	status := s.realtime.GetStatus()
	jsonResponse(w, http.StatusOK, status)
}
