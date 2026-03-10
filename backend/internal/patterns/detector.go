// ============================================
// Bot Oscar - Detector de Patrones de Velas (Candlestick Patterns)
//
// Detecta patrones clásicos de velas japonesas usando EXCLUSIVAMENTE
// los datos del activo que se está analizando. NUNCA mezcla datos
// de un activo con otro.
//
// Patrones detectados:
//
//	REVERSA ALCISTA: Martillo, Envolvente Alcista, Morning Star,
//	                 Piercing Line, Three White Soldiers, Dragonfly Doji
//	REVERSA BAJISTA: Estrella Fugaz, Envolvente Bajista, Evening Star,
//	                 Dark Cloud Cover, Three Black Crows, Gravestone Doji
//	CONTINUACIÓN:    Three Methods (alcista/bajista), Marubozu
//	INDECISIÓN:      Doji, Spinning Top, Harami
//
// Multi-timeframe: Analiza patrones en múltiples timeframes (1day, 4h, 1h)
// y reporta la confluencia entre ellos para señales más fuertes.
//
// ============================================
package patterns

import (
	"fmt"
	"math"
	"strings"

	"bot-oscar/internal/models"
)

// ============================================
// Tipos y estructuras
// ============================================

// PatternType tipo de patrón de vela
type PatternType string

const (
	Bullish PatternType = "ALCISTA"
	Bearish PatternType = "BAJISTA"
	Neutral PatternType = "NEUTRAL"
)

// PatternStrength fuerza del patrón
type PatternStrength int

const (
	Weak   PatternStrength = 1
	Medium PatternStrength = 2
	Strong PatternStrength = 3
)

// DetectedPattern un patrón detectado en las velas
type DetectedPattern struct {
	Name      string          `json:"name"`      // Nombre del patrón
	NameEN    string          `json:"nameEN"`    // Nombre en inglés
	Type      PatternType     `json:"type"`      // ALCISTA, BAJISTA, NEUTRAL
	Strength  PatternStrength `json:"strength"`  // 1-3
	Timeframe string          `json:"timeframe"` // En qué timeframe se detectó
	Position  int             `json:"position"`  // Índice de la vela donde se detectó
	Details   string          `json:"details"`   // Descripción detallada
}

// PatternAnalysis resultado completo del análisis de patrones para UN activo
type PatternAnalysis struct {
	Symbol         string            `json:"symbol"`         // El activo analizado
	PatternsFound  []DetectedPattern `json:"patternsFound"`  // Todos los patrones encontrados
	BullishCount   int               `json:"bullishCount"`   // Cantidad alcistas
	BearishCount   int               `json:"bearishCount"`   // Cantidad bajistas
	NeutralCount   int               `json:"neutralCount"`   // Cantidad neutrales
	Bias           string            `json:"bias"`           // Sesgo general: ALCISTA/BAJISTA/NEUTRAL
	BiasStrength   int               `json:"biasStrength"`   // Fuerza del sesgo 0-100
	MultiTimeframe map[string]string `json:"multiTimeframe"` // Sesgo por timeframe
	Confluences    []string          `json:"confluences"`    // Confluencias entre timeframes
	SummaryForAI   string            `json:"summaryForAI"`   // Resumen formateado para DeepSeek
}

// ============================================
// Detector principal
// ============================================

