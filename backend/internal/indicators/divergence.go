// ============================================
// Bot Oscar - Detector de Divergencias RSI/MACD vs Precio
//
// Las divergencias son las señales ANTICIPATORIAS más potentes
// en análisis técnico. Detectan agotamiento de tendencia ANTES
// de que el patrón de velas se forme.
//
// Tipos de divergencia:
//
//	DIVERGENCIA ALCISTA (Bullish):
//	- Precio hace mínimo MÁS BAJO → pero RSI/MACD hace mínimo MÁS ALTO
//	- Señal: la presión vendedora se agota → probable reversal al alza
//	- Aparece en fondos de mercado
//
//	DIVERGENCIA BAJISTA (Bearish):
//	- Precio hace máximo MÁS ALTO → pero RSI/MACD hace máximo MÁS BAJO
//	- Señal: la presión compradora se agota → probable reversal a la baja
//	- Aparece en techos de mercado
//
//	DIVERGENCIA OCULTA ALCISTA (Hidden Bullish):
//	- Precio hace mínimo MÁS ALTO → pero RSI/MACD hace mínimo MÁS BAJO
//	- Señal: continuación de tendencia alcista (pullback sano)
//
//	DIVERGENCIA OCULTA BAJISTA (Hidden Bearish):
//	- Precio hace máximo MÁS BAJO → pero RSI/MACD hace máximo MÁS ALTO
//	- Señal: continuación de tendencia bajista
//
// Ventana de detección: últimas 5-30 velas (configurable)
// ============================================
package indicators

import "fmt"

// DivergenceType tipo de divergencia detectada
type DivergenceType string

const (
	BullishDivergence       DivergenceType = "ALCISTA"
	BearishDivergence       DivergenceType = "BAJISTA"
	HiddenBullishDivergence DivergenceType = "OCULTA_ALCISTA"
	HiddenBearishDivergence DivergenceType = "OCULTA_BAJISTA"
)

// Divergence representa una divergencia detectada entre precio e indicador
type Divergence struct {
	Type       DivergenceType `json:"type"`       // Tipo de divergencia
	Indicator  string         `json:"indicator"`  // "RSI" o "MACD"
	Strength   int            `json:"strength"`   // 1=débil, 2=moderada, 3=fuerte
	Signal     string         `json:"signal"`     // "COMPRA" o "VENTA" (anticipada)
	Details    string         `json:"details"`    // Explicación legible
	PricePeak1 float64        `json:"pricePeak1"` // Primer extremo del precio
	PricePeak2 float64        `json:"pricePeak2"` // Segundo extremo del precio
	IndPeak1   float64        `json:"indPeak1"`   // Primer extremo del indicador
	IndPeak2   float64        `json:"indPeak2"`   // Segundo extremo del indicador
	BarsAgo    int            `json:"barsAgo"`    // Hace cuántas velas se detectó
}

// DivergenceResult resultado completo del análisis de divergencias
type DivergenceResult struct {
	Divergences     []Divergence `json:"divergences"`     // Divergencias encontradas
	HasBullish      bool         `json:"hasBullish"`      // ¿Hay alguna alcista?
	HasBearish      bool         `json:"hasBearish"`      // ¿Hay alguna bajista?
	StrongestSignal string       `json:"strongestSignal"` // "COMPRA", "VENTA" o "NINGUNA"
	MaxStrength     int          `json:"maxStrength"`     // Fuerza máxima encontrada
	SummaryForAI    string       `json:"summaryForAI"`    // Resumen para DeepSeek
}

