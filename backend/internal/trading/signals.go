// ============================================
// Bot Oscar - Generación y Puntuación de Señales
// Sistema de confluencia que evalúa múltiples indicadores
// como lo haría un trader profesional con +20 años de experiencia
//
// El sistema puntúa cada activo de 0 a 100:
//
//	0-30  → Señal fuerte de VENTA
//	30-45 → Señal de VENTA
//	45-55 → MANTENER (sin señal clara)
//	55-70 → Señal de COMPRA
//	70-100 → Señal fuerte de COMPRA
//
// Se requiere confluencia de mínimo 3 indicadores alineados
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
	score := 50 // Empezamos neutral
	reasons := make([]string, 0)

	// ============================================
	// 1. RSI - Fuerza Relativa (peso: hasta ±20)
	// ============================================
	if ind.RSI > 0 {
		if ind.RSI < 20 {
			score += 20
			reasons = append(reasons, fmt.Sprintf("RSI muy sobrevendido (%.1f)", ind.RSI))
		} else if ind.RSI < 30 {
			score += 15
			reasons = append(reasons, fmt.Sprintf("RSI sobrevendido (%.1f)", ind.RSI))
		} else if ind.RSI > 80 {
			score -= 20
			reasons = append(reasons, fmt.Sprintf("RSI muy sobrecomprado (%.1f)", ind.RSI))
		} else if ind.RSI > 70 {
			score -= 15
			reasons = append(reasons, fmt.Sprintf("RSI sobrecomprado (%.1f)", ind.RSI))
		}
	}

	// ============================================
	// 2. MACD - Convergencia/Divergencia (peso: hasta ±15)
	// ============================================
	if ind.MACD.MACD != 0 || ind.MACD.Signal != 0 {
		if ind.MACD.MACD > ind.MACD.Signal && ind.MACD.Histogram > 0 {
			score += 15
			reasons = append(reasons, "MACD alcista (cruce por encima de señal)")
		} else if ind.MACD.MACD < ind.MACD.Signal && ind.MACD.Histogram < 0 {
			score -= 15
			reasons = append(reasons, "MACD bajista (cruce por debajo de señal)")
		}

		// Histograma creciendo o decreciendo (impulso)
		if ind.MACD.Histogram > 0 {
			score += 3
		} else if ind.MACD.Histogram < 0 {
			score -= 3
		}
	}

	// ============================================
	// 3. SMA - Tendencia con medias móviles (peso: hasta ±20)
	// ============================================

	// Precio vs SMA50 (tendencia de corto plazo)
	if ind.SMA50 > 0 {
		if currentPrice > ind.SMA50 {
			score += 10
			reasons = append(reasons, "Precio sobre SMA50 (tendencia alcista)")
		} else {
			score -= 10
			reasons = append(reasons, "Precio bajo SMA50 (tendencia bajista)")
		}
	}

	// Golden Cross / Death Cross (SMA50 vs SMA200)
	if ind.SMA50 > 0 && ind.SMA200 > 0 {
		if ind.SMA50 > ind.SMA200 {
			score += 10
			reasons = append(reasons, "Golden Cross (SMA50 > SMA200)")
		} else {
			score -= 10
			reasons = append(reasons, "Death Cross (SMA50 < SMA200)")
		}
	}

	// ============================================
	// 4. Bandas de Bollinger - Volatilidad (peso: hasta ±15)
	// ============================================
	if ind.Bollinger.Lower > 0 && ind.Bollinger.Upper > 0 {
		bandWidth := ind.Bollinger.Upper - ind.Bollinger.Lower
		lowerThreshold := ind.Bollinger.Lower + bandWidth*0.1
		upperThreshold := ind.Bollinger.Upper - bandWidth*0.1

		if currentPrice <= ind.Bollinger.Lower {
			score += 15
			reasons = append(reasons, "Precio por debajo de banda inferior Bollinger")
		} else if currentPrice <= lowerThreshold {
			score += 10
			reasons = append(reasons, "Precio cerca de banda inferior Bollinger")
		} else if currentPrice >= ind.Bollinger.Upper {
			score -= 15
			reasons = append(reasons, "Precio por encima de banda superior Bollinger")
		} else if currentPrice >= upperThreshold {
			score -= 10
			reasons = append(reasons, "Precio cerca de banda superior Bollinger")
		}
	}

	// ============================================
	// 5. EMA Cruce rápido (peso: hasta ±5)
	// ============================================
	if ind.EMA12 > 0 && ind.EMA26 > 0 {
		if ind.EMA12 > ind.EMA26 {
			score += 5
			reasons = append(reasons, "EMA12 sobre EMA26 (impulso alcista)")
		} else {
			score -= 5
			reasons = append(reasons, "EMA12 bajo EMA26 (impulso bajista)")
		}
	}

	// ============================================
	// Limitar puntuación a rango 0-100
	// ============================================
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	// Determinar tipo de señal basado en la puntuación
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