// DetectPatterns analiza las velas de UN activo específico y detecta patrones
// Las velas DEBEN estar ordenadas cronológicamente (antiguo → reciente)
// IMPORTANTE: Solo analiza velas de ESE activo, nunca las mezcla con otro
func DetectPatterns(candles []models.Candle, timeframe string) []DetectedPattern {
	if len(candles) < 5 {
		return nil // Mínimo 5 velas para detectar patrones
	}

	var patterns []DetectedPattern

	// Analizar solo las últimas 50 velas (las más relevantes)
	start := len(candles) - 50
	if start < 0 {
		start = 0
	}
	subset := candles[start:]

	// Tamaño promedio del cuerpo para calibrar (adaptativo a cada activo)
	avgBody := calcAvgBody(subset)

	for i := 0; i < len(subset); i++ {
		// --- Patrones de 1 vela ---
		if p := detectDoji(subset, i, avgBody, timeframe); p != nil {
			patterns = append(patterns, *p)
		}
		if p := detectHammer(subset, i, avgBody, timeframe); p != nil {
			patterns = append(patterns, *p)
		}
		if p := detectShootingStar(subset, i, avgBody, timeframe); p != nil {
			patterns = append(patterns, *p)
		}
		if p := detectMarubozu(subset, i, avgBody, timeframe); p != nil {
			patterns = append(patterns, *p)
		}
		if p := detectSpinningTop(subset, i, avgBody, timeframe); p != nil {
			patterns = append(patterns, *p)
		}
		if p := detectDragonflyDoji(subset, i, avgBody, timeframe); p != nil {
			patterns = append(patterns, *p)
		}
		if p := detectGravestoneDoji(subset, i, avgBody, timeframe); p != nil {
			patterns = append(patterns, *p)
		}

		// --- Patrones de 2 velas ---
		if i >= 1 {
			if p := detectEngulfing(subset, i, avgBody, timeframe); p != nil {
				patterns = append(patterns, *p)
			}
			if p := detectHarami(subset, i, avgBody, timeframe); p != nil {
				patterns = append(patterns, *p)
			}
			if p := detectPiercingLine(subset, i, avgBody, timeframe); p != nil {
				patterns = append(patterns, *p)
			}
			if p := detectDarkCloudCover(subset, i, avgBody, timeframe); p != nil {
				patterns = append(patterns, *p)
			}
			if p := detectTweezerTop(subset, i, avgBody, timeframe); p != nil {
				patterns = append(patterns, *p)
			}
			if p := detectTweezerBottom(subset, i, avgBody, timeframe); p != nil {
				patterns = append(patterns, *p)
			}
		}

		// --- Patrones de 3 velas ---
		if i >= 2 {
			if p := detectMorningStar(subset, i, avgBody, timeframe); p != nil {
				patterns = append(patterns, *p)
			}
			if p := detectEveningStar(subset, i, avgBody, timeframe); p != nil {
				patterns = append(patterns, *p)
			}
			if p := detectThreeWhiteSoldiers(subset, i, avgBody, timeframe); p != nil {
				patterns = append(patterns, *p)
			}
			if p := detectThreeBlackCrows(subset, i, avgBody, timeframe); p != nil {
				patterns = append(patterns, *p)
			}
		}
	}

	return patterns
}

