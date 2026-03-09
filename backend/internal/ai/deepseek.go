// ============================================
// Bot Oscar - Cliente DeepSeek AI
// Envía datos de indicadores técnicos a DeepSeek para obtener
// un análisis profesional de trading interpretado por IA.
//
// IMPORTANTE: DeepSeek NO predice el mercado. Lo que hace es
// interpretar los indicadores técnicos como lo haría un analista
// con experiencia, condensando RSI, MACD, Bollinger, SMA en una
// opinión estructurada con niveles de entrada, SL y TP.
// ============================================
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"bot-oscar/internal/models"
)

// DeepSeekClient maneja la comunicación con la API de DeepSeek
type DeepSeekClient struct {
	apiKey     string
	httpClient *http.Client
}

// AISignalResponse es la respuesta estructurada que devolvemos al frontend
type AISignalResponse struct {
	Symbol     string  `json:"symbol"`
	AssetName  string  `json:"assetName"`
	Signal     string  `json:"signal"`      // "COMPRA", "VENTA", "MANTENER"
	Confidence int     `json:"confidence"`   // 0-100
	EntryPrice float64 `json:"entryPrice"`
	StopLoss   float64 `json:"stopLoss"`
	TakeProfit float64 `json:"takeProfit"`
	Timeframe  string  `json:"timeframe"`    // "corto", "medio", "largo"
	RiskLevel  string  `json:"riskLevel"`    // "bajo", "medio", "alto"
	Analysis   string  `json:"analysis"`     // Análisis detallado en español
	KeyFactors []string `json:"keyFactors"`  // Factores clave de la decisión
	Disclaimer string  `json:"disclaimer"`   // Advertencia legal
	Timestamp  string  `json:"timestamp"`
	Model      string  `json:"model"`        // Modelo de IA usado
}

// deepseekRequest estructura de petición al API de DeepSeek (compatible OpenAI)
type deepseekRequest struct {
	Model       string            `json:"model"`
	Messages    []deepseekMessage `json:"messages"`
	Temperature float64           `json:"temperature"`
	MaxTokens   int               `json:"max_tokens"`
}

// deepseekMessage estructura de un mensaje en el chat
type deepseekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// deepseekResponse estructura de respuesta del API
type deepseekResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// NewDeepSeekClient crea un nuevo cliente de DeepSeek
func NewDeepSeekClient(apiKey string) *DeepSeekClient {
	return &DeepSeekClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // DeepSeek puede tardar en responder
		},
	}
}

// IsConfigured verifica si el cliente tiene una API key válida
func (c *DeepSeekClient) IsConfigured() bool {
	return c.apiKey != "" && c.apiKey != "tu_api_key_aqui"
}

