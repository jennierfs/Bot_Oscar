// ============================================
// Bot Oscar - Indicador: Bandas de Bollinger
// Miden la volatilidad del mercado
//
// Componentes:
//   - Banda media = SMA(20)
//   - Banda superior = SMA + (2 × desviación estándar)
//   - Banda inferior = SMA - (2 × desviación estándar)
//
// Señales clave:
//   - Precio toca banda inferior → posible rebote alcista
//   - Precio toca banda superior → posible retroceso bajista
//   - Bandas estrechas → baja volatilidad, posible ruptura próxima
//   - Bandas amplias → alta volatilidad
//
// ============================================
package indicators

import "math"

// BollingerResult contiene las tres bandas de Bollinger
type BollingerResult struct {
	Upper  []float64 // Banda superior
	Middle []float64 // Banda media (SMA)
	Lower  []float64 // Banda inferior
}

// CalculateBollinger calcula las Bandas de Bollinger
// Parámetros estándar: period=20, numStdDev=2.0
func CalculateBollinger(prices []float64, period int, numStdDev float64) *BollingerResult {
	if period <= 0 {
		return nil
	}

	// La banda media es la SMA
	sma := CalculateSMA(prices, period)
	if sma == nil {
		return nil
	}

	upper := make([]float64, len(sma))
	lower := make([]float64, len(sma))

	// Para cada punto, calcular la desviación estándar de la ventana
	for i := range sma {
		// Ventana de precios para este punto
		windowStart := i
		windowEnd := i + period

		// Calcular desviación estándar de la ventana
		sumSqDiff := 0.0
		for j := windowStart; j < windowEnd; j++ {
			diff := prices[j] - sma[i]
			sumSqDiff += diff * diff
		}
		stdDev := math.Sqrt(sumSqDiff / float64(period))

		// Banda superior y inferior
		upper[i] = sma[i] + numStdDev*stdDev
		lower[i] = sma[i] - numStdDev*stdDev
	}

	return &BollingerResult{
		Upper:  upper,
		Middle: sma,
		Lower:  lower,
	}
}