// AnalyzeMultiTimeframe analiza patrones en múltiples timeframes de UN SOLO activo
// candlesByTF es un mapa timeframe → velas de ESE activo
// NUNCA debe contener velas de otros activos
func AnalyzeMultiTimeframe(symbol string, candlesByTF map[string][]models.Candle) *PatternAnalysis {
	analysis := &PatternAnalysis{
		Symbol:         symbol,
		PatternsFound:  make([]DetectedPattern, 0),
		MultiTimeframe: make(map[string]string),
		Confluences:    make([]string, 0),
	}

	// Timeframes en orden de importancia (mayor primero)
	tfOrder := []string{"1day", "4h", "1h"}

	for _, tf := range tfOrder {
		candles, ok := candlesByTF[tf]
		if !ok || len(candles) < 5 {
			continue
		}

		patterns := DetectPatterns(candles, tf)

		// Solo considerar patrones de las últimas 5 velas (los recientes)
		// Position es relativo al subset (últimas 50 velas), no al array completo
		subsetLen := len(candles)
		if subsetLen > 50 {
			subsetLen = 50
		}
		var recentPatterns []DetectedPattern
		for _, p := range patterns {
			if p.Position >= subsetLen-5 {
				recentPatterns = append(recentPatterns, p)
			}
		}

		analysis.PatternsFound = append(analysis.PatternsFound, recentPatterns...)

		// Determinar sesgo por timeframe
		bullish, bearish := 0, 0
		for _, p := range recentPatterns {
			switch p.Type {
			case Bullish:
				bullish += int(p.Strength)
			case Bearish:
				bearish += int(p.Strength)
			}
		}

		if bullish > bearish+1 {
			analysis.MultiTimeframe[tf] = "ALCISTA"
		} else if bearish > bullish+1 {
			analysis.MultiTimeframe[tf] = "BAJISTA"
		} else {
			analysis.MultiTimeframe[tf] = "NEUTRAL"
		}
	}

	// Contar totales
	for _, p := range analysis.PatternsFound {
		switch p.Type {
		case Bullish:
			analysis.BullishCount++
		case Bearish:
			analysis.BearishCount++
		case Neutral:
			analysis.NeutralCount++
		}
	}

	// Determinar sesgo general con pesos por timeframe
	totalBull, totalBear := 0, 0
	tfWeights := map[string]int{"1day": 3, "4h": 2, "1h": 1}
	for tf, bias := range analysis.MultiTimeframe {
		w := tfWeights[tf]
		if w == 0 {
			w = 1
		}
		switch bias {
		case "ALCISTA":
			totalBull += w
		case "BAJISTA":
			totalBear += w
		}
	}

	total := totalBull + totalBear
	if total == 0 {
		analysis.Bias = "NEUTRAL"
		analysis.BiasStrength = 50
	} else if totalBull > totalBear {
		analysis.Bias = "ALCISTA"
		analysis.BiasStrength = 50 + (totalBull*50)/(totalBull+totalBear)
	} else if totalBear > totalBull {
		analysis.Bias = "BAJISTA"
		analysis.BiasStrength = 50 + (totalBear*50)/(totalBull+totalBear)
	} else {
		analysis.Bias = "NEUTRAL"
		analysis.BiasStrength = 50
	}

	// Detectar confluencias multi-timeframe
	analysis.detectConfluences()

	// Generar resumen para DeepSeek
	analysis.SummaryForAI = analysis.buildSummary()

	return analysis
}

// detectConfluences encuentra alineación entre timeframes
func (a *PatternAnalysis) detectConfluences() {
	tfs := []string{"1day", "4h", "1h"}
	for i := 0; i < len(tfs)-1; i++ {
		for j := i + 1; j < len(tfs); j++ {
			b1, ok1 := a.MultiTimeframe[tfs[i]]
			b2, ok2 := a.MultiTimeframe[tfs[j]]
			if ok1 && ok2 && b1 == b2 && b1 != "NEUTRAL" {
				a.Confluences = append(a.Confluences,
					fmt.Sprintf("Confluencia %s entre %s y %s", b1, tfs[i], tfs[j]))
			}
		}
	}

	// Triple confluencia
	if len(a.MultiTimeframe) == 3 {
		b1, _ := a.MultiTimeframe["1day"]
		b2, _ := a.MultiTimeframe["4h"]
		b3, _ := a.MultiTimeframe["1h"]
		if b1 == b2 && b2 == b3 && b1 != "NEUTRAL" {
			a.Confluences = append(a.Confluences,
				fmt.Sprintf("⚡ TRIPLE CONFLUENCIA %s (1day + 4h + 1h) — señal MUY fuerte", b1))
			a.BiasStrength = min(a.BiasStrength+15, 100)
		}
	}
}

