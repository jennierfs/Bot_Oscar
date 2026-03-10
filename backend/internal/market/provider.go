// ============================================
// Bot Oscar - Proveedor de Datos de Mercado
// Interfaz y generador de datos demo como último recurso
//
// Prioridad de proveedores:
//  1. Twelve Data (PRINCIPAL - 800 créditos/día gratis, acciones + ETFs)
//  2. Yahoo Finance (RESPALDO - gratis, sin API key)
//  3. Demo (solo si todo lo anterior falla)
//
// ============================================
package market

import (
	"context"
	"log"
	"math"
	"math/rand"
	"time"

	"bot-oscar/internal/cache"
	"bot-oscar/internal/models"
)

// Provider define la interfaz para obtener datos de mercado
type Provider interface {
	// GetPrices obtiene los últimos N días de precios OHLCV para un símbolo
	GetPrices(ctx context.Context, symbol string, days int) ([]models.Price, error)
}

// NewProvider crea el proveedor de datos de mercado
// Prioridad: Twelve Data (principal) → Yahoo Finance (respaldo) → Demo
func NewProvider(apiKey string, redisCache *cache.Cache) Provider {
	// Yahoo Finance como proveedor de respaldo
	yahooProvider := &YahooFinanceProvider{
		cache: redisCache,
	}

	// Retornar proveedor con fallback
	return &MultiProvider{
		yahoo: yahooProvider,
		demo:  &DemoProvider{cache: redisCache},
		cache: redisCache,
	}
}

// NewProviderWithTwelveData crea el proveedor con Twelve Data como fuente principal
// Prioridad: Twelve Data → Yahoo Finance → Demo
func NewProviderWithTwelveData(twelveDataKey string, redisCache *cache.Cache) Provider {
	// Twelve Data como proveedor principal
	var tdProvider *TwelveDataProvider
	if twelveDataKey != "" {
		tdProvider = NewTwelveDataProvider(twelveDataKey, redisCache)
	}

	// Yahoo Finance como respaldo
	yahooProvider := &YahooFinanceProvider{
		cache: redisCache,
	}

	return &MultiProvider{
		twelveData: tdProvider,
		yahoo:      yahooProvider,
		demo:       &DemoProvider{cache: redisCache},
		cache:      redisCache,
	}
}

// ============================================
// MultiProvider - Proveedor con sistema de fallback
// Intenta Twelve Data → Yahoo Finance → Demo (último recurso)
// ============================================

// MultiProvider intenta múltiples fuentes de datos en orden de prioridad
type MultiProvider struct {
	twelveData *TwelveDataProvider   // Fuente principal (Twelve Data API)
	yahoo      *YahooFinanceProvider // Respaldo (Yahoo Finance, sin API key)
	demo       *DemoProvider         // Último recurso (datos simulados)
	cache      *cache.Cache
}

// GetPrices obtiene precios intentando cada proveedor en orden de prioridad:
// 1. Twelve Data (si está configurado)
// 2. Yahoo Finance (respaldo gratuito)
// 3. Demo (último recurso)
func (m *MultiProvider) GetPrices(ctx context.Context, symbol string, days int) ([]models.Price, error) {
	// 1. Intentar Twelve Data (fuente principal)
	if m.twelveData != nil {
		prices, err := m.twelveData.GetPrices(ctx, symbol, days)
		if err == nil && len(prices) > 0 {
			return prices, nil
		}
		if err != nil {
			log.Printf("⚠️ [%s] Twelve Data falló: %v", symbol, err)
		}
	}

	// 2. Intentar Yahoo Finance como respaldo
	prices, err := m.yahoo.GetPrices(ctx, symbol, days)
	if err == nil && len(prices) > 0 {
		log.Printf("✅ [%s] Datos obtenidos de Yahoo Finance (respaldo)", symbol)
		return prices, nil
	}
	if err != nil {
		log.Printf("⚠️ [%s] Yahoo Finance también falló: %v", symbol, err)
	}

	// 3. Último recurso: datos demo (solo si TODAS las fuentes fallaron)
	log.Printf("🟡 [%s] Usando datos DEMO - todas las fuentes reales fallaron", symbol)
	return m.demo.GetPrices(ctx, symbol, days)
}

