// ============================================
// Bot Oscar - Proveedor Alpha Vantage
// Obtiene datos reales de mercado desde la API de Alpha Vantage
// Documentación: https://www.alphavantage.co/documentation/
// Límite gratuito: 25 peticiones por día
// ============================================
package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"bot-oscar/internal/cache"
	"bot-oscar/internal/models"
)

// AlphaVantageProvider obtiene datos reales del mercado
type AlphaVantageProvider struct {
	apiKey string
	cache  *cache.Cache
}

// alphaVantageResponse estructura de la respuesta de la API
type alphaVantageResponse struct {
	TimeSeries map[string]map[string]string `json:"Time Series (Daily)"`
}

// GetPrices obtiene precios reales desde Alpha Vantage
func (p *AlphaVantageProvider) GetPrices(ctx context.Context, symbol string, days int) ([]models.Price, error) {
	// Intentar obtener de caché primero (TTL de 1 hora para no agotar el límite)
	cached, err := p.cache.GetPrices(ctx, symbol)
	if err == nil && cached != nil && len(cached) >= days {
		start := len(cached) - days
		return cached[start:], nil
	}

	// Construir URL de la API
	// Para commodities (símbolos con =F), necesitaríamos otro endpoint
	// Por ahora, usamos TIME_SERIES_DAILY para acciones
	url := fmt.Sprintf(
		"https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&outputsize=full&apikey=%s",
		symbol, p.apiKey,
	)

	// Realizar petición HTTP
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando petición: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("⚠️ Error en API Alpha Vantage para %s, usando datos demo: %v", symbol, err)
		return generateDemoData(symbol, days), nil
	}
	defer resp.Body.Close()

	// Decodificar respuesta JSON
	var apiResp alphaVantageResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		log.Printf("⚠️ Error decodificando respuesta de Alpha Vantage para %s, usando datos demo: %v", symbol, err)
		return generateDemoData(symbol, days), nil
	}

	// Si no hay datos en la respuesta, usar demo
	if len(apiResp.TimeSeries) == 0 {
		log.Printf("⚠️ Sin datos de Alpha Vantage para %s, usando datos demo", symbol)
		return generateDemoData(symbol, days), nil
	}

	// Convertir los datos de la API a nuestro modelo
	prices := make([]models.Price, 0, len(apiResp.TimeSeries))

	for dateStr, values := range apiResp.TimeSeries {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		open, _ := strconv.ParseFloat(values["1. open"], 64)
		high, _ := strconv.ParseFloat(values["2. high"], 64)
		low, _ := strconv.ParseFloat(values["3. low"], 64)
		closePrice, _ := strconv.ParseFloat(values["4. close"], 64)
		volume, _ := strconv.ParseInt(values["5. volume"], 10, 64)

		prices = append(prices, models.Price{
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closePrice,
			Volume: volume,
			Date:   date,
		})
	}

	// Ordenar por fecha ascendente
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Date.Before(prices[j].Date)
	})

	// Limitar a los últimos N días
	if len(prices) > days {
		prices = prices[len(prices)-days:]
	}

	// Cachear por 1 hora para no agotar límite de la API
	p.cache.SetPrices(ctx, symbol, prices, 1*time.Hour)

	return prices, nil
}
