// ============================================
// Bot Oscar - Sentimiento de Mercado (Yahoo Finance)
// Obtiene Analyst Ratings y Short Interest REALES para cada activo
//
// Fuentes de datos (100% gratis, sin API key):
//   - Yahoo Finance quoteSummary → recommendationTrend (Buy/Hold/Sell de analistas)
//   - Yahoo Finance quoteSummary → financialData (precio objetivo)
//   - Yahoo Finance quoteSummary → defaultKeyStatistics (short interest)
//
// Cada activo se consulta de forma INDEPENDIENTE.
// Los datos se cachean en Redis con TTL de 1 hora para no saturar Yahoo.
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
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"bot-oscar/internal/cache"
	"bot-oscar/internal/models"
)

// SentimentProvider obtiene datos de sentimiento desde Yahoo Finance
type SentimentProvider struct {
	cache  *cache.Cache
	client *http.Client

	// Yahoo Finance requiere crumb+cookie para la API v10
	mu          sync.Mutex
	crumb       string
	cookie      string
	crumbExpiry time.Time
}

// NewSentimentProvider crea un nuevo proveedor de sentimiento
func NewSentimentProvider(redisCache *cache.Cache) *SentimentProvider {
	jar, _ := cookiejar.New(nil)
	return &SentimentProvider{
		cache:  redisCache,
		client: &http.Client{Timeout: 30 * time.Second, Jar: jar},
	}
}

// GetSentiment obtiene el sentimiento completo de un activo (analyst ratings + short interest)
func (sp *SentimentProvider) GetSentiment(ctx context.Context, symbol string, assetName string) (*models.MarketSentiment, error) {
	// 1. Intentar obtener de caché (TTL 1 hora)
	cacheKey := fmt.Sprintf("sentiment:%s", symbol)
	var cached models.MarketSentiment
	if err := sp.cache.GetJSON(ctx, cacheKey, &cached); err == nil {
		log.Printf("📦 [%s] Sentimiento obtenido de caché", symbol)
		return &cached, nil
	}

	// 2. Consultar Yahoo Finance quoteSummary
	data, err := sp.fetchQuoteSummary(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo sentimiento de Yahoo Finance para %s: %w", symbol, err)
	}

	// 3. Parsear datos de analistas
	analystRatings := sp.parseAnalystRatings(data)

	// 4. Parsear short interest
	shortInterest := sp.parseShortInterest(data)

	// 5. Construir resultado
	result := &models.MarketSentiment{
		Symbol:         symbol,
		AssetName:      assetName,
		AnalystRatings: analystRatings,
		ShortInterest:  shortInterest,
		Summary:        sp.buildSummary(symbol, analystRatings, shortInterest),
		UpdatedAt:      time.Now().Format(time.RFC3339),
	}

	// 6. Cachear resultado (1 hora)
	if err := sp.cache.SetJSON(ctx, cacheKey, result, 1*time.Hour); err != nil {
		log.Printf("⚠️ [%s] Error cacheando sentimiento: %v", symbol, err)
	}

	log.Printf("✅ [%s] Sentimiento de mercado obtenido de Yahoo Finance", symbol)
	return result, nil
}

// ============================================
// Estructuras de respuesta de Yahoo Finance quoteSummary
// ============================================

