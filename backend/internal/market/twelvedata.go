// ============================================
// Bot Oscar - Proveedor Twelve Data
// Obtiene datos REALES de mercado desde Twelve Data API
//
// Características:
//   - Soporta acciones (LMT, RTX, BA...) y ETFs de commodities (GLD, SLV, USO, UNG)
//   - Batch requests: hasta 8 símbolos por llamada (1 crédito cada uno)
//   - Rate limiter integrado: máximo 8 créditos por minuto
//   - Caché en Redis con TTL inteligente (10 min horario mercado, 4h fuera)
//   - Contador de créditos diarios para monitoreo
//   - Plan gratis: 800 créditos/día, 8 créditos/minuto
//
// Documentación: https://twelvedata.com/docs
// ============================================
package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"bot-oscar/internal/cache"
	"bot-oscar/internal/models"
)

// ============================================
// Estructura del proveedor Twelve Data
// ============================================

// TwelveDataProvider obtiene datos reales desde Twelve Data API
type TwelveDataProvider struct {
	apiKey string       // API key de Twelve Data (gratis)
	cache  *cache.Cache // Caché Redis para no repetir llamadas
	client *http.Client // Cliente HTTP con timeout

	// Rate limiter: máximo 8 peticiones por minuto
	mu             sync.Mutex   // Protege el rate limiter
	requestTimes   []time.Time  // Timestamps de las últimas peticiones
	maxPerMinute   int          // Límite de créditos por minuto (8)
	creditosUsados atomic.Int64 // Contador de créditos usados hoy
	ultimoResetDia time.Time    // Fecha del último reset del contador
}

// NewTwelveDataProvider crea un nuevo proveedor de Twelve Data
func NewTwelveDataProvider(apiKey string, redisCache *cache.Cache) *TwelveDataProvider {
	return &TwelveDataProvider{
		apiKey:         apiKey,
		cache:          redisCache,
		client:         &http.Client{Timeout: 30 * time.Second},
		requestTimes:   make([]time.Time, 0, 10),
		maxPerMinute:   8,
		ultimoResetDia: time.Now().Truncate(24 * time.Hour),
	}
}

// ============================================
// Estructuras de respuesta de Twelve Data API
// ============================================

// twelveDataResponse estructura para respuesta de un solo símbolo
type twelveDataResponse struct {
	Meta   twelveDataMeta    `json:"meta"`
	Values []twelveDataValue `json:"values"`
	Status string            `json:"status"`
	Code   int               `json:"code"`
	Msg    string            `json:"message"`
}

// twelveDataMeta metadatos del símbolo
type twelveDataMeta struct {
	Symbol   string `json:"symbol"`
	Interval string `json:"interval"`
	Currency string `json:"currency"`
	Exchange string `json:"exchange"`
	Type     string `json:"type"`
}

// twelveDataValue una vela OHLCV
type twelveDataValue struct {
	Datetime string `json:"datetime"`
	Open     string `json:"open"`
	High     string `json:"high"`
	Low      string `json:"low"`
	Close    string `json:"close"`
	Volume   string `json:"volume"`
}

// ============================================
// Método principal: obtener precios de un símbolo
// ============================================

// GetPrices obtiene precios REALES desde Twelve Data con caché inteligente
func (p *TwelveDataProvider) GetPrices(ctx context.Context, symbol string, days int) ([]models.Price, error) {
	// 1. Intentar obtener de caché Redis primero
	cached, err := p.cache.GetPrices(ctx, symbol)
	if err == nil && cached != nil && len(cached) >= days {
		start := len(cached) - days
		log.Printf("📦 [%s] Datos de Twelve Data obtenidos de caché (%d velas)", symbol, len(cached[start:]))
		return cached[start:], nil
	}

	// 2. Verificar rate limit antes de hacer la petición
	if err := p.esperarRateLimit(); err != nil {
		return nil, fmt.Errorf("[%s] rate limit de Twelve Data: %w", symbol, err)
	}

	// 3. Hacer petición a Twelve Data
	prices, err := p.fetchTimeSeries(ctx, symbol, days)
	if err != nil {
		return nil, fmt.Errorf("[%s] error obteniendo datos de Twelve Data: %w", symbol, err)
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("[%s] sin datos de Twelve Data", symbol)
	}

	// 4. Determinar TTL de caché: 10 min en horario de mercado, 4h fuera
	ttl := p.calcularTTLCache()

	// 5. Guardar en caché Redis
	if err := p.cache.SetPrices(ctx, symbol, prices, ttl); err != nil {
		log.Printf("⚠️ [%s] Error guardando en caché: %v", symbol, err)
	}

	log.Printf("✅ [%s] %d velas reales obtenidas de Twelve Data (créditos hoy: %d/800)",
		symbol, len(prices), p.creditosUsados.Load())

	return prices, nil
}

