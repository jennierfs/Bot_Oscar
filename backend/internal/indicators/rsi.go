// ============================================
// Bot Oscar - Indicador: RSI (Relative Strength Index)
// Mide la velocidad y magnitud de los cambios de precio
// Valores: 0-100
//
//	< 30 = Sobrevendido (posible señal de compra)
//	> 70 = Sobrecomprado (posible señal de venta)
//
// Periodo estándar: 14
// ============================================
package indicators

import "math"

// CalculateRSI calcula el Índice de Fuerza Relativa
// Fórmula: RSI = 100 - (100 / (1 + RS))
// donde RS = Ganancia Promedio / Pérdida Promedio
func CalculateRSI(prices []float64, period int) []float64 {
	if len(prices) < period+1 || period <= 0 {
		return nil
	}

	// Calcular los cambios de precio entre cada par de periodos
	changes := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		changes[i-1] = prices[i] - prices[i-1]
	}

	// Calcular la primera ganancia y pérdida promedio (SMA)
	var avgGain, avgLoss float64
	for i := 0; i < period; i++ {
		if changes[i] > 0 {
			avgGain += changes[i]
		} else {
			avgLoss += math.Abs(changes[i])
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	result := make([]float64, 0, len(changes)-period+1)

	// Primer valor RSI
	if avgLoss == 0 {
		result = append(result, 100)
	} else {
		rs := avgGain / avgLoss
		result = append(result, 100-(100/(1+rs)))
	}

	// RSI subsecuentes usando media suavizada (método de Wilder)
	// avgGain = (avgGain_prev * (period-1) + ganancia_actual) / period
	for i := period; i < len(changes); i++ {
		gain := 0.0
		loss := 0.0

		if changes[i] > 0 {
			gain = changes[i]
		} else {
			loss = math.Abs(changes[i])
		}

		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)

		if avgLoss == 0 {
			result = append(result, 100)
		} else {
			rs := avgGain / avgLoss
			result = append(result, 100-(100/(1+rs)))
		}
	}

	return result
}