// DetectDivergences analiza precios, RSI y MACD para encontrar divergencias
// closes: precios de cierre (mínimo 50)
// rsiValues: serie RSI ya calculada
// macdHist: serie de histograma MACD ya calculada
// lookback: ventana de búsqueda (recomendado: 20-30)
func DetectDivergences(closes []float64, rsiValues []float64, macdHist []float64, lookback int) *DivergenceResult {
	result := &DivergenceResult{
		Divergences:     make([]Divergence, 0),
		StrongestSignal: "NINGUNA",
	}

	if lookback < 10 {
		lookback = 20
	}

	// ============================================
	// 1. Detectar divergencias RSI vs Precio
	// ============================================
	if len(rsiValues) >= lookback && len(closes) >= lookback {
		// Alinear: tomar los últimos N de ambos
		priceWindow := closes[len(closes)-lookback:]
		rsiWindow := rsiValues[len(rsiValues)-lookback:]

		// Buscar mínimos locales (para divergencia alcista)
		priceMins := findLocalMinima(priceWindow, 3)
		rsiMins := findLocalMinima(rsiWindow, 3)

		if len(priceMins) >= 2 && len(rsiMins) >= 2 {
			pm1, pm2 := priceMins[len(priceMins)-2], priceMins[len(priceMins)-1]
			rm1, rm2 := rsiMins[len(rsiMins)-2], rsiMins[len(rsiMins)-1]

			// Divergencia ALCISTA regular: precio baja, RSI sube
			if priceWindow[pm2.idx] < priceWindow[pm1.idx] && rsiWindow[rm2.idx] > rsiWindow[rm1.idx] {
				strength := calcDivStrength(
					priceWindow[pm1.idx], priceWindow[pm2.idx],
					rsiWindow[rm1.idx], rsiWindow[rm2.idx],
				)
				result.Divergences = append(result.Divergences, Divergence{
					Type:       BullishDivergence,
					Indicator:  "RSI",
					Strength:   strength,
					Signal:     "COMPRA",
					PricePeak1: priceWindow[pm1.idx],
					PricePeak2: priceWindow[pm2.idx],
					IndPeak1:   rsiWindow[rm1.idx],
					IndPeak2:   rsiWindow[rm2.idx],
					BarsAgo:    lookback - pm2.idx,
					Details: fmt.Sprintf("Divergencia ALCISTA RSI: precio hizo mínimo más bajo ($%.2f→$%.2f) pero RSI subió (%.1f→%.1f) → agotamiento vendedor, probable rebote",
						priceWindow[pm1.idx], priceWindow[pm2.idx], rsiWindow[rm1.idx], rsiWindow[rm2.idx]),
				})
			}

			// Divergencia OCULTA ALCISTA: precio sube (mínimo más alto), RSI baja
			if priceWindow[pm2.idx] > priceWindow[pm1.idx] && rsiWindow[rm2.idx] < rsiWindow[rm1.idx] {
				strength := calcDivStrength(
					priceWindow[pm1.idx], priceWindow[pm2.idx],
					rsiWindow[rm1.idx], rsiWindow[rm2.idx],
				)
				if strength >= 2 { // Solo reportar ocultas si son moderadas+
					result.Divergences = append(result.Divergences, Divergence{
						Type:       HiddenBullishDivergence,
						Indicator:  "RSI",
						Strength:   strength,
						Signal:     "COMPRA",
						PricePeak1: priceWindow[pm1.idx],
						PricePeak2: priceWindow[pm2.idx],
						IndPeak1:   rsiWindow[rm1.idx],
						IndPeak2:   rsiWindow[rm2.idx],
						BarsAgo:    lookback - pm2.idx,
						Details: fmt.Sprintf("Divergencia OCULTA ALCISTA RSI: tendencia alcista con pullback sano (mínimo $%.2f→$%.2f, RSI %.1f→%.1f) → continuación alcista",
							priceWindow[pm1.idx], priceWindow[pm2.idx], rsiWindow[rm1.idx], rsiWindow[rm2.idx]),
					})
				}
			}
		}

		// Buscar máximos locales (para divergencia bajista)
		priceMaxs := findLocalMaxima(priceWindow, 3)
		rsiMaxs := findLocalMaxima(rsiWindow, 3)

		if len(priceMaxs) >= 2 && len(rsiMaxs) >= 2 {
			pm1, pm2 := priceMaxs[len(priceMaxs)-2], priceMaxs[len(priceMaxs)-1]
			rm1, rm2 := rsiMaxs[len(rsiMaxs)-2], rsiMaxs[len(rsiMaxs)-1]

			// Divergencia BAJISTA regular: precio sube, RSI baja
			if priceWindow[pm2.idx] > priceWindow[pm1.idx] && rsiWindow[rm2.idx] < rsiWindow[rm1.idx] {
				strength := calcDivStrength(
					priceWindow[pm1.idx], priceWindow[pm2.idx],
					rsiWindow[rm1.idx], rsiWindow[rm2.idx],
				)
				result.Divergences = append(result.Divergences, Divergence{
					Type:       BearishDivergence,
					Indicator:  "RSI",
					Strength:   strength,
					Signal:     "VENTA",
					PricePeak1: priceWindow[pm1.idx],
					PricePeak2: priceWindow[pm2.idx],
					IndPeak1:   rsiWindow[rm1.idx],
					IndPeak2:   rsiWindow[rm2.idx],
					BarsAgo:    lookback - pm2.idx,
					Details: fmt.Sprintf("Divergencia BAJISTA RSI: precio hizo máximo más alto ($%.2f→$%.2f) pero RSI bajó (%.1f→%.1f) → agotamiento comprador, probable caída",
						priceWindow[pm1.idx], priceWindow[pm2.idx], rsiWindow[rm1.idx], rsiWindow[rm2.idx]),
				})
			}

			// Divergencia OCULTA BAJISTA: precio baja (máximo más bajo), RSI sube
			if priceWindow[pm2.idx] < priceWindow[pm1.idx] && rsiWindow[rm2.idx] > rsiWindow[rm1.idx] {
				strength := calcDivStrength(
					priceWindow[pm1.idx], priceWindow[pm2.idx],
					rsiWindow[rm1.idx], rsiWindow[rm2.idx],
				)
				if strength >= 2 {
					result.Divergences = append(result.Divergences, Divergence{
						Type:       HiddenBearishDivergence,
						Indicator:  "RSI",
						Strength:   strength,
						Signal:     "VENTA",
						PricePeak1: priceWindow[pm1.idx],
						PricePeak2: priceWindow[pm2.idx],
						IndPeak1:   rsiWindow[rm1.idx],
						IndPeak2:   rsiWindow[rm2.idx],
						BarsAgo:    lookback - pm2.idx,
						Details: fmt.Sprintf("Divergencia OCULTA BAJISTA RSI: tendencia bajista con rally débil (máximo $%.2f→$%.2f, RSI %.1f→%.1f) → continuación bajista",
							priceWindow[pm1.idx], priceWindow[pm2.idx], rsiWindow[rm1.idx], rsiWindow[rm2.idx]),
					})
				}
			}
		}
	}

	// ============================================
	// 2. Detectar divergencias MACD Histograma vs Precio
	// ============================================
	if len(macdHist) >= lookback && len(closes) >= lookback {
		priceWindow := closes[len(closes)-lookback:]
		macdWindow := macdHist[len(macdHist)-lookback:]

		// Mínimos (divergencia alcista MACD)
		priceMins := findLocalMinima(priceWindow, 3)
		macdMins := findLocalMinima(macdWindow, 3)

		if len(priceMins) >= 2 && len(macdMins) >= 2 {
			pm1, pm2 := priceMins[len(priceMins)-2], priceMins[len(priceMins)-1]
			mm1, mm2 := macdMins[len(macdMins)-2], macdMins[len(macdMins)-1]

			// Divergencia ALCISTA MACD: precio baja, histograma MACD sube
			if priceWindow[pm2.idx] < priceWindow[pm1.idx] && macdWindow[mm2.idx] > macdWindow[mm1.idx] {
				strength := calcDivStrength(
					priceWindow[pm1.idx], priceWindow[pm2.idx],
					macdWindow[mm1.idx], macdWindow[mm2.idx],
				)
				result.Divergences = append(result.Divergences, Divergence{
					Type:       BullishDivergence,
					Indicator:  "MACD",
					Strength:   strength,
					Signal:     "COMPRA",
					PricePeak1: priceWindow[pm1.idx],
					PricePeak2: priceWindow[pm2.idx],
					IndPeak1:   macdWindow[mm1.idx],
					IndPeak2:   macdWindow[mm2.idx],
					BarsAgo:    lookback - pm2.idx,
					Details: fmt.Sprintf("Divergencia ALCISTA MACD: precio bajó ($%.2f→$%.2f) pero momentum MACD subió → pérdida de impulso bajista",
						priceWindow[pm1.idx], priceWindow[pm2.idx]),
				})
			}
		}

		// Máximos (divergencia bajista MACD)
		priceMaxs := findLocalMaxima(priceWindow, 3)
		macdMaxs := findLocalMaxima(macdWindow, 3)

		if len(priceMaxs) >= 2 && len(macdMaxs) >= 2 {
			pm1, pm2 := priceMaxs[len(priceMaxs)-2], priceMaxs[len(priceMaxs)-1]
			mm1, mm2 := macdMaxs[len(macdMaxs)-2], macdMaxs[len(macdMaxs)-1]

			// Divergencia BAJISTA MACD: precio sube, histograma MACD baja
			if priceWindow[pm2.idx] > priceWindow[pm1.idx] && macdWindow[mm2.idx] < macdWindow[mm1.idx] {
				strength := calcDivStrength(
					priceWindow[pm1.idx], priceWindow[pm2.idx],
					macdWindow[mm1.idx], macdWindow[mm2.idx],
				)
				result.Divergences = append(result.Divergences, Divergence{
					Type:       BearishDivergence,
					Indicator:  "MACD",
					Strength:   strength,
					Signal:     "VENTA",
					PricePeak1: priceWindow[pm1.idx],
					PricePeak2: priceWindow[pm2.idx],
					IndPeak1:   macdWindow[mm1.idx],
					IndPeak2:   macdWindow[mm2.idx],
					BarsAgo:    lookback - pm2.idx,
					Details: fmt.Sprintf("Divergencia BAJISTA MACD: precio subió ($%.2f→$%.2f) pero momentum MACD bajó → pérdida de impulso alcista",
						priceWindow[pm1.idx], priceWindow[pm2.idx]),
				})
			}
		}
	}

	// ============================================
	// 3. Calcular resumen y señal más fuerte
	// ============================================
	maxBullStr := 0
	maxBearStr := 0

	for _, d := range result.Divergences {
		if d.Signal == "COMPRA" {
			result.HasBullish = true
			if d.Strength > maxBullStr {
				maxBullStr = d.Strength
			}
		} else {
			result.HasBearish = true
			if d.Strength > maxBearStr {
				maxBearStr = d.Strength
			}
		}
	}

	if maxBullStr > maxBearStr {
		result.StrongestSignal = "COMPRA"
		result.MaxStrength = maxBullStr
	} else if maxBearStr > maxBullStr {
		result.StrongestSignal = "VENTA"
		result.MaxStrength = maxBearStr
	}

	// Generar resumen para DeepSeek
	result.SummaryForAI = buildDivergenceSummary(result)

	return result
}