// ============================================
// Petición HTTP a la API de Twelve Data
// ============================================

// fetchTimeSeries obtiene la serie temporal de un símbolo
func (p *TwelveDataProvider) fetchTimeSeries(ctx context.Context, symbol string, days int) ([]models.Price, error) {
	// Determinar outputsize: Twelve Data permite hasta 5000 en plan gratis
	// Pedimos un poco más de lo necesario para tener margen
	outputSize := days + 20
	if outputSize > 800 {
		outputSize = 800
	}

	// Construir URL de la API
	url := fmt.Sprintf(
		"https://api.twelvedata.com/time_series?symbol=%s&interval=1day&outputsize=%d&apikey=%s",
		symbol, outputSize, p.apiKey,
	)

	// Crear petición HTTP
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando petición: %w", err)
	}

	// Ejecutar petición
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en petición HTTP: %w", err)
	}
	defer resp.Body.Close()

	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %w", err)
	}

	// Registrar crédito usado
	p.registrarCredito()

	// Verificar código HTTP
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Twelve Data respondió con código %d: %s",
			resp.StatusCode, string(body[:minInt(len(body), 200)]))
	}

	// Decodificar JSON
	var tdResp twelveDataResponse
	if err := json.Unmarshal(body, &tdResp); err != nil {
		return nil, fmt.Errorf("error decodificando JSON: %w", err)
	}

	// Verificar errores de la API
	if tdResp.Status == "error" {
		return nil, fmt.Errorf("error de Twelve Data (código %d): %s", tdResp.Code, tdResp.Msg)
	}

	// Convertir valores a nuestro modelo de precios
	prices := make([]models.Price, 0, len(tdResp.Values))

	for _, v := range tdResp.Values {
		// Parsear fecha (formato: "2026-03-09")
		date, err := time.Parse("2006-01-02", v.Datetime)
		if err != nil {
			continue // Saltar velas con fecha inválida
		}

		// Parsear valores numéricos
		openVal, _ := strconv.ParseFloat(v.Open, 64)
		highVal, _ := strconv.ParseFloat(v.High, 64)
		lowVal, _ := strconv.ParseFloat(v.Low, 64)
		closeVal, _ := strconv.ParseFloat(v.Close, 64)
		volumeVal, _ := strconv.ParseInt(v.Volume, 10, 64)

		// Saltar velas con datos incompletos
		if closeVal == 0 || openVal == 0 {
			continue
		}

		prices = append(prices, models.Price{
			Open:   math.Round(openVal*100) / 100,
			High:   math.Round(highVal*100) / 100,
			Low:    math.Round(lowVal*100) / 100,
			Close:  math.Round(closeVal*100) / 100,
			Volume: volumeVal,
			Date:   date,
		})
	}

	// Twelve Data devuelve los datos del más reciente al más antiguo
	// Invertir para tener orden cronológico (antiguo → reciente)
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Date.Before(prices[j].Date)
	})

	// Limitar a los últimos N días solicitados
	if len(prices) > days {
		prices = prices[len(prices)-days:]
	}

	return prices, nil
}

// ============================================
// Rate Limiter - Controla uso de créditos
// Máximo 8 créditos por minuto (plan gratis)
// ============================================