// buildSummary genera el texto formateado para incluir en el prompt de DeepSeek
func (a *PatternAnalysis) buildSummary() string {
	if len(a.PatternsFound) == 0 {
		return fmt.Sprintf("No se detectaron patrones de velas significativos en %s.", a.Symbol)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("PATRONES DE VELAS DETECTADOS EN %s (exclusivos de este activo):\n", a.Symbol))

	// Agrupar por timeframe
	byTF := make(map[string][]DetectedPattern)
	for _, p := range a.PatternsFound {
		byTF[p.Timeframe] = append(byTF[p.Timeframe], p)
	}

	for _, tf := range []string{"1day", "4h", "1h"} {
		pats, ok := byTF[tf]
		if !ok || len(pats) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("\n  [%s] ", tf))
		for i, p := range pats {
			stars := strings.Repeat("★", int(p.Strength))
			if i > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(fmt.Sprintf("%s (%s) %s", p.Name, p.Type, stars))
		}
	}

	// Sesgo y confluencias
	sb.WriteString(fmt.Sprintf("\n\nSESGO GENERAL DE PATRONES: %s (fuerza: %d/100)", a.Bias, a.BiasStrength))

	if len(a.Confluences) > 0 {
		sb.WriteString("\nCONFLUENCIAS MULTI-TIMEFRAME:")
		for _, c := range a.Confluences {
			sb.WriteString(fmt.Sprintf("\n  → %s", c))
		}
	}

	// Resumen por tipo
	sb.WriteString(fmt.Sprintf("\n\nResumen: %d alcistas, %d bajistas, %d neutrales",
		a.BullishCount, a.BearishCount, a.NeutralCount))

	return sb.String()
}

// ============================================
// Helpers de cálculo
// ============================================

// calcAvgBody calcula el cuerpo promedio de las velas (adaptativo al activo)
func calcAvgBody(candles []models.Candle) float64 {
	if len(candles) == 0 {
		return 0
	}
	total := 0.0
	for _, c := range candles {
		total += math.Abs(c.Close - c.Open)
	}
	return total / float64(len(candles))
}

// body tamaño del cuerpo de una vela
func body(c models.Candle) float64 {
	return math.Abs(c.Close - c.Open)
}

// upperShadow sombra superior
func upperShadow(c models.Candle) float64 {
	return c.High - math.Max(c.Open, c.Close)
}

// lowerShadow sombra inferior
func lowerShadow(c models.Candle) float64 {
	return math.Min(c.Open, c.Close) - c.Low
}

// isBullish vela alcista (cierre > apertura)
func isBullish(c models.Candle) bool {
	return c.Close > c.Open
}

// isBearish vela bajista (cierre < apertura)
func isBearish(c models.Candle) bool {
	return c.Close < c.Open
}

// range_ rango total de la vela (high - low)
func range_(c models.Candle) float64 {
	return c.High - c.Low
}

// min retorna el menor de dos enteros
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================
// PATRONES DE 1 VELA
// ============================================

// detectDoji — Cuerpo muy pequeño, indica indecisión
func detectDoji(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	c := candles[i]
	if avgBody == 0 || range_(c) == 0 {
		return nil
	}
	// Cuerpo < 10% del rango total y < 25% del cuerpo promedio
	if body(c) < range_(c)*0.10 && body(c) < avgBody*0.25 {
		// No es Dragonfly ni Gravestone (esos se detectan aparte)
		us := upperShadow(c)
		ls := lowerShadow(c)
		if us > 0 && ls > 0 && us/ls < 3 && ls/us < 3 {
			return &DetectedPattern{
				Name:      "Doji",
				NameEN:    "Doji",
				Type:      Neutral,
				Strength:  Weak,
				Timeframe: tf,
				Position:  i,
				Details:   "Indecisión total del mercado. Posible cambio de dirección.",
			}
		}
	}
	return nil
}

// detectDragonflyDoji — Doji con sombra inferior larga, sin sombra superior → alcista
func detectDragonflyDoji(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	c := candles[i]
	if avgBody == 0 || range_(c) == 0 {
		return nil
	}
	if body(c) < range_(c)*0.10 && lowerShadow(c) > range_(c)*0.60 && upperShadow(c) < range_(c)*0.10 {
		return &DetectedPattern{
			Name:      "Doji Libélula",
			NameEN:    "Dragonfly Doji",
			Type:      Bullish,
			Strength:  Medium,
			Timeframe: tf,
			Position:  i,
			Details:   "Rechazo fuerte de precios bajos. Señal alcista en soporte.",
		}
	}
	return nil
}

