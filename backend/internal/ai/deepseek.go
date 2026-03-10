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
	Symbol     string   `json:"symbol"`
	AssetName  string   `json:"assetName"`
	Signal     string   `json:"signal"`     // "COMPRA", "VENTA", "MANTENER"
	Confidence int      `json:"confidence"` // 0-100
	EntryPrice float64  `json:"entryPrice"`
	StopLoss   float64  `json:"stopLoss"`
	TakeProfit float64  `json:"takeProfit"`
	Timeframe  string   `json:"timeframe"`  // "corto", "medio", "largo"
	RiskLevel  string   `json:"riskLevel"`  // "bajo", "medio", "alto"
	Analysis   string   `json:"analysis"`   // Análisis detallado en español
	KeyFactors []string `json:"keyFactors"` // Factores clave de la decisión
	Disclaimer string   `json:"disclaimer"` // Advertencia legal
	Timestamp  string   `json:"timestamp"`
	Model      string   `json:"model"` // Modelo de IA usado
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

	// Últimas 20 velas para contexto de acción del precio (patrones)
	var recentCandles strings.Builder
	startIdx := len(prices) - 20
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

	// Determinar tendencia por EMAs (más importante que SMAs)
	emaTrend := "INDEFINIDA"
	if ind.EMA50 > 0 && ind.EMA200 > 0 {
		if ind.EMA50 > ind.EMA200 {
			emaTrend = "ALCISTA (Golden Cross EMA - EMA50 > EMA200)"
		} else {
			emaTrend = "BAJISTA (Death Cross EMA - EMA50 < EMA200)"
		}
	}

	// Posición del precio respecto a la EMA200
	ema200Pos := "Sin datos"
	if ind.EMA200 > 0 {
		distPct := ((currentPrice - ind.EMA200) / ind.EMA200) * 100
		if distPct > 0 {
			ema200Pos = fmt.Sprintf("ENCIMA (+%.1f%%) → mercado alcista", distPct)
		} else {
			ema200Pos = fmt.Sprintf("DEBAJO (%.1f%%) → mercado bajista", distPct)
		}
	}

	// Posición respecto a VWAP
	vwapPos := "Sin datos"
	if ind.VWAP > 0 {
		if currentPrice > ind.VWAP {
			vwapPos = "ENCIMA del VWAP (precio 'caro' vs promedio institucional)"
		} else {
			vwapPos = "DEBAJO del VWAP (precio 'barato' vs promedio institucional)"
		}
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

	// Análisis de volumen
	volAnalysis := "Sin datos"
	if ind.VolumenRatio > 0 {
		if ind.VolumenRatio > 2.0 {
			volAnalysis = fmt.Sprintf("PICO DE VOLUMEN MUY ALTO (%.1fx del promedio) → movimiento significativo", ind.VolumenRatio)
		} else if ind.VolumenRatio > 1.5 {
			volAnalysis = fmt.Sprintf("Volumen elevado (%.1fx del promedio) → confirma dirección", ind.VolumenRatio)
		} else if ind.VolumenRatio < 0.5 {
			volAnalysis = fmt.Sprintf("Volumen MUY BAJO (%.1fx del promedio) → movimiento débil/sin convicción", ind.VolumenRatio)
		} else {
			volAnalysis = fmt.Sprintf("Volumen normal (%.1fx del promedio)", ind.VolumenRatio)
		}
	}

	prompt := fmt.Sprintf(`Eres un trader profesional institucional con más de 20 años de experiencia en análisis técnico. Analiza estos datos REALES y genera una señal de trading precisa.

=== ACTIVO ===
Símbolo: %s
Nombre: %s
Tipo: %s
Precio actual: $%.2f
Cambio diario: %.2f%%

=== TENDENCIA PRINCIPAL (LO MÁS IMPORTANTE) ===
EMA(200): $%.2f → Precio %s
EMA(50):  $%.2f
EMA(21):  $%.2f (pullbacks)
Tendencia EMA: %s
SMA(50):  $%.2f
SMA(200): $%.2f

=== MOMENTUM ===
RSI(14): %.2f %s
MACD: %.4f | Señal: %.4f | Histograma: %.4f
EMA(12): $%.2f | EMA(26): $%.2f (cruce: %s)

=== VOLUMEN INSTITUCIONAL ===
VWAP(20): $%.2f → Precio %s
Volumen hoy: %d | Promedio 20d: %d
Ratio volumen: %s

=== VOLATILIDAD ===
ATR(14): $%.2f (volatilidad diaria real)
Bollinger Superior: $%.2f
Bollinger Media:    $%.2f
Bollinger Inferior: $%.2f
Posición Bollinger: %s

=== PUNTUACIÓN DEL SISTEMA ===
Confluencia: %d/100 → %s

=== ÚLTIMAS 20 VELAS DIARIAS (para detectar patrones) ===
%s
=== INSTRUCCIONES ===
1. La EMA200 determina la tendencia principal. Si el precio está encima = solo buscar COMPRAS. Si está debajo = solo buscar VENTAS.
2. Usa VWAP para confirmar si el precio está caro o barato para institucionales.
3. ATR te dice la volatilidad real - úsalo para SL/TP realistas.
4. El volumen confirma o invalida el movimiento. Sin volumen = señal débil.
5. Busca confluencia de al menos 3-4 indicadores alineados.
6. Sé HONESTO: si hay contradicción, di "MANTENER".
7. SL basado en ATR (2x ATR). TP con ratio mínimo 1:2.

=== REGLA CRÍTICA: STOP LOSS Y TAKE PROFIT OBLIGATORIOS ===
SIEMPRE proporciona valores numéricos concretos (mayor que 0) para stopLoss y takeProfit.
- COMPRA: stopLoss = soporte (EMA200, banda inferior Bollinger, o mínimo reciente - usa ATR como referencia). takeProfit = resistencia con ratio 1:2.
- VENTA: stopLoss = resistencia (EMA200, banda superior Bollinger, o máximo reciente). takeProfit = soporte con ratio 1:2.
- MANTENER: stopLoss = nivel de invalidación. takeProfit = nivel donde actuarías.

Responde EXCLUSIVAMENTE en JSON puro (sin markdown, sin backticks):
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
		// Activo
		asset.Symbol, asset.Name, asset.Type, currentPrice, dailyChange,
		// Tendencia principal
		ind.EMA200, ema200Pos,
		ind.EMA50, ind.EMA21, emaTrend,
		ind.SMA50, ind.SMA200,
		// Momentum
		ind.RSI, rsiInterpretation(ind.RSI),
		ind.MACD.MACD, ind.MACD.Signal, ind.MACD.Histogram,
		ind.EMA12, ind.EMA26, emaCrossDescription(ind.EMA12, ind.EMA26),
		// Volumen
		ind.VWAP, vwapPos,
		ind.VolumenHoy, ind.VolumenProm, volAnalysis,
		// Volatilidad
		ind.ATR,
		ind.Bollinger.Upper, ind.Bollinger.Middle, ind.Bollinger.Lower, bollingerPos,
		// Sistema
		ind.Score, ind.Signal,
		// Velas
		recentCandles.String(),
	)

	return prompt
}

// emaCrossDescription describe el cruce EMA12/EMA26
func emaCrossDescription(ema12, ema26 float64) string {
	if ema12 > 0 && ema26 > 0 {
		if ema12 > ema26 {
			return "EMA12 > EMA26 → impulso alcista"
		}
		return "EMA12 < EMA26 → impulso bajista"
	}
	return "sin datos"
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