// esperarRateLimit espera si es necesario para no exceder el límite por minuto
func (p *TwelveDataProvider) esperarRateLimit() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	ahora := time.Now()

	// Limpiar timestamps de hace más de 1 minuto
	limpio := make([]time.Time, 0, len(p.requestTimes))
	for _, t := range p.requestTimes {
		if ahora.Sub(t) < time.Minute {
			limpio = append(limpio, t)
		}
	}
	p.requestTimes = limpio

	// Si hemos alcanzado el límite por minuto, esperar
	if len(p.requestTimes) >= p.maxPerMinute {
		tiempoEspera := time.Minute - ahora.Sub(p.requestTimes[0])
		if tiempoEspera > 0 {
			log.Printf("⏳ Twelve Data rate limit: esperando %v...", tiempoEspera.Round(time.Second))
			p.mu.Unlock()
			time.Sleep(tiempoEspera + 500*time.Millisecond) // +500ms de margen
			p.mu.Lock()
			// Limpiar de nuevo después de esperar
			p.requestTimes = make([]time.Time, 0, 10)
		}
	}

	// Registrar esta petición con el timestamp ACTUAL (después de cualquier espera)
	p.requestTimes = append(p.requestTimes, time.Now())

	return nil
}

// registrarCredito incrementa el contador diario de créditos
func (p *TwelveDataProvider) registrarCredito() {
	hoy := time.Now().Truncate(24 * time.Hour)

	// Reset diario del contador
	if hoy.After(p.ultimoResetDia) {
		p.creditosUsados.Store(0)
		p.ultimoResetDia = hoy
	}

	p.creditosUsados.Add(1)
}

// CreditosUsadosHoy retorna los créditos consumidos hoy (para monitoreo)
func (p *TwelveDataProvider) CreditosUsadosHoy() int64 {
	return p.creditosUsados.Load()
}

// ============================================
// Caché inteligente según horario de mercado
// ============================================

// calcularTTLCache determina el TTL según si el mercado está abierto o cerrado
// Mercado NYSE/NASDAQ: 9:30 AM - 4:00 PM ET (lunes a viernes)
func (p *TwelveDataProvider) calcularTTLCache() time.Duration {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return 10 * time.Minute // Fallback: 10 minutos
	}

	ahora := time.Now().In(loc)
	diaSemana := ahora.Weekday()
	hora := ahora.Hour()
	minuto := ahora.Minute()
	minutosDelDia := hora*60 + minuto

	// Fin de semana: caché de 6 horas
	if diaSemana == time.Saturday || diaSemana == time.Sunday {
		return 6 * time.Hour
	}

	// Horario de mercado: 9:30 AM (570 min) a 4:00 PM (960 min) ET
	mercadoAbre := 9*60 + 30 // 570
	mercadoCierra := 16 * 60 // 960

	if minutosDelDia >= mercadoAbre && minutosDelDia < mercadoCierra {
		// Mercado abierto: caché de 10 minutos
		return 10 * time.Minute
	}

	// Mercado cerrado: caché de 4 horas
	return 4 * time.Hour
}

// ============================================
// Batch Request - Obtener múltiples símbolos en 1 llamada
// Cada símbolo cuenta como 1 crédito, pero es más eficiente
// ============================================