// detectGravestoneDoji — Doji con sombra superior larga, sin sombra inferior → bajista
func detectGravestoneDoji(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	c := candles[i]
	if avgBody == 0 || range_(c) == 0 {
		return nil
	}
	if body(c) < range_(c)*0.10 && upperShadow(c) > range_(c)*0.60 && lowerShadow(c) < range_(c)*0.10 {
		return &DetectedPattern{
			Name:      "Doji Lápida",
			NameEN:    "Gravestone Doji",
			Type:      Bearish,
			Strength:  Medium,
			Timeframe: tf,
			Position:  i,
			Details:   "Rechazo fuerte de precios altos. Señal bajista en resistencia.",
		}
	}
	return nil
}

// detectHammer — Cuerpo pequeño arriba, sombra inferior larga → alcista en tendencia bajista
func detectHammer(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	c := candles[i]
	if avgBody == 0 || range_(c) == 0 {
		return nil
	}
	b := body(c)
	ls := lowerShadow(c)
	us := upperShadow(c)

	// Sombra inferior >= 2x el cuerpo, sombra superior < 30% del cuerpo
	if b > avgBody*0.3 && ls >= b*2 && us < b*0.5 {
		// Verificar que hay tendencia bajista previa (3 velas bajistas antes)
		if i >= 3 {
			bajistas := 0
			for j := i - 3; j < i; j++ {
				if isBearish(candles[j]) {
					bajistas++
				}
			}
			if bajistas >= 2 {
				return &DetectedPattern{
					Name:      "Martillo",
					NameEN:    "Hammer",
					Type:      Bullish,
					Strength:  Strong,
					Timeframe: tf,
					Position:  i,
					Details:   "Martillo tras tendencia bajista. Los compradores rechazaron los mínimos con fuerza.",
				}
			}
		}
	}
	return nil
}

// detectShootingStar — Cuerpo pequeño abajo, sombra superior larga → bajista en tendencia alcista
func detectShootingStar(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	c := candles[i]
	if avgBody == 0 || range_(c) == 0 {
		return nil
	}
	b := body(c)
	us := upperShadow(c)
	ls := lowerShadow(c)

	// Sombra superior >= 2x el cuerpo, sombra inferior < 30% del cuerpo
	if b > avgBody*0.3 && us >= b*2 && ls < b*0.5 {
		// Verificar tendencia alcista previa
		if i >= 3 {
			alcistas := 0
			for j := i - 3; j < i; j++ {
				if isBullish(candles[j]) {
					alcistas++
				}
			}
			if alcistas >= 2 {
				return &DetectedPattern{
					Name:      "Estrella Fugaz",
					NameEN:    "Shooting Star",
					Type:      Bearish,
					Strength:  Strong,
					Timeframe: tf,
					Position:  i,
					Details:   "Estrella fugaz tras tendencia alcista. Los vendedores rechazaron los máximos con fuerza.",
				}
			}
		}
	}
	return nil
}

// detectMarubozu — Vela con cuerpo grande y casi sin sombras → momentum fuerte
func detectMarubozu(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	c := candles[i]
	r := range_(c)
	if r == 0 {
		return nil
	}
	// Cuerpo > 90% del rango y > 1.5x el cuerpo promedio
	if body(c) > r*0.90 && body(c) > avgBody*1.5 {
		patternType := Bullish
		name := "Marubozu Alcista"
		details := "Cuerpo lleno alcista sin sombras. Presión compradora extrema."
		if isBearish(c) {
			patternType = Bearish
			name = "Marubozu Bajista"
			details = "Cuerpo lleno bajista sin sombras. Presión vendedora extrema."
		}
		return &DetectedPattern{
			Name:      name,
			NameEN:    "Marubozu",
			Type:      patternType,
			Strength:  Strong,
			Timeframe: tf,
			Position:  i,
			Details:   details,
		}
	}
	return nil
}

