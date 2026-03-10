// ============================================
// Bot Oscar - Generación y Puntuación de Señales
// Sistema de confluencia profesional que evalúa múltiples indicadores
// como lo haría un trader institucional con +20 años de experiencia
//
// Distribución de pesos (total teórico: ±100 desde base 50):
//
//	TENDENCIA (peso máximo: ±35)
//	  - Precio vs EMA200:        ±12  (la más importante)
//	  - EMA50 vs EMA200:         ±10  (Golden/Death Cross exponencial)
//	  - Precio vs EMA50:          ±8  (tendencia intermedia)
//	  - Precio vs EMA21:          ±5  (pullback)
//
//	MOMENTUM (peso máximo: ±20)
//	  - MACD cruce + histograma: ±12
//	  - RSI zonas extremas:       ±8
//
//	VOLUMEN (peso máximo: ±15)
//	  - Precio vs VWAP:          ±10  (clave institucional)
//	  - Pico de volumen:          ±5  (confirma movimiento)
//
//	VOLATILIDAD (peso máximo: ±15)
//	  - Bollinger Bands:         ±10
//	  - EMA12 vs EMA26:          ±5  (cruce rápido)
//
//	ESTRUCTURA (peso máximo: ±15)
//	  - SMA50 vs SMA200:         ±8  (Golden/Death Cross clásico)
//	  - Precio vs SMA200:        ±7  (soporte/resistencia dinámica)
//
// Resultado:
//
//	0-30  → Señal fuerte de VENTA
//	30-45 → Señal de VENTA
//	45-55 → MANTENER (sin confluencia clara)
//	55-70 → Señal de COMPRA
//	70-100 → Señal fuerte de COMPRA
//
// ============================================
package trading

import (
	"fmt"
	"strings"

	"bot-oscar/internal/models"
)