type yahooQuoteSummaryResponse struct {
	QuoteSummary struct {
		Result []yahooQuoteSummaryResult `json:"result"`
		Error  *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"quoteSummary"`
}

type yahooQuoteSummaryResult struct {
	RecommendationTrend struct {
		Trend []struct {
			Period     string `json:"period"`
			StrongBuy  int    `json:"strongBuy"`
			Buy        int    `json:"buy"`
			Hold       int    `json:"hold"`
			Sell       int    `json:"sell"`
			StrongSell int    `json:"strongSell"`
		} `json:"trend"`
	} `json:"recommendationTrend"`

	FinancialData struct {
		CurrentPrice      yahooRawValue `json:"currentPrice"`
		TargetHighPrice   yahooRawValue `json:"targetHighPrice"`
		TargetLowPrice    yahooRawValue `json:"targetLowPrice"`
		TargetMeanPrice   yahooRawValue `json:"targetMeanPrice"`
		RecommendationKey string        `json:"recommendationKey"`
	} `json:"financialData"`

	DefaultKeyStatistics struct {
		SharesShort         yahooRawValue `json:"sharesShort"`
		ShortRatio          yahooRawValue `json:"shortRatio"`
		ShortPercentOfFloat yahooRawValue `json:"shortPercentOfFloat"`
		FloatShares         yahooRawValue `json:"floatShares"`
		SharesOutstanding   yahooRawValue `json:"sharesOutstanding"`
	} `json:"defaultKeyStatistics"`
}

type yahooRawValue struct {
	Raw float64 `json:"raw"`
	Fmt string  `json:"fmt"`
}

// ============================================
// Autenticación Yahoo Finance (Crumb + Cookie)
// Yahoo requiere obtener un crumb token antes de usar la API v10
// ============================================

const yahooUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// ensureCrumb obtiene o reutiliza el crumb de Yahoo Finance
func (sp *SentimentProvider) ensureCrumb(ctx context.Context) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Reutilizar crumb si todavía es válido (30 minutos)
	if sp.crumb != "" && time.Now().Before(sp.crumbExpiry) {
		return nil
	}

	log.Println("🔑 Obteniendo crumb de Yahoo Finance...")

	// Paso 1: Visitar una página para obtener cookies
	req1, err := http.NewRequestWithContext(ctx, "GET", "https://fc.yahoo.com/", nil)
	if err != nil {
		return fmt.Errorf("error creando petición de cookie: %w", err)
	}
	req1.Header.Set("User-Agent", yahooUserAgent)

	resp1, err := sp.client.Do(req1)
	if err != nil {
		return fmt.Errorf("error obteniendo cookie: %w", err)
	}
	io.ReadAll(resp1.Body)
	resp1.Body.Close()

	// Paso 2: Obtener el crumb usando las cookies recibidas
	req2, err := http.NewRequestWithContext(ctx, "GET", "https://query2.finance.yahoo.com/v1/test/getcrumb", nil)
	if err != nil {
		return fmt.Errorf("error creando petición de crumb: %w", err)
	}
	req2.Header.Set("User-Agent", yahooUserAgent)

	resp2, err := sp.client.Do(req2)
	if err != nil {
		return fmt.Errorf("error obteniendo crumb: %w", err)
	}
	defer resp2.Body.Close()

	crumbBytes, err := io.ReadAll(resp2.Body)
	if err != nil {
		return fmt.Errorf("error leyendo crumb: %w", err)
	}

	crumb := strings.TrimSpace(string(crumbBytes))
	if crumb == "" || resp2.StatusCode != http.StatusOK {
		return fmt.Errorf("crumb vacío o error HTTP %d", resp2.StatusCode)
	}

	sp.crumb = crumb
	sp.crumbExpiry = time.Now().Add(30 * time.Minute)
	log.Printf("✅ Crumb de Yahoo Finance obtenido: %s...", crumb[:min(len(crumb), 8)])
	return nil
}

// ============================================
// Petición HTTP a Yahoo Finance (con crumb)
// ============================================

func (sp *SentimentProvider) fetchQuoteSummary(ctx context.Context, symbol string) (*yahooQuoteSummaryResult, error) {
	// Asegurar que tenemos un crumb válido
	if err := sp.ensureCrumb(ctx); err != nil {
		return nil, fmt.Errorf("error obteniendo autenticación Yahoo: %w", err)
	}

	encodedSymbol := url.PathEscape(symbol)
	sp.mu.Lock()
	crumb := sp.crumb
	sp.mu.Unlock()

	apiURL := fmt.Sprintf(
		"https://query2.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=recommendationTrend,financialData,defaultKeyStatistics&crumb=%s",
		encodedSymbol, url.QueryEscape(crumb),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando petición: %w", err)
	}
	req.Header.Set("User-Agent", yahooUserAgent)

	resp, err := sp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en petición HTTP: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %w", err)
	}

	// Si recibimos 401, invalidar crumb y reintentar una vez
	if resp.StatusCode == http.StatusUnauthorized {
		log.Printf("⚠️ [%s] Crumb expirado, reintentando...", symbol)
		sp.mu.Lock()
		sp.crumb = ""
		sp.crumbExpiry = time.Time{}
		sp.mu.Unlock()

		if err := sp.ensureCrumb(ctx); err != nil {
			return nil, fmt.Errorf("error renovando crumb: %w", err)
		}

		sp.mu.Lock()
		crumb = sp.crumb
		sp.mu.Unlock()

		retryURL := fmt.Sprintf(
			"https://query2.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=recommendationTrend,financialData,defaultKeyStatistics&crumb=%s",
			encodedSymbol, url.QueryEscape(crumb),
		)

		req2, _ := http.NewRequestWithContext(ctx, "GET", retryURL, nil)
		req2.Header.Set("User-Agent", yahooUserAgent)

		resp2, err := sp.client.Do(req2)
		if err != nil {
			return nil, fmt.Errorf("error en reintento: %w", err)
		}
		defer resp2.Body.Close()

		body, err = io.ReadAll(resp2.Body)
		if err != nil {
			return nil, fmt.Errorf("error leyendo reintento: %w", err)
		}

		if resp2.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Yahoo Finance respondió con código %d en reintento", resp2.StatusCode)
		}
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Yahoo Finance respondió con código %d", resp.StatusCode)
	}

	var summaryResp yahooQuoteSummaryResponse
	if err := json.Unmarshal(body, &summaryResp); err != nil {
		return nil, fmt.Errorf("error decodificando JSON: %w", err)
	}

	if summaryResp.QuoteSummary.Error != nil {
		return nil, fmt.Errorf("error de Yahoo: %s", summaryResp.QuoteSummary.Error.Description)
	}

	if len(summaryResp.QuoteSummary.Result) == 0 {
		return nil, fmt.Errorf("sin resultados para %s", symbol)
	}

	return &summaryResp.QuoteSummary.Result[0], nil
}

// ============================================
// Parseo de Analyst Ratings
// ============================================

func (sp *SentimentProvider) parseAnalystRatings(data *yahooQuoteSummaryResult) *models.AnalystRatings {
	if data == nil {
		return nil
	}

	// Buscar el período actual (0m = mes actual)
	var strongBuy, buy, hold, sell, strongSell int
	found := false

	for _, trend := range data.RecommendationTrend.Trend {
		if trend.Period == "0m" {
			strongBuy = trend.StrongBuy
			buy = trend.Buy
			hold = trend.Hold
			sell = trend.Sell
			strongSell = trend.StrongSell
			found = true
			break
		}
	}

	if !found {
		// Si no hay período "0m", tomar el primero disponible
		if len(data.RecommendationTrend.Trend) > 0 {
			t := data.RecommendationTrend.Trend[0]
			strongBuy = t.StrongBuy
			buy = t.Buy
			hold = t.Hold
			sell = t.Sell
			strongSell = t.StrongSell
			found = true
		}
	}

	if !found || (strongBuy+buy+hold+sell+strongSell) == 0 {
		return nil
	}

	total := strongBuy + buy + hold + sell + strongSell

	buyPct := 0.0
	sellPct := 0.0
	if total > 0 {
		buyPct = math.Round(float64(strongBuy+buy)/float64(total)*1000) / 10
		sellPct = math.Round(float64(strongSell+sell)/float64(total)*1000) / 10
	}

	// Determinar consenso
	consensus := sp.determineConsensus(strongBuy, buy, hold, sell, strongSell, total)

	// Datos de precio objetivo
	currentPrice := data.FinancialData.CurrentPrice.Raw
	targetMean := data.FinancialData.TargetMeanPrice.Raw

	upsidePercent := 0.0
	if currentPrice > 0 && targetMean > 0 {
		upsidePercent = math.Round(((targetMean-currentPrice)/currentPrice)*1000) / 10
	}

	return &models.AnalystRatings{
		StrongBuy:     strongBuy,
		Buy:           buy,
		Hold:          hold,
		Sell:          sell,
		StrongSell:    strongSell,
		Total:         total,
		BuyPercent:    buyPct,
		SellPercent:   sellPct,
		Consensus:     consensus,
		TargetHigh:    data.FinancialData.TargetHighPrice.Raw,
		TargetLow:     data.FinancialData.TargetLowPrice.Raw,
		TargetMean:    targetMean,
		CurrentPrice:  currentPrice,
		UpsidePercent: upsidePercent,
	}
}

func (sp *SentimentProvider) determineConsensus(strongBuy, buy, hold, sell, strongSell, total int) string {
	if total == 0 {
		return "Sin datos"
	}

	buyTotal := strongBuy + buy
	sellTotal := strongSell + sell
	buyPct := float64(buyTotal) / float64(total)
	sellPct := float64(sellTotal) / float64(total)

	if buyPct >= 0.7 && strongBuy > buy {
		return "Compra Fuerte"
	}
	if buyPct >= 0.5 {
		return "Compra"
	}
	if sellPct >= 0.7 && strongSell > sell {
		return "Venta Fuerte"
	}
	if sellPct >= 0.5 {
		return "Venta"
	}
	return "Mantener"
}

// ============================================
// Parseo de Short Interest
// ============================================

func (sp *SentimentProvider) parseShortInterest(data *yahooQuoteSummaryResult) *models.ShortInterest {
	if data == nil {
		return nil
	}

	stats := data.DefaultKeyStatistics
	sharesShort := int64(stats.SharesShort.Raw)
	floatShares := int64(stats.FloatShares.Raw)
	shortRatio := stats.ShortRatio.Raw
	shortPctFloat := stats.ShortPercentOfFloat.Raw * 100 // Yahoo lo da en decimal (0.05 = 5%)

	// Si no hay datos suficientes, retornar nil
	if sharesShort == 0 && shortRatio == 0 && shortPctFloat == 0 {
		return nil
	}

	// Redondear
	shortPctFloat = math.Round(shortPctFloat*100) / 100
	shortRatio = math.Round(shortRatio*100) / 100

	// Determinar nivel
	level := "Bajo"
	if shortPctFloat >= 20 {
		level = "Muy Alto"
	} else if shortPctFloat >= 10 {
		level = "Alto"
	} else if shortPctFloat >= 5 {
		level = "Moderado"
	}

	return &models.ShortInterest{
		ShortPercentOfFloat: shortPctFloat,
		ShortRatio:          shortRatio,
		SharesShort:         sharesShort,
		SharesFloat:         floatShares,
		Level:               level,
	}
}

// ============================================
// Resumen en español
// ============================================

func (sp *SentimentProvider) buildSummary(symbol string, ratings *models.AnalystRatings, shortInt *models.ShortInterest) string {
	parts := make([]string, 0, 3)

	if ratings != nil && ratings.Total > 0 {
		parts = append(parts, fmt.Sprintf(
			"%d analistas: %d%% recomiendan COMPRA, %d%% VENTA → Consenso: %s",
			ratings.Total, int(ratings.BuyPercent), int(ratings.SellPercent), ratings.Consensus,
		))

		if ratings.TargetMean > 0 && ratings.CurrentPrice > 0 {
			direction := "subida"
			if ratings.UpsidePercent < 0 {
				direction = "bajada"
			}
			parts = append(parts, fmt.Sprintf(
				"Precio objetivo medio: $%.2f (%.1f%% de %s desde $%.2f)",
				ratings.TargetMean, math.Abs(ratings.UpsidePercent), direction, ratings.CurrentPrice,
			))
		}
	}

	if shortInt != nil && shortInt.ShortPercentOfFloat > 0 {
		parts = append(parts, fmt.Sprintf(
			"Short Interest: %.2f%% del float (%s) — Días para cubrir: %.1f",
			shortInt.ShortPercentOfFloat, shortInt.Level, shortInt.ShortRatio,
		))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Sin datos de sentimiento disponibles para %s", symbol)
	}

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " | "
		}
		result += p
	}
	return result
}