// detectSpinningTop — Cuerpo pequeño con sombras largas simétricas
func detectSpinningTop(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	c := candles[i]
	r := range_(c)
	if r == 0 || avgBody == 0 {
		return nil
	}
	b := body(c)
	us := upperShadow(c)
	ls := lowerShadow(c)

	// Cuerpo pequeño (< 40% del rango), ambas sombras > cuerpo
	if b < r*0.40 && b < avgBody*0.8 && us > b && ls > b {
		return &DetectedPattern{
			Name:      "Trompo",
			NameEN:    "Spinning Top",
			Type:      Neutral,
			Strength:  Weak,
			Timeframe: tf,
			Position:  i,
			Details:   "Indecisión con lucha activa entre compradores y vendedores.",
		}
	}
	return nil
}

// ============================================
// PATRONES DE 2 VELAS
// ============================================

// detectEngulfing — Envolvente: la segunda vela envuelve completamente el cuerpo de la primera
func detectEngulfing(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	if i < 1 {
		return nil
	}
	prev := candles[i-1]
	curr := candles[i]

	prevBody := body(prev)
	currBody := body(curr)

	// La segunda vela debe tener cuerpo significativo y mayor que la primera
	if currBody < avgBody*0.8 || currBody < prevBody*1.1 {
		return nil
	}

	// Envolvente Alcista: prev bajista, curr alcista, curr envuelve prev
	if isBearish(prev) && isBullish(curr) {
		if curr.Open <= prev.Close && curr.Close >= prev.Open {
			return &DetectedPattern{
				Name:      "Envolvente Alcista",
				NameEN:    "Bullish Engulfing",
				Type:      Bullish,
				Strength:  Strong,
				Timeframe: tf,
				Position:  i,
				Details:   "Vela alcista envuelve completamente la bajista anterior. Fuerte señal de reversión alcista.",
			}
		}
	}

	// Envolvente Bajista: prev alcista, curr bajista, curr envuelve prev
	if isBullish(prev) && isBearish(curr) {
		if curr.Open >= prev.Close && curr.Close <= prev.Open {
			return &DetectedPattern{
				Name:      "Envolvente Bajista",
				NameEN:    "Bearish Engulfing",
				Type:      Bearish,
				Strength:  Strong,
				Timeframe: tf,
				Position:  i,
				Details:   "Vela bajista envuelve completamente la alcista anterior. Fuerte señal de reversión bajista.",
			}
		}
	}

	return nil
}

// detectHarami — La segunda vela está contenida dentro del cuerpo de la primera
func detectHarami(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	if i < 1 {
		return nil
	}
	prev := candles[i-1]
	curr := candles[i]

	// Primera vela de cuerpo grande, segunda de cuerpo pequeño dentro
	if body(prev) < avgBody*0.8 || body(curr) > body(prev)*0.6 {
		return nil
	}

	prevHigh := math.Max(prev.Open, prev.Close)
	prevLow := math.Min(prev.Open, prev.Close)
	currHigh := math.Max(curr.Open, curr.Close)
	currLow := math.Min(curr.Open, curr.Close)

	if currHigh <= prevHigh && currLow >= prevLow {
		patternType := Neutral
		name := "Harami"
		details := "Vela pequeña contenida en la anterior. Posible pausa o reversión."

		if isBearish(prev) && isBullish(curr) {
			patternType = Bullish
			name = "Harami Alcista"
			details = "Vela alcista pequeña dentro de bajista grande. Posible reversión alcista."
		} else if isBullish(prev) && isBearish(curr) {
			patternType = Bearish
			name = "Harami Bajista"
			details = "Vela bajista pequeña dentro de alcista grande. Posible reversión bajista."
		}

		return &DetectedPattern{
			Name:      name,
			NameEN:    "Harami",
			Type:      patternType,
			Strength:  Medium,
			Timeframe: tf,
			Position:  i,
			Details:   details,
		}
	}
	return nil
}