// ScoreSignal evalúa todos los indicadores y genera una puntuación de confluencia
// Retorna: puntuación (0-100), tipo de señal, y razón detallada
func ScoreSignal(ind models.IndicatorValues, currentPrice float64) (int, string, string) {
	score := 50 // Base neutral
	reasons := make([]string, 0)

	// ============================================
	// 1. TENDENCIA PRINCIPAL - EMA 200 (peso: ±12)
	//    La EMA200 es EL indicador más importante.
	//    Si el precio está encima = mercado alcista.
	//    Si está debajo = mercado bajista.
	// ============================================
	if ind.EMA200 > 0 {
		if currentPrice > ind.EMA200 {
			score += 12
			distancia := ((currentPrice - ind.EMA200) / ind.EMA200) * 100
			reasons = append(reasons, fmt.Sprintf("Precio SOBRE EMA200 (+%.1f%%) → tendencia alcista", distancia))
		} else {
			score -= 12
			distancia := ((ind.EMA200 - currentPrice) / ind.EMA200) * 100
			reasons = append(reasons, fmt.Sprintf("Precio BAJO EMA200 (-%.1f%%) → tendencia bajista", distancia))
		}
	}

	// ============================================
	// 2. TENDENCIA - EMA 50 vs EMA 200 (peso: ±10)
	//    Golden Cross exponencial = EMA50 cruza por encima de EMA200
	//    Death Cross exponencial = EMA50 cruza por debajo de EMA200
	// ============================================
	if ind.EMA50 > 0 && ind.EMA200 > 0 {
		if ind.EMA50 > ind.EMA200 {
			score += 10
			reasons = append(reasons, "Golden Cross EMA (EMA50 > EMA200)")
		} else {
			score -= 10
			reasons = append(reasons, "Death Cross EMA (EMA50 < EMA200)")
		}
	}

	// ============================================
	// 3. TENDENCIA - Precio vs EMA 50 (peso: ±8)
	//    Confirma la tendencia intermedia
	// ============================================
	if ind.EMA50 > 0 {
		if currentPrice > ind.EMA50 {
			score += 8
			reasons = append(reasons, "Precio sobre EMA50 (tendencia intermedia alcista)")
		} else {
			score -= 8
			reasons = append(reasons, "Precio bajo EMA50 (tendencia intermedia bajista)")
		}
	}

	// ============================================
	// 4. TENDENCIA - Precio vs EMA 21 (peso: ±5)
	//    La EMA21 es para pullbacks y entradas en tendencia
	// ============================================
	if ind.EMA21 > 0 {
		if currentPrice > ind.EMA21 {
			score += 5
			reasons = append(reasons, "Precio sobre EMA21 (pullback alcista)")
		} else {
			score -= 5
			reasons = append(reasons, "Precio bajo EMA21 (pullback bajista)")
		}
	}

	// ============================================
	// 5. MOMENTUM - MACD (peso: ±12)
	//    Cruce de MACD con su línea de señal + histograma
	// ============================================
	if ind.MACD.MACD != 0 || ind.MACD.Signal != 0 {
		if ind.MACD.MACD > ind.MACD.Signal && ind.MACD.Histogram > 0 {
			score += 10
			reasons = append(reasons, "MACD alcista (cruce por encima de señal)")
		} else if ind.MACD.MACD < ind.MACD.Signal && ind.MACD.Histogram < 0 {
			score -= 10
			reasons = append(reasons, "MACD bajista (cruce por debajo de señal)")
		}
		// Histograma confirma impulso
		if ind.MACD.Histogram > 0 {
			score += 2
		} else if ind.MACD.Histogram < 0 {
			score -= 2
		}
	}

	// ============================================
	// 6. MOMENTUM - RSI (peso: ±8)
	//    Sobrecompra (>70) y sobreventa (<30)
	// ============================================
	if ind.RSI > 0 {
		if ind.RSI < 20 {
			score += 8
			reasons = append(reasons, fmt.Sprintf("RSI MUY sobrevendido (%.1f) → probable rebote fuerte", ind.RSI))
		} else if ind.RSI < 30 {
			score += 6
			reasons = append(reasons, fmt.Sprintf("RSI sobrevendido (%.1f) → posible rebote", ind.RSI))
		} else if ind.RSI > 80 {
			score -= 8
			reasons = append(reasons, fmt.Sprintf("RSI MUY sobrecomprado (%.1f) → probable corrección fuerte", ind.RSI))
		} else if ind.RSI > 70 {
			score -= 6
			reasons = append(reasons, fmt.Sprintf("RSI sobrecomprado (%.1f) → posible corrección", ind.RSI))
		}
	}

	// ============================================
	// 7. VOLUMEN - VWAP (peso: ±10)
	//    Los institucionales compran debajo del VWAP
	//    y venden encima del VWAP
	// ============================================
	if ind.VWAP > 0 {
		if currentPrice > ind.VWAP {
			score -= 5 // Precio caro respecto al volumen → presión vendedora
			reasons = append(reasons, fmt.Sprintf("Precio sobre VWAP ($%.2f) → presión vendedora institucional", ind.VWAP))
		} else {
			score += 5 // Precio barato respecto al volumen → compra institucional
			reasons = append(reasons, fmt.Sprintf("Precio bajo VWAP ($%.2f) → zona de compra institucional", ind.VWAP))
		}

		// VWAP + tendencia: si precio está sobre VWAP Y sobre EMA200 → muy alcista
		if ind.EMA200 > 0 {
			if currentPrice > ind.VWAP && currentPrice > ind.EMA200 {
				score += 5
				reasons = append(reasons, "VWAP + EMA200 confirman tendencia alcista")
			} else if currentPrice < ind.VWAP && currentPrice < ind.EMA200 {
				score -= 5
				reasons = append(reasons, "VWAP + EMA200 confirman tendencia bajista")
			}
		}
	}

	// ============================================
	// 8. VOLUMEN - Picos de volumen (peso: ±5)
	//    Volumen alto confirma la dirección del movimiento
	// ============================================
	if ind.VolumenRatio > 1.5 {
		// Hay pico de volumen - confirma la dirección actual
		if currentPrice > ind.EMA21 && ind.EMA21 > 0 {
			score += 5
			reasons = append(reasons, fmt.Sprintf("Pico de volumen (%.1fx) confirma impulso alcista", ind.VolumenRatio))
		} else if currentPrice < ind.EMA21 && ind.EMA21 > 0 {
			score -= 5
			reasons = append(reasons, fmt.Sprintf("Pico de volumen (%.1fx) confirma impulso bajista", ind.VolumenRatio))
		}
	}

	// ============================================
	// 9. VOLATILIDAD - Bollinger Bands (peso: ±10)
	//    Precio en bandas extremas indica posible reversión
	// ============================================
	if ind.Bollinger.Lower > 0 && ind.Bollinger.Upper > 0 {
		bandWidth := ind.Bollinger.Upper - ind.Bollinger.Lower
		lowerZone := ind.Bollinger.Lower + bandWidth*0.1
		upperZone := ind.Bollinger.Upper - bandWidth*0.1

		if currentPrice <= ind.Bollinger.Lower {
			score += 10
			reasons = append(reasons, "Precio DEBAJO de banda inferior Bollinger (sobreventa extrema)")
		} else if currentPrice <= lowerZone {
			score += 6
			reasons = append(reasons, "Precio cerca de banda inferior Bollinger")
		} else if currentPrice >= ind.Bollinger.Upper {
			score -= 10
			reasons = append(reasons, "Precio ENCIMA de banda superior Bollinger (sobrecompra extrema)")
		} else if currentPrice >= upperZone {
			score -= 6
			reasons = append(reasons, "Precio cerca de banda superior Bollinger")
		}
	}

	// ============================================
	// 10. VOLATILIDAD - EMA12 vs EMA26 (peso: ±5)
	//     Cruce rápido de EMAs confirma cambio de momentum
	// ============================================
	if ind.EMA12 > 0 && ind.EMA26 > 0 {
		if ind.EMA12 > ind.EMA26 {
			score += 5
			reasons = append(reasons, "EMA12 > EMA26 (impulso alcista a corto plazo)")
		} else {
			score -= 5
			reasons = append(reasons, "EMA12 < EMA26 (impulso bajista a corto plazo)")
		}
	}

	// ============================================
	// 11. ESTRUCTURA - SMA50 vs SMA200 (peso: ±8)
	//     Golden Cross / Death Cross clásico
	// ============================================
	if ind.SMA50 > 0 && ind.SMA200 > 0 {
		if ind.SMA50 > ind.SMA200 {
			score += 8
			reasons = append(reasons, "Golden Cross clásico (SMA50 > SMA200)")
		} else {
			score -= 8
			reasons = append(reasons, "Death Cross clásico (SMA50 < SMA200)")
		}
	}

	// ============================================
	// 12. ESTRUCTURA - Precio vs SMA200 (peso: ±7)
	//     SMA200 como soporte/resistencia dinámica
	// ============================================
	if ind.SMA200 > 0 {
		if currentPrice > ind.SMA200 {
			score += 7
			reasons = append(reasons, "Precio sobre SMA200 (soporte dinámico respetado)")
		} else {
			score -= 7
			reasons = append(reasons, "Precio bajo SMA200 (resistencia dinámica)")
		}
	}

	// ============================================
	// Limitar puntuación al rango 0-100
	// ============================================
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	// ============================================
	// Determinar tipo de señal basado en puntuación
	// ============================================
	signalType := "MANTENER"
	if score >= 65 {
		signalType = "COMPRA"
	} else if score <= 35 {
		signalType = "VENTA"
	}

	reason := strings.Join(reasons, " | ")
	if reason == "" {
		reason = "Sin señales claras de indicadores"
	}

	return score, signalType, reason
}
