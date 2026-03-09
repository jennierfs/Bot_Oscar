// ============================================
// Bot Oscar - Indicador: MACD
// Moving Average Convergence Divergence
// Detecta cambios en la fuerza, dirección y duración de una tendencia
//
// Componentes:
//   - Línea MACD = EMA rápida - EMA lenta (normalmente EMA12 - EMA26)
//   - Línea Signal = EMA de la línea MACD (normalmente EMA9 del MACD)
//   - Histograma = MACD - Signal (visualiza la divergencia)
//
// Señales clave:
//   - MACD cruza por encima de Signal → Señal alcista (compra)
//   - MACD cruza por debajo de Signal → Señal bajista (venta)
//   - Histograma creciente → Impulso aumentando
//
// ============================================
package indicators

// MACDResult contiene las tres líneas del indicador MACD
type MACDResult struct {
	MACD      []float64 // Línea MACD (diferencia entre EMAs)
	Signal    []float64 // Línea de señal (EMA del MACD)
	Histogram []float64 // Histograma (MACD - Signal)
}

// CalculateMACD calcula el MACD con los periodos especificados
// Parámetros estándar: fast=12, slow=26, signal=9
func CalculateMACD(prices []float64, fastPeriod, slowPeriod, signalPeriod int) *MACDResult {
	if len(prices) < slowPeriod || fastPeriod <= 0 || slowPeriod <= 0 || signalPeriod <= 0 {
		return nil
	}

	// Calcular EMA rápida y EMA lenta
	emaFast := CalculateEMA(prices, fastPeriod)
	emaSlow := CalculateEMA(prices, slowPeriod)

	if emaFast == nil || emaSlow == nil {
		return nil
	}

	// Alinear las EMAs (la lenta tiene menos valores)
	offset := len(emaFast) - len(emaSlow)
	if offset < 0 {
		return nil
	}
	emaFastAligned := emaFast[offset:]

	// Línea MACD = EMA rápida - EMA lenta
	macdLine := make([]float64, len(emaSlow))
	for i := range macdLine {
		macdLine[i] = emaFastAligned[i] - emaSlow[i]
	}

	// Línea Signal = EMA del MACD
	signalLine := CalculateEMA(macdLine, signalPeriod)
	if signalLine == nil {
		return nil
	}

	// Alinear MACD con Signal
	macdOffset := len(macdLine) - len(signalLine)
	macdAligned := macdLine[macdOffset:]

	// Histograma = MACD - Signal
	histogram := make([]float64, len(signalLine))
	for i := range histogram {
		histogram[i] = macdAligned[i] - signalLine[i]
	}

	return &MACDResult{
		MACD:      macdAligned,
		Signal:    signalLine,
		Histogram: histogram,
	}
}
