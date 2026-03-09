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
	// Obtener precios históricos (mínimo 250 para SMA200 + margen)
	prices, err := e.provider.GetPrices(ctx, asset.Symbol, 250)
	if err != nil {
		return nil, nil, err
	}

	if len(prices) < 50 {
		return nil, nil, nil // Datos insuficientes, no es error
	}

	// Guardar precios en la BD para referencia futura
	e.db.SavePrices(ctx, asset.ID, prices)

	// Extraer precios de cierre para los indicadores
	closes := make([]float64, len(prices))
	for i, p := range prices {
		closes[i] = p.Close
	}

	// ============================================
	// Calcular TODOS los indicadores
	// ============================================

	// Medias Móviles
	sma50 := indicators.CalculateSMA(closes, 50)
	sma200 := indicators.CalculateSMA(closes, 200)
	ema12 := indicators.CalculateEMA(closes, 12)
	ema26 := indicators.CalculateEMA(closes, 26)

	// RSI (periodo estándar: 14)
	rsiValues := indicators.CalculateRSI(closes, 14)

	// MACD (12, 26, 9 - parámetros estándar)
	macdResult := indicators.CalculateMACD(closes, 12, 26, 9)

	// Bandas de Bollinger (20, 2.0 - estándar)
	bollingerResult := indicators.CalculateBollinger(closes, 20, 2.0)

	// ============================================
	// Construir el objeto de indicadores con los últimos valores
	// ============================================
	indValues := &models.IndicatorValues{
		Symbol: asset.Symbol,
	}

	// Último valor de cada indicador (el más reciente)
	if rsiValues != nil && len(rsiValues) > 0 {
		indValues.RSI = rsiValues[len(rsiValues)-1]
	}
	if sma50 != nil && len(sma50) > 0 {
		indValues.SMA50 = sma50[len(sma50)-1]
	}
	if sma200 != nil && len(sma200) > 0 {
		indValues.SMA200 = sma200[len(sma200)-1]
	}
	if ema12 != nil && len(ema12) > 0 {
		indValues.EMA12 = ema12[len(ema12)-1]
	}
	if ema26 != nil && len(ema26) > 0 {
		indValues.EMA26 = ema26[len(ema26)-1]
	}
	if macdResult != nil && len(macdResult.MACD) > 0 {
		indValues.MACD = models.MACDValues{
			MACD:      macdResult.MACD[len(macdResult.MACD)-1],
			Signal:    macdResult.Signal[len(macdResult.Signal)-1],
			Histogram: macdResult.Histogram[len(macdResult.Histogram)-1],
		}
	}
	if bollingerResult != nil && len(bollingerResult.Upper) > 0 {
		indValues.Bollinger = models.BollingerValues{
			Upper:  bollingerResult.Upper[len(bollingerResult.Upper)-1],
			Middle: bollingerResult.Middle[len(bollingerResult.Middle)-1],
			Lower:  bollingerResult.Lower[len(bollingerResult.Lower)-1],
		}
	}

	// ============================================
	// Evaluar confluencia y generar puntuación
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
		// Calcular ATR para posicionar el stop loss
		atr := CalculateATR(prices, 14)
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