// ============================================
// Helpers internos
// ============================================

// peak representa un extremo local (mínimo o máximo)
type peak struct {
	idx int
	val float64
}

// findLocalMinima encuentra mínimos locales en una serie de datos
// order: cuántas velas a cada lado deben ser mayores para confirmar el mínimo
func findLocalMinima(data []float64, order int) []peak {
	peaks := make([]peak, 0)
	if len(data) < 2*order+1 {
		return peaks
	}

	for i := order; i < len(data)-order; i++ {
		isMin := true
		for j := 1; j <= order; j++ {
			if data[i] >= data[i-j] || data[i] >= data[i+j] {
				isMin = false
				break
			}
		}
		if isMin {
			peaks = append(peaks, peak{idx: i, val: data[i]})
		}
	}
	return peaks
}

// findLocalMaxima encuentra máximos locales en una serie de datos
func findLocalMaxima(data []float64, order int) []peak {
	peaks := make([]peak, 0)
	if len(data) < 2*order+1 {
		return peaks
	}

	for i := order; i < len(data)-order; i++ {
		isMax := true
		for j := 1; j <= order; j++ {
			if data[i] <= data[i-j] || data[i] <= data[i+j] {
				isMax = false
				break
			}
		}
		if isMax {
			peaks = append(peaks, peak{idx: i, val: data[i]})
		}
	}
	return peaks
}

