// ============================================
// Bot Oscar - Índice de Miedo y Codicia por Activo
// Calcula un índice personalizado de Fear & Greed (0-100) para cada acción,
// basado en los indicadores técnicos individuales del activo.
//
// IMPORTANTE: Cada activo se calcula de forma independiente.
//
//	Nunca se mezclan datos de un activo con otro.
//
// Escala (inspirada en el CNN Fear & Greed Index):
//
//	 0-20  → Miedo Extremo
//	21-40  → Miedo
//	41-60  → Neutral
//	61-80  → Codicia
//	81-100 → Codicia Extrema
//
// Componentes del cálculo (7 factores, como el CNN Index):
//  1. Momentum (RSI)                    → peso 20%
//  2. Fuerza de Tendencia (EMAs)        → peso 20%
//  3. Volatilidad (Bollinger + ATR)     → peso 15%
//  4. Volumen (ratio + VWAP)            → peso 15%
//  5. Cruces de Medias Móviles          → peso 10%
//  6. MACD Momentum                     → peso 10%
//  7. Posición en rango de precios      → peso 10%
//
// ============================================
package trading

import (
	"math"

	"bot-oscar/internal/models"
)

// FearGreedResult contiene el resultado del índice de miedo/codicia para un activo
type FearGreedResult struct {
	Symbol      string          `json:"symbol"`
	AssetName   string          `json:"assetName"`
	Score       int             `json:"score"`       // 0-100
	Label       string          `json:"label"`       // "Miedo Extremo", "Miedo", "Neutral", "Codicia", "Codicia Extrema"
	Description string          `json:"description"` // Explicación breve en español
	Components  []FearGreedComp `json:"components"`  // Desglose de cada componente
}

// FearGreedComp es un componente individual del índice
type FearGreedComp struct {
	Name   string  `json:"name"`   // Nombre del factor
	Score  float64 `json:"score"`  // Puntuación parcial 0-100
	Weight float64 `json:"weight"` // Peso en el total (0-1)
	Detail string  `json:"detail"` // Explicación corta
}

