// ============================================
// Bot Oscar - Motor de Trading
// Cerebro del bot: analiza activos, calcula indicadores,
// genera señales y gestiona el ciclo de análisis
//
// Flujo de análisis (como un trader profesional):
// 1. Obtener precios de cada activo
// 2. Calcular TODOS los indicadores (RSI, MACD, SMA, EMA, Bollinger)
// 3. Evaluar confluencia (múltiples indicadores deben coincidir)
// 4. Generar señal solo si hay suficiente confluencia
// 5. Calcular stop loss y take profit basados en ATR
// 6. Validar con gestión de riesgo antes de ejecutar
// ============================================
package trading

import (
	"context"
	"log"
	"sync"
	"time"

	"bot-oscar/internal/cache"
	"bot-oscar/internal/config"
	"bot-oscar/internal/db"
	"bot-oscar/internal/indicators"
	"bot-oscar/internal/market"
	"bot-oscar/internal/models"
)

// Engine es el motor principal de trading
type Engine struct {
	db       *db.Database
	cache    *cache.Cache
	provider market.Provider
	config   *config.Config

	running      bool
	stopCh       chan struct{}
	mu           sync.Mutex
	lastAnalysis *time.Time
}

// NewEngine crea una nueva instancia del motor de trading
func NewEngine(database *db.Database, redisCache *cache.Cache, provider market.Provider, cfg *config.Config) *Engine {
	return &Engine{
		db:       database,
		cache:    redisCache,
		provider: provider,
		config:   cfg,
		stopCh:   make(chan struct{}),
	}
}

// Start inicia el ciclo de análisis del motor de trading
func (e *Engine) Start() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		log.Println("⚠️ El motor de trading ya está en ejecución")
		return
	}

	e.running = true
	e.stopCh = make(chan struct{})
	log.Println("🟢 Motor de trading iniciado")

	// Lanzar goroutine con el ciclo de análisis
	go e.runAnalysisCycle()
}

// Stop detiene el motor de trading de forma segura
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	e.running = false
	close(e.stopCh)
	log.Println("🔴 Motor de trading detenido")
}

// IsRunning indica si el motor está en ejecución
func (e *Engine) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

// GetStatus devuelve el estado actual del bot
func (e *Engine) GetStatus() models.BotStatus {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Contar activos monitoreados
	assets, _ := e.db.GetActiveAssets(context.Background())

	// Obtener modo desde la BD
	cfg, _ := e.db.GetAllConfig(context.Background())
	mode := "paper"
	if m, ok := cfg["modo"]; ok {
		mode = m
	}

	// Contar señales recientes (últimas 24 horas)
	signals, _ := e.db.GetLatestSignals(context.Background(), 100)
	activeSignals := 0
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, s := range signals {
		if s.CreatedAt.After(cutoff) {
			activeSignals++
		}
	}

	return models.BotStatus{
		Running:         e.running,
		Mode:            mode,
		LastAnalysis:    e.lastAnalysis,
		AssetsMonitored: len(assets),
		ActiveSignals:   activeSignals,
	}
}

// runAnalysisCycle ejecuta el ciclo periódico de análisis
func (e *Engine) runAnalysisCycle() {
	// Ejecutar primer análisis inmediatamente
	e.runFullAnalysis()

	// Configurar intervalo desde la BD o usar default
	interval := time.Duration(e.config.AnalysisInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.runFullAnalysis()
		}
	}
}

// runFullAnalysis ejecuta un análisis completo de todos los activos
func (e *Engine) runFullAnalysis() {
	ctx := context.Background()

	// Obtener todos los activos activos
	assets, err := e.db.GetActiveAssets(ctx)
	if err != nil {
		log.Printf("❌ Error obteniendo activos: %v", err)
		return
	}

	log.Printf("📊 Iniciando análisis de %d activos...", len(assets))

	for _, asset := range assets {
		// Analizar cada activo
		signal, _, err := e.AnalyzeAsset(ctx, asset)
		if err != nil {
			log.Printf("⚠️ Error analizando %s: %v", asset.Symbol, err)
			continue
		}

		// Si se generó una señal con suficiente fuerza, guardarla
		if signal != nil {
			if err := e.db.SaveSignal(ctx, signal); err != nil {
				log.Printf("❌ Error guardando señal para %s: %v", asset.Symbol, err)
			} else {
				log.Printf("📌 Señal %s para %s (fuerza: %d): %s",
					signal.Type, asset.Symbol, signal.Strength, signal.Reason)
			}
		}
	}

	// Actualizar timestamp del último análisis
	now := time.Now()
	e.mu.Lock()
	e.lastAnalysis = &now
	e.mu.Unlock()

	log.Println("✅ Análisis completado")
}

// GetPrices obtiene precios de un activo a través del proveedor de mercado
// Método público para que otros componentes (como el handler de IA) puedan acceder
func (e *Engine) GetPrices(ctx context.Context, symbol string, days int) ([]models.Price, error) {
	return e.provider.GetPrices(ctx, symbol, days)
}

