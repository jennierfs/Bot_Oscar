// ============================================
// Bot Oscar - Proveedor de Datos de Mercado
// Interfaz y generador de datos demo como último recurso
//
// Prioridad de proveedores:
//   1. Yahoo Finance (GRATIS, sin API key, acciones + commodities)
//   2. Alpha Vantage (si se configura una API key real)
//   3. Demo (solo si todo lo anterior falla)
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
// Usa Yahoo Finance como fuente principal (gratis, sin API key)
// Si se proporciona una API key de Alpha Vantage, la usa como respaldo
func NewProvider(apiKey string, redisCache *cache.Cache) Provider {
	// Yahoo Finance como proveedor principal
	yahooProvider := &YahooFinanceProvider{
		cache: redisCache,
	}

	// Si hay API key de Alpha Vantage válida, crear proveedor de respaldo
	var alphaProvider *AlphaVantageProvider
	if apiKey != "" && apiKey != "demo" {
		alphaProvider = &AlphaVantageProvider{
			apiKey: apiKey,
			cache:  redisCache,
		}
	}

	// Retornar proveedor con fallback
	return &MultiProvider{
		primary:  yahooProvider,
		fallback: alphaProvider,
		demo:     &DemoProvider{cache: redisCache},
		cache:    redisCache,
	}
}

// ============================================
// MultiProvider - Proveedor con sistema de fallback
// Intenta Yahoo → Alpha Vantage → Demo (último recurso)
// ============================================

// MultiProvider intenta múltiples fuentes de datos en orden
type MultiProvider struct {
	primary  *YahooFinanceProvider
	fallback *AlphaVantageProvider
	demo     *DemoProvider
	cache    *cache.Cache
}

// GetPrices obtiene precios intentando cada proveedor en orden de prioridad
func (m *MultiProvider) GetPrices(ctx context.Context, symbol string, days int) ([]models.Price, error) {
	// 1. Intentar Yahoo Finance (fuente principal)
	prices, err := m.primary.GetPrices(ctx, symbol, days)
	if err == nil && len(prices) > 0 {
		return prices, nil
	}
	if err != nil {
		log.Printf("⚠️ [%s] Yahoo Finance falló: %v", symbol, err)
	}

	// 2. Intentar Alpha Vantage como respaldo (si hay API key)
	if m.fallback != nil {
		prices, err = m.fallback.GetPrices(ctx, symbol, days)
		if err == nil && len(prices) > 0 {
			log.Printf("✅ [%s] Datos obtenidos de Alpha Vantage (respaldo)", symbol)
			return prices, nil
		}
		if err != nil {
			log.Printf("⚠️ [%s] Alpha Vantage también falló: %v", symbol, err)
		}
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
	"GC=F": 2050.0, // Oro
	"SI=F": 23.50,  // Plata
	"CL=F": 75.0,   // Petróleo Crudo
	"NG=F": 2.50,   // Gas Natural
	"LMT":  450.0,  // Lockheed Martin
	"RTX":  95.0,   // Raytheon
	"NOC":  470.0,  // Northrop Grumman
	"GD":   270.0,  // General Dynamics
	"BA":   210.0,  // Boeing
	"LHX":  210.0,  // L3Harris
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