// CalculateFearGreed calcula el índice de Miedo & Codicia para UN activo individual
// Usa EXCLUSIVAMENTE los indicadores de ese activo, sin mezclar con otros
func CalculateFearGreed(ind models.IndicatorValues, currentPrice float64, assetName string) FearGreedResult {
	components := make([]FearGreedComp, 0, 7)

	// ============================================
	// 1. MOMENTUM - RSI (peso: 20%)
	// RSI < 30 = miedo (sobrevendido, la gente vende por pánico)
	// RSI > 70 = codicia (sobrecomprado, la gente compra por FOMO)
	// RSI 30-70 = escala lineal entre miedo y codicia
	// ============================================
	var rsiScore float64
	rsiDetail := ""
	if ind.RSI > 0 {
		if ind.RSI <= 20 {
			rsiScore = 5 // Miedo extremo
			rsiDetail = "RSI muy sobrevendido → pánico en el mercado"
		} else if ind.RSI <= 30 {
			rsiScore = 15 + (ind.RSI-20)*2 // 15-35
			rsiDetail = "RSI sobrevendido → miedo predomina"
		} else if ind.RSI <= 50 {
			rsiScore = 35 + (ind.RSI-30)*0.75 // 35-50
			rsiDetail = "RSI en zona baja-neutral"
		} else if ind.RSI <= 70 {
			rsiScore = 50 + (ind.RSI-50)*1.25 // 50-75
			rsiDetail = "RSI en zona alta-neutral"
		} else if ind.RSI <= 80 {
			rsiScore = 75 + (ind.RSI-70)*1.5 // 75-90
			rsiDetail = "RSI sobrecomprado → codicia"
		} else {
			rsiScore = 95 // Codicia extrema
			rsiDetail = "RSI muy sobrecomprado → codicia extrema"
		}
	} else {
		rsiScore = 50
		rsiDetail = "Sin datos de RSI"
	}
	components = append(components, FearGreedComp{
		Name: "Momentum (RSI)", Score: rsiScore, Weight: 0.20, Detail: rsiDetail,
	})

	// ============================================
	// 2. FUERZA DE TENDENCIA - EMAs (peso: 20%)
	// Precio sobre EMAs = codicia (tendencia alcista fuerte)
	// Precio bajo EMAs = miedo (tendencia bajista)
	// ============================================
	var trendScore float64
	trendDetail := ""
	trendPoints := 0
	trendMax := 0

	if ind.EMA200 > 0 {
		trendMax += 3
		if currentPrice > ind.EMA200 {
			dist := ((currentPrice - ind.EMA200) / ind.EMA200) * 100
			trendPoints += 3
			if dist > 10 {
				trendPoints++ // Bonus si está muy encima
			}
			trendDetail = "Precio sobre EMA200"
		} else {
			trendDetail = "Precio bajo EMA200"
		}
	}
	if ind.EMA50 > 0 {
		trendMax += 2
		if currentPrice > ind.EMA50 {
			trendPoints += 2
		}
	}
	if ind.EMA21 > 0 {
		trendMax += 1
		if currentPrice > ind.EMA21 {
			trendPoints += 1
		}
	}

	if trendMax > 0 {
		trendScore = (float64(trendPoints) / float64(trendMax)) * 100
		if trendScore > 100 {
			trendScore = 100
		}
	} else {
		trendScore = 50
		trendDetail = "Sin datos de tendencia"
	}
	components = append(components, FearGreedComp{
		Name: "Tendencia (EMAs)", Score: trendScore, Weight: 0.20, Detail: trendDetail,
	})

	// ============================================
	// 3. VOLATILIDAD - Bollinger + ATR (peso: 15%)
	// Alta volatilidad = miedo (mercado nervioso)
	// Baja volatilidad = codicia (mercado confiado)
	// Precio cerca de banda inferior = miedo
	// Precio cerca de banda superior = codicia
	// ============================================
	var volScore float64
	volDetail := ""
	if ind.Bollinger.Lower > 0 && ind.Bollinger.Upper > 0 {
		bandWidth := ind.Bollinger.Upper - ind.Bollinger.Lower
		if bandWidth > 0 {
			// Posición del precio dentro de las bandas (0 = banda inferior, 1 = banda superior)
			position := (currentPrice - ind.Bollinger.Lower) / bandWidth
			position = math.Max(0, math.Min(1, position))
			volScore = position * 100

			if position < 0.2 {
				volDetail = "Precio en zona baja de Bollinger → miedo"
			} else if position > 0.8 {
				volDetail = "Precio en zona alta de Bollinger → codicia"
			} else {
				volDetail = "Precio en zona media de Bollinger"
			}
		}
	} else {
		volScore = 50
		volDetail = "Sin datos de volatilidad"
	}
	components = append(components, FearGreedComp{
		Name: "Volatilidad (Bollinger)", Score: volScore, Weight: 0.15, Detail: volDetail,
	})

	// ============================================
	// 4. VOLUMEN - Ratio + VWAP (peso: 15%)
	// Volumen alto + precio subiendo = codicia
	// Volumen alto + precio bajando = miedo (venta pánico)
	// Precio sobre VWAP = compradores agresivos (codicia)
	// Precio bajo VWAP = vendedores agresivos (miedo)
	// ============================================
	var volumeScore float64
	volumeDetail := ""

	if ind.VWAP > 0 {
		if currentPrice > ind.VWAP {
			volumeScore = 65
			volumeDetail = "Precio sobre VWAP → presión compradora"
		} else {
			volumeScore = 35
			volumeDetail = "Precio bajo VWAP → presión vendedora"
		}

		// Ajustar por ratio de volumen
		if ind.VolumenRatio > 2.0 {
			// Volumen extremo: amplifica la dirección
			if currentPrice > ind.VWAP {
				volumeScore = math.Min(95, volumeScore+20)
				volumeDetail = "Volumen extremo + compradores → codicia fuerte"
			} else {
				volumeScore = math.Max(5, volumeScore-20)
				volumeDetail = "Volumen extremo + vendedores → miedo fuerte"
			}
		} else if ind.VolumenRatio > 1.5 {
			if currentPrice > ind.VWAP {
				volumeScore = math.Min(85, volumeScore+10)
			} else {
				volumeScore = math.Max(15, volumeScore-10)
			}
		}
	} else {
		volumeScore = 50
		volumeDetail = "Sin datos de volumen"
	}
	components = append(components, FearGreedComp{
		Name: "Volumen (VWAP)", Score: volumeScore, Weight: 0.15, Detail: volumeDetail,
	})

	// ============================================
	// 5. CRUCES DE MEDIAS MÓVILES (peso: 10%)
	// Golden Cross (SMA50 > SMA200) = codicia
	// Death Cross (SMA50 < SMA200) = miedo
	// ============================================
	var crossScore float64
	crossDetail := ""
	if ind.SMA50 > 0 && ind.SMA200 > 0 {
		if ind.SMA50 > ind.SMA200 {
			// Cuánto por encima? más separación = más codicia
			dist := ((ind.SMA50 - ind.SMA200) / ind.SMA200) * 100
			crossScore = 70 + math.Min(30, dist*3)
			crossDetail = "Golden Cross activo → confianza alcista"
		} else {
			dist := ((ind.SMA200 - ind.SMA50) / ind.SMA200) * 100
			crossScore = 30 - math.Min(30, dist*3)
			if crossScore < 0 {
				crossScore = 0
			}
			crossDetail = "Death Cross activo → temor bajista"
		}
	} else {
		crossScore = 50
		crossDetail = "Sin datos de cruces"
	}
	components = append(components, FearGreedComp{
		Name: "Cruces de Medias", Score: crossScore, Weight: 0.10, Detail: crossDetail,
	})

	// ============================================
	// 6. MACD MOMENTUM (peso: 10%)
	// MACD positivo + histograma creciente = codicia
	// MACD negativo + histograma decreciente = miedo
	// ============================================
	var macdScore float64
	macdDetail := ""
	if ind.MACD.MACD != 0 || ind.MACD.Signal != 0 {
		if ind.MACD.MACD > ind.MACD.Signal && ind.MACD.Histogram > 0 {
			macdScore = 70 + math.Min(30, math.Abs(ind.MACD.Histogram)*100)
			macdDetail = "MACD alcista con momentum → codicia"
		} else if ind.MACD.MACD < ind.MACD.Signal && ind.MACD.Histogram < 0 {
			macdScore = 30 - math.Min(30, math.Abs(ind.MACD.Histogram)*100)
			if macdScore < 0 {
				macdScore = 0
			}
			macdDetail = "MACD bajista con momentum → miedo"
		} else if ind.MACD.MACD > ind.MACD.Signal {
			macdScore = 60
			macdDetail = "MACD alcista débil"
		} else {
			macdScore = 40
			macdDetail = "MACD bajista débil"
		}
	} else {
		macdScore = 50
		macdDetail = "Sin datos de MACD"
	}
	components = append(components, FearGreedComp{
		Name: "MACD Momentum", Score: macdScore, Weight: 0.10, Detail: macdDetail,
	})

	// ============================================
	// 7. POSICIÓN EN RANGO DE PRECIOS (peso: 10%)
	// Evalúa dónde está el precio respecto a SMA200 y Bollinger
	// Precio muy encima de SMA200 = codicia (euforia)
	// Precio muy debajo de SMA200 = miedo (depresión)
	// ============================================
	var rangeScore float64
	rangeDetail := ""
	if ind.SMA200 > 0 {
		distPercent := ((currentPrice - ind.SMA200) / ind.SMA200) * 100
		// Mapear: -20% → 0, 0% → 50, +20% → 100
		rangeScore = 50 + distPercent*2.5
		rangeScore = math.Max(0, math.Min(100, rangeScore))

		if distPercent > 10 {
			rangeDetail = "Precio muy encima de SMA200 → euforia"
		} else if distPercent > 0 {
			rangeDetail = "Precio sobre SMA200 → optimismo"
		} else if distPercent > -10 {
			rangeDetail = "Precio bajo SMA200 → pesimismo"
		} else {
			rangeDetail = "Precio muy bajo SMA200 → pánico"
		}
	} else {
		rangeScore = 50
		rangeDetail = "Sin datos de rango"
	}
	components = append(components, FearGreedComp{
		Name: "Rango de Precio", Score: rangeScore, Weight: 0.10, Detail: rangeDetail,
	})

	// ============================================
	// Calcular score ponderado final
	// ============================================
	var totalScore float64
	for _, c := range components {
		totalScore += c.Score * c.Weight
	}

	finalScore := int(math.Round(totalScore))
	if finalScore < 0 {
		finalScore = 0
	}
	if finalScore > 100 {
		finalScore = 100
	}

	// Determinar etiqueta y descripción
	label, description := getFearGreedLabel(finalScore, ind.Symbol)

	return FearGreedResult{
		Symbol:      ind.Symbol,
		AssetName:   assetName,
		Score:       finalScore,
		Label:       label,
		Description: description,
		Components:  components,
	}
}

// getFearGreedLabel retorna la etiqueta y descripción según la puntuación
func getFearGreedLabel(score int, symbol string) (string, string) {
	switch {
	case score <= 20:
		return "Miedo Extremo",
			"El mercado muestra pánico extremo en " + symbol + ". Los indicadores señalan sobreventa masiva y pesimismo generalizado. Históricamente, estos niveles pueden representar oportunidades de compra."
	case score <= 40:
		return "Miedo",
			"Predomina el miedo en " + symbol + ". Los indicadores técnicos muestran debilidad y presión vendedora. El sentimiento es negativo pero no extremo."
	case score <= 60:
		return "Neutral",
			"El sentimiento en " + symbol + " está equilibrado. No hay señales claras de miedo ni codicia. El mercado está en espera de un catalizador."
	case score <= 80:
		return "Codicia",
			"La codicia domina en " + symbol + ". Los indicadores muestran fortaleza y optimismo. Los compradores son agresivos, pero podría haber una corrección."
	default:
		return "Codicia Extrema",
			"Euforia extrema en " + symbol + ". Todos los indicadores apuntan a sobrecompra. Históricamente, estos niveles pueden preceder correcciones."
	}
}