// GenerateSignal envía los datos técnicos a DeepSeek y obtiene un análisis
func (c *DeepSeekClient) GenerateSignal(
	ctx context.Context,
	asset models.Asset,
	indicators *models.IndicatorValues,
	prices []models.Price,
) (*AISignalResponse, error) {

	if !c.IsConfigured() {
		return nil, fmt.Errorf("DeepSeek API key no configurada. Configura DEEPSEEK_API_KEY en .env")
	}

	if indicators == nil {
		return nil, fmt.Errorf("no hay indicadores calculados para %s", asset.Symbol)
	}

	// Construir el prompt con TODOS los datos numéricos reales
	prompt := buildAnalysisPrompt(asset, indicators, prices)

	log.Printf("🧠 [%s] Enviando datos a DeepSeek para análisis IA...", asset.Symbol)

	// Llamar a la API de DeepSeek
	response, err := c.callAPI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("error llamando a DeepSeek: %w", err)
	}

	// Parsear la respuesta JSON de DeepSeek
	aiSignal, err := parseAIResponse(response, asset, indicators)
	if err != nil {
		log.Printf("⚠️ [%s] Error parseando respuesta IA, usando respuesta cruda", asset.Symbol)
		// Si no puede parsear JSON, devolver análisis en texto plano con SL/TP calculados
		entry := prices[len(prices)-1].Close
		bollingerW := indicators.Bollinger.Upper - indicators.Bollinger.Lower
		margin := bollingerW * 0.25
		if margin < entry*0.01 {
			margin = entry * 0.02
		}
		fallbackSL := indicators.Bollinger.Lower
		if indicators.SMA200 > 0 && indicators.SMA200 < entry {
			fallbackSL = indicators.SMA200
		}
		if fallbackSL >= entry || fallbackSL <= 0 {
			fallbackSL = entry - margin
		}
		fallbackTP := indicators.Bollinger.Upper
		if fallbackTP <= entry {
			fallbackTP = entry + margin
		}
		return &AISignalResponse{
			Symbol:     asset.Symbol,
			AssetName:  asset.Name,
			Signal:     indicators.Signal,
			Confidence: indicators.Score,
			EntryPrice: math.Round(entry*100) / 100,
			StopLoss:   math.Round(fallbackSL*100) / 100,
			TakeProfit: math.Round(fallbackTP*100) / 100,
			Timeframe:  "medio",
			RiskLevel:  "medio",
			Analysis:   response,
			KeyFactors: []string{"Análisis generado en texto libre por la IA"},
			Disclaimer: "⚠️ Esto NO es asesoría financiera. Es una interpretación de IA sobre indicadores técnicos.",
			Timestamp:  time.Now().Format(time.RFC3339),
			Model:      "deepseek-chat",
		}, nil
	}

	log.Printf("✅ [%s] Señal IA generada: %s (confianza: %d%%)", asset.Symbol, aiSignal.Signal, aiSignal.Confidence)
	return aiSignal, nil
}