// detectPiercingLine — Bajista seguida de alcista que cierra > 50% del cuerpo de la primera
func detectPiercingLine(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	if i < 1 {
		return nil
	}
	prev := candles[i-1]
	curr := candles[i]

	if !isBearish(prev) || !isBullish(curr) {
		return nil
	}
	if body(prev) < avgBody*0.8 || body(curr) < avgBody*0.8 {
		return nil
	}

	// Abre por debajo del cierre anterior, cierra por encima del 50% del cuerpo de la anterior
	midPrev := (prev.Open + prev.Close) / 2
	if curr.Open < prev.Close && curr.Close > midPrev && curr.Close < prev.Open {
		return &DetectedPattern{
			Name:      "Línea Penetrante",
			NameEN:    "Piercing Line",
			Type:      Bullish,
			Strength:  Medium,
			Timeframe: tf,
			Position:  i,
			Details:   "La alcista penetra más del 50% de la bajista anterior. Señal de reversión alcista.",
		}
	}
	return nil
}

// detectDarkCloudCover — Alcista seguida de bajista que cierra < 50% del cuerpo de la primera
func detectDarkCloudCover(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	if i < 1 {
		return nil
	}
	prev := candles[i-1]
	curr := candles[i]

	if !isBullish(prev) || !isBearish(curr) {
		return nil
	}
	if body(prev) < avgBody*0.8 || body(curr) < avgBody*0.8 {
		return nil
	}

	midPrev := (prev.Open + prev.Close) / 2
	if curr.Open > prev.Close && curr.Close < midPrev && curr.Close > prev.Open {
		return &DetectedPattern{
			Name:      "Nube Oscura",
			NameEN:    "Dark Cloud Cover",
			Type:      Bearish,
			Strength:  Medium,
			Timeframe: tf,
			Position:  i,
			Details:   "La bajista cierra por debajo del 50% de la alcista anterior. Señal de reversión bajista.",
		}
	}
	return nil
}

// detectTweezerTop — Dos velas con máximos casi iguales en techo
func detectTweezerTop(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	if i < 1 {
		return nil
	}
	prev := candles[i-1]
	curr := candles[i]

	tolerance := avgBody * 0.1
	if tolerance == 0 {
		return nil
	}

	if isBullish(prev) && isBearish(curr) && math.Abs(prev.High-curr.High) < tolerance {
		return &DetectedPattern{
			Name:      "Pinza Superior",
			NameEN:    "Tweezer Top",
			Type:      Bearish,
			Strength:  Medium,
			Timeframe: tf,
			Position:  i,
			Details:   "Dos velas rechazan el mismo máximo. Fuerte resistencia, posible caída.",
		}
	}
	return nil
}

// detectTweezerBottom — Dos velas con mínimos casi iguales en suelo
func detectTweezerBottom(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	if i < 1 {
		return nil
	}
	prev := candles[i-1]
	curr := candles[i]

	tolerance := avgBody * 0.1
	if tolerance == 0 {
		return nil
	}

	if isBearish(prev) && isBullish(curr) && math.Abs(prev.Low-curr.Low) < tolerance {
		return &DetectedPattern{
			Name:      "Pinza Inferior",
			NameEN:    "Tweezer Bottom",
			Type:      Bullish,
			Strength:  Medium,
			Timeframe: tf,
			Position:  i,
			Details:   "Dos velas rechazan el mismo mínimo. Fuerte soporte, posible rebote.",
		}
	}
	return nil
}

// ============================================
// PATRONES DE 3 VELAS
// ============================================