// GetPricesBatch obtiene precios de múltiples símbolos en una sola petición
// Retorna un mapa símbolo → precios. Los que fallan se omiten del resultado.
func (p *TwelveDataProvider) GetPricesBatch(ctx context.Context, symbols []string, days int) map[string][]models.Price {
	resultado := make(map[string][]models.Price)

	// Verificar qué símbolos ya están en caché
	sinCache := make([]string, 0, len(symbols))
	for _, sym := range symbols {
		cached, err := p.cache.GetPrices(ctx, sym)
		if err == nil && cached != nil && len(cached) >= days {
			start := len(cached) - days
			resultado[sym] = cached[start:]
		} else {
			sinCache = append(sinCache, sym)
		}
	}

	if len(sinCache) == 0 {
		log.Printf("📦 Batch: todos los %d símbolos obtenidos de caché", len(symbols))
		return resultado
	}

	// Dividir en lotes de máximo 8 símbolos (límite de Twelve Data)
	for i := 0; i < len(sinCache); i += 8 {
		fin := i + 8
		if fin > len(sinCache) {
			fin = len(sinCache)
		}
		lote := sinCache[i:fin]

		// Rate limit para el lote (cada símbolo = 1 crédito)
		for range lote {
			if err := p.esperarRateLimit(); err != nil {
				log.Printf("⚠️ Rate limit error en batch: %v", err)
				break
			}
		}

		// Hacer petición batch
		preciosLote, err := p.fetchBatch(ctx, lote, days)
		if err != nil {
			log.Printf("⚠️ Error en batch [%s]: %v", strings.Join(lote, ","), err)
			continue
		}

		// Agregar resultados y cachear
		ttl := p.calcularTTLCache()
		for sym, prices := range preciosLote {
			resultado[sym] = prices
			if err := p.cache.SetPrices(ctx, sym, prices, ttl); err != nil {
				log.Printf("⚠️ [%s] Error guardando batch en caché: %v", sym, err)
			}
		}

		log.Printf("✅ Batch [%s]: %d/%d símbolos obtenidos (créditos hoy: %d/800)",
			strings.Join(lote, ","), len(preciosLote), len(lote), p.creditosUsados.Load())
	}

	return resultado
}

// fetchBatch realiza una petición batch a Twelve Data (máximo 8 símbolos)
func (p *TwelveDataProvider) fetchBatch(ctx context.Context, symbols []string, days int) (map[string][]models.Price, error) {
	outputSize := days + 20
	if outputSize > 800 {
		outputSize = 800
	}

	// Construir URL con múltiples símbolos separados por coma
	symbolsStr := strings.Join(symbols, ",")
	url := fmt.Sprintf(
		"https://api.twelvedata.com/time_series?symbol=%s&interval=1day&outputsize=%d&apikey=%s",
		symbolsStr, outputSize, p.apiKey,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando petición batch: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en petición batch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta batch: %w", err)
	}

	// Registrar créditos usados (1 por símbolo)
	for range symbols {
		p.registrarCredito()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Twelve Data batch respondió con código %d", resp.StatusCode)
	}

	resultado := make(map[string][]models.Price)

	// Si es un solo símbolo, la respuesta es un objeto directo
	if len(symbols) == 1 {
		var tdResp twelveDataResponse
		if err := json.Unmarshal(body, &tdResp); err != nil {
			return nil, fmt.Errorf("error decodificando batch (1 símbolo): %w", err)
		}
		if tdResp.Status == "error" {
			return nil, fmt.Errorf("error Twelve Data para %s: %s", symbols[0], tdResp.Msg)
		}
		prices := convertirValoresTD(tdResp.Values, days)
		if len(prices) > 0 {
			resultado[symbols[0]] = prices
		}
		return resultado, nil
	}

	// Múltiples símbolos: la respuesta es un mapa de objetos
	var batchResp map[string]twelveDataResponse
	if err := json.Unmarshal(body, &batchResp); err != nil {
		return nil, fmt.Errorf("error decodificando batch: %w", err)
	}

	for sym, tdResp := range batchResp {
		if tdResp.Status == "error" {
			log.Printf("⚠️ [%s] Error en batch: %s", sym, tdResp.Msg)
			continue
		}
		prices := convertirValoresTD(tdResp.Values, days)
		if len(prices) > 0 {
			resultado[sym] = prices
		}
	}

	return resultado, nil
}

// ============================================
// Utilidades de conversión
// ============================================