// buildAnalysisPrompt construye el prompt con datos reales para DeepSeek
func buildAnalysisPrompt(asset models.Asset, ind *models.IndicatorValues, prices []models.Price) string {
	// Calcular datos adicionales de contexto
	currentPrice := prices[len(prices)-1].Close
	prevClose := prices[len(prices)-2].Close
	dailyChange := ((currentPrice - prevClose) / prevClose) * 100

	// Últimas 5 velas para contexto de acción del precio
	var recentCandles strings.Builder
	startIdx := len(prices) - 5
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < len(prices); i++ {
		p := prices[i]
		direction := "↑"
		if p.Close < p.Open {
			direction = "↓"
		}
		recentCandles.WriteString(fmt.Sprintf(
			"  %s: O=%.2f H=%.2f L=%.2f C=%.2f V=%d %s\n",
			p.Date.Format("2006-01-02"), p.Open, p.High, p.Low, p.Close, p.Volume, direction,
		))
	}

	// Determinar tendencia SMA
	smaTrend := "INDEFINIDA"
	if ind.SMA50 > ind.SMA200 {
		smaTrend = "ALCISTA (Golden Cross - SMA50 > SMA200)"
	} else if ind.SMA50 < ind.SMA200 {
		smaTrend = "BAJISTA (Death Cross - SMA50 < SMA200)"
	}

	// Posición respecto a Bollinger
	bollingerPos := "MEDIA"
	if currentPrice > ind.Bollinger.Upper {
		bollingerPos = "POR ENCIMA de banda superior (sobrecompra)"
	} else if currentPrice < ind.Bollinger.Lower {
		bollingerPos = "POR DEBAJO de banda inferior (sobreventa)"
	} else if currentPrice > ind.Bollinger.Middle {
		bollingerPos = "Entre media y banda superior"
	} else {
		bollingerPos = "Entre banda inferior y media"
	}

	prompt := fmt.Sprintf(`Eres un trader profesional con más de 20 años de experiencia en análisis técnico de mercados financieros. Analiza los siguientes datos REALES de mercado y genera una señal de trading.

=== ACTIVO ===
Símbolo: %s
Nombre: %s
Tipo: %s
Precio actual: $%.2f
Cambio diario: %.2f%%

=== INDICADORES TÉCNICOS (CALCULADOS CON DATOS REALES) ===
RSI(14): %.2f %s
MACD: %.4f | Señal MACD: %.4f | Histograma: %.4f
SMA(50): $%.2f
SMA(200): $%.2f
Tendencia SMA: %s
EMA(12): $%.2f
EMA(26): $%.2f
Bollinger Superior: $%.2f
Bollinger Media: $%.2f  
Bollinger Inferior: $%.2f
Posición Bollinger: %s
Puntuación de confluencia del sistema: %d/100 → %s

=== ÚLTIMAS 5 VELAS DIARIAS ===
%s
=== INSTRUCCIONES ===
1. Analiza TODOS los indicadores en conjunto (confluencia)
2. Identifica la tendencia principal y la fuerza de la misma
3. Determina niveles de soporte y resistencia basados en los datos
4. Genera una señal con niveles específicos de entrada, stop loss y take profit
5. Sé HONESTO: si los indicadores son contradictorios, di "MANTENER"
6. El stop loss debe ser realista basado en la volatilidad (Bollinger width)
7. El take profit debe respetar un ratio riesgo/beneficio mínimo de 1:2

=== REGLA CRÍTICA: STOP LOSS Y TAKE PROFIT OBLIGATORIOS ===
SIEMPRE debes proporcionar valores numéricos concretos (mayor que 0) para stopLoss y takeProfit. NUNCA devuelvas 0 en esos campos.
- Para señal COMPRA: stopLoss = soporte más cercano (banda inferior Bollinger, SMA200 o mínimo reciente). takeProfit = resistencia más cercana (banda superior Bollinger o máximo reciente) con ratio mínimo 1:2.
- Para señal VENTA: stopLoss = resistencia más cercana (banda superior Bollinger, SMA50 o máximo reciente). takeProfit = soporte más cercano (banda inferior Bollinger o mínimo reciente) con ratio mínimo 1:2.
- Para señal MANTENER: Calcula IGUALMENTE los niveles donde COMPRARÍAS (soporte clave = takeProfit si baja) y donde VENDERÍAS (resistencia clave = takeProfit si sube). El stopLoss será el nivel de invalidación de la estructura actual. Esto ayuda al trader a saber DÓNDE actuar cuando el precio se mueva.
- Usa los datos de Bollinger Bands, SMA50, SMA200, mínimos y máximos de las últimas velas para calcular estos niveles.

Responde EXCLUSIVAMENTE en formato JSON con esta estructura (sin markdown, sin backticks, solo JSON puro):
{
  "signal": "COMPRA" | "VENTA" | "MANTENER",
  "confidence": <número 0-100>,
  "entryPrice": <precio de entrada sugerido>,
  "stopLoss": <nivel de stop loss concreto, SIEMPRE mayor que 0>,
  "takeProfit": <nivel de take profit concreto, SIEMPRE mayor que 0>,
  "timeframe": "corto" | "medio" | "largo",
  "riskLevel": "bajo" | "medio" | "alto",
  "analysis": "<análisis detallado en español, máximo 3 párrafos>",
  "keyFactors": ["factor1", "factor2", "factor3"]
}`,
		asset.Symbol,
		asset.Name,
		asset.Type,
		currentPrice,
		dailyChange,
		ind.RSI, rsiInterpretation(ind.RSI),
		ind.MACD.MACD, ind.MACD.Signal, ind.MACD.Histogram,
		ind.SMA50,
		ind.SMA200,
		smaTrend,
		ind.EMA12,
		ind.EMA26,
		ind.Bollinger.Upper,
		ind.Bollinger.Middle,
		ind.Bollinger.Lower,
		bollingerPos,
		ind.Score, ind.Signal,
		recentCandles.String(),
	)

	return prompt
}

// rsiInterpretation devuelve la interpretación textual del RSI
func rsiInterpretation(rsi float64) string {
	switch {
	case rsi >= 80:
		return "(MUY SOBRECOMPRADO - señal fuerte de venta)"
	case rsi >= 70:
		return "(SOBRECOMPRADO - posible corrección)"
	case rsi >= 60:
		return "(Alcista moderado)"
	case rsi >= 40:
		return "(NEUTRAL)"
	case rsi >= 30:
		return "(Bajista moderado)"
	case rsi >= 20:
		return "(SOBREVENDIDO - posible rebote)"
	default:
		return "(MUY SOBREVENDIDO - señal fuerte de compra)"
	}
}