// AnalyzeAsset analiza un activo individual calculando todos los indicadores
// Retorna: señal (si hay), valores de indicadores, error
func (e *Engine) AnalyzeAsset(ctx context.Context, asset models.Asset) (*models.Signal, *models.IndicatorValues, error) {
	// Obtener precios históricos (mínimo 300 para EMA200 + margen)
	prices, err := e.provider.GetPrices(ctx, asset.Symbol, 300)
	if err != nil {
		return nil, nil, err
	}

	if len(prices) < 50 {
		return nil, nil, nil // Datos insuficientes, no es error
	}

	// Guardar precios en la BD para referencia futura
	e.db.SavePrices(ctx, asset.ID, prices)

	// Extraer arrays OHLCV para los indicadores
	n := len(prices)
	closes := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	volumes := make([]int64, n)
	for i, p := range prices {
		closes[i] = p.Close
		highs[i] = p.High
		lows[i] = p.Low
		volumes[i] = p.Volume
	}

	// ============================================
	// Calcular TODOS los indicadores profesionales
	// ============================================

	// --- Medias Móviles Simples ---
	sma50 := indicators.CalculateSMA(closes, 50)
	sma200 := indicators.CalculateSMA(closes, 200)

	// --- Medias Móviles Exponenciales ---
	ema12 := indicators.CalculateEMA(closes, 12)   // Componente MACD rápido
	ema26 := indicators.CalculateEMA(closes, 26)   // Componente MACD lento
	ema21 := indicators.CalculateEMA(closes, 21)   // Pullbacks institucionales
	ema50 := indicators.CalculateEMA(closes, 50)   // Tendencia intermedia
	ema200 := indicators.CalculateEMA(closes, 200) // Tendencia principal (la más importante)

	// --- Osciladores ---
	rsiValues := indicators.CalculateRSI(closes, 14)          // RSI estándar
	macdResult := indicators.CalculateMACD(closes, 12, 26, 9) // MACD estándar

	// --- Volatilidad ---
	bollingerResult := indicators.CalculateBollinger(closes, 20, 2.0) // Bollinger estándar
	atr := CalculateATR(prices, 14)                                   // ATR para volatilidad real

	// --- Volumen ---
	vwapResult := indicators.CalculateVWAP(highs, lows, closes, volumes, 20) // VWAP de 20 días

	// ============================================
	// Construir el objeto con los últimos valores de cada indicador
	// ============================================
	indValues := &models.IndicatorValues{
		Symbol: asset.Symbol,
		ATR:    atr,
	}

	// Medias Móviles Simples
	if sma50 != nil && len(sma50) > 0 {
		indValues.SMA50 = sma50[len(sma50)-1]
	}
	if sma200 != nil && len(sma200) > 0 {
		indValues.SMA200 = sma200[len(sma200)-1]
	}

	// Medias Móviles Exponenciales
	if ema12 != nil && len(ema12) > 0 {
		indValues.EMA12 = ema12[len(ema12)-1]
	}
	if ema26 != nil && len(ema26) > 0 {
		indValues.EMA26 = ema26[len(ema26)-1]
	}
	if ema21 != nil && len(ema21) > 0 {
		indValues.EMA21 = ema21[len(ema21)-1]
	}
	if ema50 != nil && len(ema50) > 0 {
		indValues.EMA50 = ema50[len(ema50)-1]
	}
	if ema200 != nil && len(ema200) > 0 {
		indValues.EMA200 = ema200[len(ema200)-1]
	}

	// Osciladores
	if rsiValues != nil && len(rsiValues) > 0 {
		indValues.RSI = rsiValues[len(rsiValues)-1]
	}
	if macdResult != nil && len(macdResult.MACD) > 0 {
		indValues.MACD = models.MACDValues{
			MACD:      macdResult.MACD[len(macdResult.MACD)-1],
			Signal:    macdResult.Signal[len(macdResult.Signal)-1],
			Histogram: macdResult.Histogram[len(macdResult.Histogram)-1],
		}
	}

	// Volatilidad
	if bollingerResult != nil && len(bollingerResult.Upper) > 0 {
		indValues.Bollinger = models.BollingerValues{
			Upper:  bollingerResult.Upper[len(bollingerResult.Upper)-1],
			Middle: bollingerResult.Middle[len(bollingerResult.Middle)-1],
			Lower:  bollingerResult.Lower[len(bollingerResult.Lower)-1],
		}
	}

	// Volumen (VWAP + análisis de volumen)
	if vwapResult != nil {
		indValues.VWAP = vwapResult.VWAP
		indValues.VolumenProm = vwapResult.VolumenProm
		indValues.VolumenHoy = vwapResult.VolumenHoy
		indValues.VolumenRatio = vwapResult.VolumenRatio
	}

	// ============================================
	// Evaluar confluencia profesional y generar puntuación
	// ============================================
	currentPrice := closes[len(closes)-1]
	score, signalType, reason := ScoreSignal(*indValues, currentPrice)

	indValues.Score = score
	indValues.Signal = signalType

	// ============================================
	// Generar señal si la fuerza es suficiente
	// ============================================
	minScore := e.config.MinSignalScore
	if signalType == "COMPRA" && score >= minScore || signalType == "VENTA" && score <= (100-minScore) {
		sl := CalculateStopLoss(currentPrice, signalType, atr)
		tp := CalculateTakeProfit(currentPrice, sl, e.config.RiskRewardRatio)

		signal := &models.Signal{
			AssetID:    asset.ID,
			Symbol:     asset.Symbol,
			AssetName:  asset.Name,
			Type:       signalType,
			Strength:   score,
			EntryPrice: currentPrice,
			StopLoss:   sl,
			TakeProfit: tp,
			Reason:     reason,
			CreatedAt:  time.Now(),
		}

		return signal, indValues, nil
	}

	return nil, indValues, nil
}