// calcDivStrength calcula la fuerza de la divergencia (1-3)
// Basado en la magnitud de la divergencia entre precio e indicador
func calcDivStrength(price1, price2, ind1, ind2 float64) int {
	// Calcular porcentaje de cambio en precio
	pricePctChange := 0.0
	if price1 != 0 {
		pricePctChange = ((price2 - price1) / price1) * 100
	}
	if pricePctChange < 0 {
		pricePctChange = -pricePctChange
	}

	// Calcular porcentaje de cambio en indicador
	indChange := 0.0
	if ind1 != 0 {
		indChange = ((ind2 - ind1) / ind1) * 100
	}
	if indChange < 0 {
		indChange = -indChange
	}

	// La fuerza depende de cuánto divergen
	totalDivergence := pricePctChange + indChange

	if totalDivergence > 15 {
		return 3 // Fuerte
	} else if totalDivergence > 7 {
		return 2 // Moderada
	}
	return 1 // Débil
}

// buildDivergenceSummary genera un resumen textual para el prompt de DeepSeek
func buildDivergenceSummary(result *DivergenceResult) string {
	if len(result.Divergences) == 0 {
		return "No se detectaron divergencias RSI/MACD vs Precio en la ventana de análisis."
	}

	summary := fmt.Sprintf("Se detectaron %d divergencia(s):\n", len(result.Divergences))

	strengthLabels := map[int]string{1: "DÉBIL", 2: "MODERADA", 3: "FUERTE"}

	for i, d := range result.Divergences {
		stars := ""
		for s := 0; s < d.Strength; s++ {
			stars += "⭐"
		}
		summary += fmt.Sprintf("  %d. [%s] %s %s (%s) %s → señal anticipatoria de %s (hace %d velas)\n",
			i+1, d.Indicator, string(d.Type), stars, strengthLabels[d.Strength], d.Details, d.Signal, d.BarsAgo)
	}

	if result.HasBullish && result.HasBearish {
		summary += "\n⚠️ CONFLICTO: Hay divergencias alcistas Y bajistas simultáneas → señal confusa, precaución."
	} else if result.HasBullish {
		summary += fmt.Sprintf("\n✅ SESGO: Divergencia(s) ALCISTA(s) dominan → señal anticipatoria de COMPRA (fuerza: %s)", strengthLabels[result.MaxStrength])
	} else {
		summary += fmt.Sprintf("\n✅ SESGO: Divergencia(s) BAJISTA(s) dominan → señal anticipatoria de VENTA (fuerza: %s)", strengthLabels[result.MaxStrength])
	}

	return summary
}