// callAPI realiza la petición HTTP a la API de DeepSeek
func (c *DeepSeekClient) callAPI(ctx context.Context, prompt string) (string, error) {
	reqBody := deepseekRequest{
		Model: "deepseek-chat",
		Messages: []deepseekMessage{
			{
				Role:    "system",
				Content: "Eres un analista técnico de mercados financieros experto con más de 20 años de experiencia. Solo respondes en JSON válido. Eres honesto y realista, nunca garantizas ganancias. Si los datos son ambiguos, recomiendas MANTENER.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3, // Baja temperatura para respuestas más consistentes y menos creativas
		MaxTokens:   1500,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error serializando petición: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.deepseek.com/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error creando petición HTTP: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error en petición a DeepSeek: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error leyendo respuesta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DeepSeek respondió con código %d: %s", resp.StatusCode, string(body[:minInt(len(body), 500)]))
	}

	var dsResp deepseekResponse
	if err := json.Unmarshal(body, &dsResp); err != nil {
		return "", fmt.Errorf("error decodificando respuesta: %w", err)
	}

	if dsResp.Error != nil {
		return "", fmt.Errorf("error de DeepSeek: %s (%s)", dsResp.Error.Message, dsResp.Error.Type)
	}

	if len(dsResp.Choices) == 0 {
		return "", fmt.Errorf("DeepSeek no generó respuesta")
	}

	content := strings.TrimSpace(dsResp.Choices[0].Message.Content)
	log.Printf("🧠 DeepSeek respondió (%d tokens usados)", dsResp.Usage.TotalTokens)

	return content, nil
}

// parseAIResponse intenta parsear la respuesta JSON de DeepSeek
func parseAIResponse(raw string, asset models.Asset, indicators *models.IndicatorValues) (*AISignalResponse, error) {
	// Limpiar posibles backticks de markdown que DeepSeek a veces incluye
	cleaned := raw
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	// Intentar parsear el JSON
	var parsed struct {
		Signal     string   `json:"signal"`
		Confidence int      `json:"confidence"`
		EntryPrice float64  `json:"entryPrice"`
		StopLoss   float64  `json:"stopLoss"`
		TakeProfit float64  `json:"takeProfit"`
		Timeframe  string   `json:"timeframe"`
		RiskLevel  string   `json:"riskLevel"`
		Analysis   string   `json:"analysis"`
		KeyFactors []string `json:"keyFactors"`
	}

	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, fmt.Errorf("JSON inválido de DeepSeek: %w", err)
	}

	// Validar que los campos críticos tengan sentido
	if parsed.Signal != "COMPRA" && parsed.Signal != "VENTA" && parsed.Signal != "MANTENER" {
		parsed.Signal = "MANTENER"
	}
	if parsed.Confidence < 0 || parsed.Confidence > 100 {
		parsed.Confidence = 50
	}
	if parsed.EntryPrice <= 0 {
		parsed.EntryPrice = indicators.SMA50 // Fallback al SMA50
	}

	// === FALLBACK: Calcular SL/TP si DeepSeek devuelve 0 ===
	if parsed.StopLoss <= 0 || parsed.TakeProfit <= 0 {
		log.Printf("⚠️ [%s] DeepSeek no calculó SL/TP, usando cálculo automático por Bollinger/SMA", asset.Symbol)
		calculateFallbackLevels(&parsed, indicators)
	}

	return &AISignalResponse{
		Symbol:     asset.Symbol,
		AssetName:  asset.Name,
		Signal:     parsed.Signal,
		Confidence: parsed.Confidence,
		EntryPrice: parsed.EntryPrice,
		StopLoss:   parsed.StopLoss,
		TakeProfit: parsed.TakeProfit,
		Timeframe:  parsed.Timeframe,
		RiskLevel:  parsed.RiskLevel,
		Analysis:   parsed.Analysis,
		KeyFactors: parsed.KeyFactors,
		Disclaimer: "⚠️ Esto NO es asesoría financiera. Es una interpretación de IA basada en indicadores técnicos. Opera bajo tu propio riesgo.",
		Timestamp:  time.Now().Format(time.RFC3339),
		Model:      "deepseek-chat",
	}, nil
}

// calculateFallbackLevels calcula SL y TP automáticamente cuando DeepSeek no los proporciona
// Usa Bollinger Bands, SMAs y la volatilidad implícita para niveles realistas
func calculateFallbackLevels(parsed *struct {
	Signal     string   `json:"signal"`
	Confidence int      `json:"confidence"`
	EntryPrice float64  `json:"entryPrice"`
	StopLoss   float64  `json:"stopLoss"`
	TakeProfit float64  `json:"takeProfit"`
	Timeframe  string   `json:"timeframe"`
	RiskLevel  string   `json:"riskLevel"`
	Analysis   string   `json:"analysis"`
	KeyFactors []string `json:"keyFactors"`
}, ind *models.IndicatorValues) {
	entry := parsed.EntryPrice

	// Ancho de Bollinger como proxy de volatilidad
	bollingerWidth := ind.Bollinger.Upper - ind.Bollinger.Lower
	volatilityMargin := bollingerWidth * 0.25 // 25% del ancho de Bollinger

	if volatilityMargin < entry*0.01 {
		volatilityMargin = entry * 0.02 // Mínimo 2% del precio
	}

	switch parsed.Signal {
	case "COMPRA":
		// SL: el mayor entre banda inferior de Bollinger y SMA200 (soporte fuerte)
		if parsed.StopLoss <= 0 {
			sl := ind.Bollinger.Lower
			if ind.SMA200 > 0 && ind.SMA200 > sl && ind.SMA200 < entry {
				sl = ind.SMA200 // SMA200 como soporte más relevante
			}
			if sl >= entry {
				sl = entry - volatilityMargin // Fallback por volatilidad
			}
			parsed.StopLoss = math.Round(sl*100) / 100
		}
		// TP: banda superior Bollinger o ratio 1:2 sobre el riesgo
		if parsed.TakeProfit <= 0 {
			risk := entry - parsed.StopLoss
			tp := entry + (risk * 2) // Ratio 1:2
			if ind.Bollinger.Upper > tp {
				tp = ind.Bollinger.Upper // Usar Bollinger si da más
			}
			parsed.TakeProfit = math.Round(tp*100) / 100
		}

	case "VENTA":
		// SL: el menor entre banda superior Bollinger y SMA50
		if parsed.StopLoss <= 0 {
			sl := ind.Bollinger.Upper
			if ind.SMA50 > 0 && ind.SMA50 < sl && ind.SMA50 > entry {
				sl = ind.SMA50
			}
			if sl <= entry {
				sl = entry + volatilityMargin
			}
			parsed.StopLoss = math.Round(sl*100) / 100
		}
		// TP: banda inferior Bollinger o ratio 1:2
		if parsed.TakeProfit <= 0 {
			risk := parsed.StopLoss - entry
			tp := entry - (risk * 2)
			if ind.Bollinger.Lower < tp && ind.Bollinger.Lower > 0 {
				tp = ind.Bollinger.Lower
			}
			if tp <= 0 {
				tp = entry - volatilityMargin*2
			}
			parsed.TakeProfit = math.Round(tp*100) / 100
		}

	default: // MANTENER
		// SL: nivel de invalidación = banda inferior o SMA200 (soporte clave)
		if parsed.StopLoss <= 0 {
			sl := ind.Bollinger.Lower
			if ind.SMA200 > 0 && ind.SMA200 < entry {
				// Usar SMA200 si está por debajo (soporte estructural)
				sl = ind.SMA200
			}
			if sl >= entry || sl <= 0 {
				sl = entry - volatilityMargin
			}
			parsed.StopLoss = math.Round(sl*100) / 100
		}
		// TP: resistencia clave = banda superior o SMA50 (el que esté más arriba)
		if parsed.TakeProfit <= 0 {
			tp := ind.Bollinger.Upper
			if ind.SMA50 > tp {
				tp = ind.SMA50
			}
			if tp <= entry {
				tp = entry + volatilityMargin
			}
			parsed.TakeProfit = math.Round(tp*100) / 100
		}
	}
}

// minInt devuelve el mínimo entre dos enteros
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