// convertirValoresTD convierte valores de Twelve Data a nuestro modelo Price
func convertirValoresTD(values []twelveDataValue, maxDays int) []models.Price {
	prices := make([]models.Price, 0, len(values))

	for _, v := range values {
		date, err := time.Parse("2006-01-02", v.Datetime)
		if err != nil {
			continue
		}

		openVal, _ := strconv.ParseFloat(v.Open, 64)
		highVal, _ := strconv.ParseFloat(v.High, 64)
		lowVal, _ := strconv.ParseFloat(v.Low, 64)
		closeVal, _ := strconv.ParseFloat(v.Close, 64)
		volumeVal, _ := strconv.ParseInt(v.Volume, 10, 64)

		if closeVal == 0 || openVal == 0 {
			continue
		}

		prices = append(prices, models.Price{
			Open:   math.Round(openVal*100) / 100,
			High:   math.Round(highVal*100) / 100,
			Low:    math.Round(lowVal*100) / 100,
			Close:  math.Round(closeVal*100) / 100,
			Volume: volumeVal,
			Date:   date,
		})
	}

	// Ordenar cronológicamente (antiguo → reciente)
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Date.Before(prices[j].Date)
	})

	// Limitar a los últimos N días
	if len(prices) > maxDays {
		prices = prices[len(prices)-maxDays:]
	}

	return prices
}

// ============================================
// Descarga de velas históricas (multi-timeframe)
// Usado por el CandleDownloader para llenar la BD
// ============================================

// FetchHistoricalCandles descarga velas históricas de un símbolo en cualquier timeframe
// Soporta paginación usando endDate para ir hacia atrás en el tiempo
// Retorna las velas ordenadas de más antigua a más reciente
func (p *TwelveDataProvider) FetchHistoricalCandles(ctx context.Context, symbol, interval string, outputSize int, endDate string) ([]models.Candle, error) {
	// Rate limit
	if err := p.esperarRateLimit(); err != nil {
		return nil, fmt.Errorf("[%s/%s] rate limit: %w", symbol, interval, err)
	}

	// Construir URL
	url := fmt.Sprintf(
		"https://api.twelvedata.com/time_series?symbol=%s&interval=%s&outputsize=%d&apikey=%s",
		symbol, interval, outputSize, p.apiKey,
	)
	if endDate != "" {
		url += "&end_date=" + endDate
	}

	// Petición HTTP
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando petición: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en petición HTTP: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %w", err)
	}

	p.registrarCredito()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body[:minInt(len(body), 200)]))
	}

	// Decodificar respuesta
	var tdResp twelveDataResponse
	if err := json.Unmarshal(body, &tdResp); err != nil {
		return nil, fmt.Errorf("error decodificando JSON: %w", err)
	}

	if tdResp.Status == "error" {
		return nil, fmt.Errorf("Twelve Data error (código %d): %s", tdResp.Code, tdResp.Msg)
	}

	// Convertir a modelo Candle
	candles := make([]models.Candle, 0, len(tdResp.Values))
	for _, v := range tdResp.Values {
		// Intentar formato intraday primero, luego diario
		date, err := time.Parse("2006-01-02 15:04:05", v.Datetime)
		if err != nil {
			date, err = time.Parse("2006-01-02", v.Datetime)
			if err != nil {
				continue
			}
		}

		openVal, _ := strconv.ParseFloat(v.Open, 64)
		highVal, _ := strconv.ParseFloat(v.High, 64)
		lowVal, _ := strconv.ParseFloat(v.Low, 64)
		closeVal, _ := strconv.ParseFloat(v.Close, 64)
		volumeVal, _ := strconv.ParseInt(v.Volume, 10, 64)

		if closeVal == 0 || openVal == 0 {
			continue
		}

		candles = append(candles, models.Candle{
			Open:   math.Round(openVal*10000) / 10000,
			High:   math.Round(highVal*10000) / 10000,
			Low:    math.Round(lowVal*10000) / 10000,
			Close:  math.Round(closeVal*10000) / 10000,
			Volume: volumeVal,
			Date:   date,
		})
	}

	// Ordenar cronológicamente (antiguo → reciente)
	sort.Slice(candles, func(i, j int) bool {
		return candles[i].Date.Before(candles[j].Date)
	})

	return candles, nil
}

// minInt retorna el menor de dos enteros (helper)
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
