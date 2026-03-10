// ============================================
// Bot Oscar - Indicador: VWAP
// Precio Promedio Ponderado por Volumen
// (Volume Weighted Average Price)
//
// El VWAP es el indicador #1 de los traders institucionales.
// Los fondos grandes compran cuando el precio está DEBAJO del VWAP
// y venden cuando está ENCIMA, porque indica "precio justo".
//
// Fórmula: VWAP = Σ(Precio_Típico × Volumen) / Σ(Volumen)
// Precio Típico = (High + Low + Close) / 3
// ============================================
package indicators

// VWAPResult contiene el resultado del cálculo VWAP
type VWAPResult struct {
	VWAP         float64 // Precio promedio ponderado por volumen
	VolumenProm  int64   // Volumen promedio del periodo
	VolumenHoy   int64   // Volumen del último día
	VolumenRatio float64 // Ratio volumen actual vs promedio
}

// CalculateVWAP calcula el VWAP usando datos OHLCV
// Recibe arrays paralelos de high, low, close y volume
// period = número de días para el cálculo (normalmente 20)
func CalculateVWAP(highs, lows, closes []float64, volumes []int64, period int) *VWAPResult {
	n := len(closes)
	if n < period || len(highs) != n || len(lows) != n || len(volumes) != n {
		return nil
	}

	// Usar solo los últimos 'period' días para el VWAP
	start := n - period

	var sumPV float64  // Suma de (Precio Típico × Volumen)
	var sumVol int64   // Suma de Volumen
	var volTotal int64 // Para calcular promedio de volumen

	for i := start; i < n; i++ {
		// Precio típico = (High + Low + Close) / 3
		precioTipico := (highs[i] + lows[i] + closes[i]) / 3.0
		sumPV += precioTipico * float64(volumes[i])
		sumVol += volumes[i]
		volTotal += volumes[i]
	}

	// Evitar división por cero
	if sumVol == 0 {
		return nil
	}

	// VWAP = Σ(PT × V) / Σ(V)
	vwap := sumPV / float64(sumVol)

	// Volumen promedio diario
	volProm := volTotal / int64(period)

	// Volumen del último día
	volHoy := volumes[n-1]

	// Ratio de volumen: si es > 1.5 hay un pico de volumen importante
	var volRatio float64
	if volProm > 0 {
		volRatio = float64(volHoy) / float64(volProm)
	}

	return &VWAPResult{
		VWAP:         vwap,
		VolumenProm:  volProm,
		VolumenHoy:   volHoy,
		VolumenRatio: volRatio,
	}
}
