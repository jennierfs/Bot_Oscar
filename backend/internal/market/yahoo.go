// ============================================
// Bot Oscar - Proveedor Yahoo Finance
// Obtiene datos REALES de mercado desde Yahoo Finance
// Ventajas:
//   - NO requiere API key
//   - Soporta acciones (LMT, RTX, BA...) Y commodities (GC=F, CL=F...)
//   - Límite generoso (miles de peticiones por día)
//   - Datos OHLCV diarios históricos
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
	"net/url"
	"sort"
	"time"

	"bot-oscar/internal/cache"
	"bot-oscar/internal/models"
)

// YahooFinanceProvider obtiene datos reales desde Yahoo Finance
type YahooFinanceProvider struct {
	cache *cache.Cache
}

// yahooChartResponse estructura de la respuesta del API de Yahoo Finance
type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol             string  `json:"symbol"`
				Currency           string  `json:"currency"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []interface{} `json:"open"`
					High   []interface{} `json:"high"`
					Low    []interface{} `json:"low"`
					Close  []interface{} `json:"close"`
					Volume []interface{} `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// GetPrices obtiene precios REALES desde Yahoo Finance
func (p *YahooFinanceProvider) GetPrices(ctx context.Context, symbol string, days int) ([]models.Price, error) {
	// 1. Intentar obtener de caché primero (TTL de 4 horas)
	cached, err := p.cache.GetPrices(ctx, symbol)
	if err == nil && cached != nil && len(cached) >= days {
		start := len(cached) - days
		log.Printf("📦 [%s] Datos obtenidos de caché (%d velas)", symbol, len(cached[start:]))
		return cached[start:], nil
	}

	// 2. Consultar Yahoo Finance
	prices, err := p.fetchFromYahoo(ctx, symbol, days)
	if err != nil {
		log.Printf("⚠️ [%s] Error en Yahoo Finance: %v", symbol, err)
		return nil, fmt.Errorf("error obteniendo datos de Yahoo Finance para %s: %w", symbol, err)
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("sin datos de Yahoo Finance para %s", symbol)
	}

	// 3. Cachear los datos obtenidos (4 horas para no saturar)
	if err := p.cache.SetPrices(ctx, symbol, prices, 4*time.Hour); err != nil {
		log.Printf("⚠️ [%s] Error guardando en caché: %v", symbol, err)
	}

	log.Printf("✅ [%s] %d velas reales obtenidas de Yahoo Finance", symbol, len(prices))
	return prices, nil
}

// fetchFromYahoo realiza la petición HTTP a Yahoo Finance
func (p *YahooFinanceProvider) fetchFromYahoo(ctx context.Context, symbol string, days int) ([]models.Price, error) {
	// Determinar el rango temporal basado en los días solicitados
	rangeParam := "1y" // 1 año por defecto
	if days > 365 {
		rangeParam = "2y"
	} else if days <= 30 {
		rangeParam = "3mo"
	} else if days <= 90 {
		rangeParam = "6mo"
	}

	// Construir URL - el símbolo se codifica para manejar el = de commodities (GC=F)
	encodedSymbol := url.PathEscape(symbol)
	apiURL := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?range=%s&interval=1d&includePrePost=false",
		encodedSymbol, rangeParam,
	)

	// Crear petición HTTP con User-Agent (Yahoo requiere uno válido)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando petición: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Ejecutar petición con timeout
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en petición HTTP: %w", err)
	}
	defer resp.Body.Close()

	// Leer el cuerpo de la respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Yahoo Finance respondió con código %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	// Decodificar JSON
	var yahooResp yahooChartResponse
	if err := json.Unmarshal(body, &yahooResp); err != nil {
		return nil, fmt.Errorf("error decodificando JSON: %w", err)
	}

	// Verificar errores en la respuesta
	if yahooResp.Chart.Error != nil {
		return nil, fmt.Errorf("error de Yahoo: %s - %s", yahooResp.Chart.Error.Code, yahooResp.Chart.Error.Description)
	}

	// Verificar que hay resultados
	if len(yahooResp.Chart.Result) == 0 {
		return nil, fmt.Errorf("sin resultados para %s", symbol)
	}

	result := yahooResp.Chart.Result[0]

	// Verificar que hay datos de precio
	if len(result.Indicators.Quote) == 0 || len(result.Timestamp) == 0 {
		return nil, fmt.Errorf("sin datos OHLCV para %s", symbol)
	}

	quote := result.Indicators.Quote[0]
	timestamps := result.Timestamp

	// Convertir a nuestro modelo de precios
	prices := make([]models.Price, 0, len(timestamps))

	for i, ts := range timestamps {
		// Extraer valores con protección contra nulos
		openVal := toFloat64(quote.Open, i)
		highVal := toFloat64(quote.High, i)
		lowVal := toFloat64(quote.Low, i)
		closeVal := toFloat64(quote.Close, i)
		volumeVal := toInt64(quote.Volume, i)

		// Saltar velas con datos incompletos (días sin mercado)
		if closeVal == 0 || openVal == 0 {
			continue
		}

		// Convertir timestamp Unix a fecha
		date := time.Unix(ts, 0).UTC()

		prices = append(prices, models.Price{
			Open:   math.Round(openVal*100) / 100,
			High:   math.Round(highVal*100) / 100,
			Low:    math.Round(lowVal*100) / 100,
			Close:  math.Round(closeVal*100) / 100,
			Volume: volumeVal,
			Date:   date,
		})
	}

	// Ordenar por fecha ascendente (debería venir así, pero por seguridad)
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
// Helpers para parsear valores de Yahoo Finance
// Yahoo puede devolver null en algunos campos
// ============================================

// toFloat64 extrae un float64 de un slice de interface{}, manejando nulos
func toFloat64(data []interface{}, index int) float64 {
	if index >= len(data) || data[index] == nil {
		return 0
	}
	switch v := data[index].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

// toInt64 extrae un int64 de un slice de interface{}, manejando nulos
func toInt64(data []interface{}, index int) int64 {
	if index >= len(data) || data[index] == nil {
		return 0
	}
	switch v := data[index].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case json.Number:
		i, _ := v.Int64()
		return i
	default:
		return 0
	}
}

// min devuelve el mínimo entre dos enteros
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