// ============================================
// Proveedor de datos Demo
// Genera datos OHLCV realistas basados en random walk
// ============================================

// DemoProvider genera datos de mercado simulados para pruebas
type DemoProvider struct {
	cache *cache.Cache
}

// preciosBase contiene precios de referencia para cada activo
// Estos valores son aproximaciones realistas del mercado
var preciosBase = map[string]float64{
	"GLD": 470.0, // ETF Oro (SPDR Gold Trust)
	"SLV": 78.0,  // ETF Plata (iShares Silver)
	"USO": 104.0, // ETF Petróleo (US Oil Fund)
	"UNG": 13.0,  // ETF Gas Natural (US Natural Gas)
	"LMT": 665.0, // Lockheed Martin
	"RTX": 130.0, // Raytheon
	"NOC": 530.0, // Northrop Grumman
	"GD":  360.0, // General Dynamics
	"BA":  230.0, // Boeing
	"LHX": 240.0, // L3Harris
}

// GetPrices genera datos de precios demo para un activo
func (p *DemoProvider) GetPrices(ctx context.Context, symbol string, days int) ([]models.Price, error) {
	// Intentar obtener de caché primero
	cached, err := p.cache.GetPrices(ctx, symbol)
	if err == nil && cached != nil && len(cached) >= days {
		// Devolver los últimos N días del caché
		start := len(cached) - days
		return cached[start:], nil
	}

	// Generar datos demo
	prices := generateDemoData(symbol, days)

	// Cachear por 5 minutos
	p.cache.SetPrices(ctx, symbol, prices, 5*time.Minute)

	return prices, nil
}

// generateDemoData genera datos OHLCV realistas usando random walk
func generateDemoData(symbol string, days int) []models.Price {
	// Obtener precio base del activo
	basePrice, exists := preciosBase[symbol]
	if !exists {
		basePrice = 100.0
	}

	// Semilla basada en el símbolo para datos consistentes en la sesión
	seed := int64(0)
	for _, c := range symbol {
		seed += int64(c)
	}
	rng := rand.New(rand.NewSource(seed + time.Now().Unix()/300))

	prices := make([]models.Price, days)
	currentPrice := basePrice

	for i := 0; i < days; i++ {
		// Cambio diario: entre -2% y +2.2% (ligero sesgo alcista)
		changePercent := (rng.Float64() - 0.48) * 0.04
		currentPrice *= (1 + changePercent)

		// Asegurar que el precio no sea negativo
		if currentPrice < basePrice*0.5 {
			currentPrice = basePrice * 0.5
		}
		if currentPrice > basePrice*1.5 {
			currentPrice = basePrice * 1.5
		}

		// Generar OHLC a partir del cierre
		volatility := basePrice * 0.015
		high := currentPrice + rng.Float64()*volatility
		low := currentPrice - rng.Float64()*volatility
		open := low + rng.Float64()*(high-low)

		// Asegurar que high >= max(open, close) y low <= min(open, close)
		high = math.Max(high, math.Max(open, currentPrice))
		low = math.Min(low, math.Min(open, currentPrice))

		// Volumen aleatorio realista
		volume := int64(rng.Intn(2000000) + 500000)

		// Fecha: hoy - (days - i) días
		date := time.Now().AddDate(0, 0, -(days - i))

		prices[i] = models.Price{
			AssetID: 0,
			Open:    math.Round(open*100) / 100,
			High:    math.Round(high*100) / 100,
			Low:     math.Round(low*100) / 100,
			Close:   math.Round(currentPrice*100) / 100,
			Volume:  volume,
			Date:    date,
		}
	}

	return prices
}