// detectMorningStar — Bajista + pequeña + alcista (gap down, reversión)
func detectMorningStar(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	if i < 2 {
		return nil
	}
	first := candles[i-2]
	second := candles[i-1]
	third := candles[i]

	if !isBearish(first) || !isBullish(third) {
		return nil
	}
	if body(first) < avgBody*0.8 || body(third) < avgBody*0.8 {
		return nil
	}
	// Segunda vela: cuerpo pequeño (< 50% del promedio)
	if body(second) > avgBody*0.5 {
		return nil
	}
	// Tercera cierra al menos al 50% del cuerpo de la primera
	midFirst := (first.Open + first.Close) / 2
	if third.Close > midFirst {
		return &DetectedPattern{
			Name:      "Estrella de la Mañana",
			NameEN:    "Morning Star",
			Type:      Bullish,
			Strength:  Strong,
			Timeframe: tf,
			Position:  i,
			Details:   "Patrón de 3 velas: bajista + indecisión + alcista fuerte. Reversión alcista confirmada.",
		}
	}
	return nil
}

// detectEveningStar — Alcista + pequeña + bajista (gap up, reversión)
func detectEveningStar(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	if i < 2 {
		return nil
	}
	first := candles[i-2]
	second := candles[i-1]
	third := candles[i]

	if !isBullish(first) || !isBearish(third) {
		return nil
	}
	if body(first) < avgBody*0.8 || body(third) < avgBody*0.8 {
		return nil
	}
	if body(second) > avgBody*0.5 {
		return nil
	}
	midFirst := (first.Open + first.Close) / 2
	if third.Close < midFirst {
		return &DetectedPattern{
			Name:      "Estrella Vespertina",
			NameEN:    "Evening Star",
			Type:      Bearish,
			Strength:  Strong,
			Timeframe: tf,
			Position:  i,
			Details:   "Patrón de 3 velas: alcista + indecisión + bajista fuerte. Reversión bajista confirmada.",
		}
	}
	return nil
}

// detectThreeWhiteSoldiers — Tres velas alcistas consecutivas con cuerpos grandes
func detectThreeWhiteSoldiers(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	if i < 2 {
		return nil
	}
	c1 := candles[i-2]
	c2 := candles[i-1]
	c3 := candles[i]

	// Las 3 deben ser alcistas con cuerpos significativos
	if !isBullish(c1) || !isBullish(c2) || !isBullish(c3) {
		return nil
	}
	if body(c1) < avgBody*0.6 || body(c2) < avgBody*0.6 || body(c3) < avgBody*0.6 {
		return nil
	}
	// Cada una abre dentro del cuerpo de la anterior y cierra más alto
	if c2.Open >= c1.Open && c2.Open <= c1.Close && c2.Close > c1.Close &&
		c3.Open >= c2.Open && c3.Open <= c2.Close && c3.Close > c2.Close {
		return &DetectedPattern{
			Name:      "Tres Soldados Blancos",
			NameEN:    "Three White Soldiers",
			Type:      Bullish,
			Strength:  Strong,
			Timeframe: tf,
			Position:  i,
			Details:   "Tres velas alcistas consecutivas con cierres progresivos. Momentum alcista fuerte.",
		}
	}
	return nil
}

// detectThreeBlackCrows — Tres velas bajistas consecutivas con cuerpos grandes
func detectThreeBlackCrows(candles []models.Candle, i int, avgBody float64, tf string) *DetectedPattern {
	if i < 2 {
		return nil
	}
	c1 := candles[i-2]
	c2 := candles[i-1]
	c3 := candles[i]

	if !isBearish(c1) || !isBearish(c2) || !isBearish(c3) {
		return nil
	}
	if body(c1) < avgBody*0.6 || body(c2) < avgBody*0.6 || body(c3) < avgBody*0.6 {
		return nil
	}
	if c2.Open <= c1.Open && c2.Open >= c1.Close && c2.Close < c1.Close &&
		c3.Open <= c2.Open && c3.Open >= c2.Close && c3.Close < c2.Close {
		return &DetectedPattern{
			Name:      "Tres Cuervos Negros",
			NameEN:    "Three Black Crows",
			Type:      Bearish,
			Strength:  Strong,
			Timeframe: tf,
			Position:  i,
			Details:   "Tres velas bajistas consecutivas con cierres progresivos. Momentum bajista fuerte.",
		}
	}
	return nil
}
