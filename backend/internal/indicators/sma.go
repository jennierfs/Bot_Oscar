// ============================================
// Bot Oscar - Indicadores: SMA y EMA
// Media Móvil Simple y Media Móvil Exponencial
// Usados por traders profesionales para identificar tendencias
// ============================================
package indicators

// CalculateSMA calcula la Media Móvil Simple (Simple Moving Average)
// La SMA suaviza el ruido del precio promediando los últimos N periodos
// Ejemplo: SMA(50) = promedio de los últimos 50 cierres
func CalculateSMA(prices []float64, period int) []float64 {
	if len(prices) < period || period <= 0 {
		return nil
	}

	result := make([]float64, len(prices)-period+1)

	// Calcular la primera SMA sumando los primeros N valores
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	result[0] = sum / float64(period)

	// Las siguientes SMAs usan ventana deslizante (más eficiente)
	for i := 1; i < len(result); i++ {
		sum = sum - prices[i-1] + prices[i+period-1]
		result[i] = sum / float64(period)
	}

	return result
}

// CalculateEMA calcula la Media Móvil Exponencial (Exponential Moving Average)
// La EMA da más peso a los precios recientes, reacciona más rápido que la SMA
// Multiplicador = 2 / (periodo + 1)
func CalculateEMA(prices []float64, period int) []float64 {
	if len(prices) < period || period <= 0 {
		return nil
	}

	result := make([]float64, len(prices))
	multiplier := 2.0 / float64(period+1)

	// La primera EMA es una SMA de los primeros N valores
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	result[period-1] = sum / float64(period)

	// Cada EMA posterior: EMA = Precio * k + EMA_anterior * (1 - k)
	for i := period; i < len(prices); i++ {
		result[i] = (prices[i]-result[i-1])*multiplier + result[i-1]
	}

	// Devolver solo los valores válidos (desde el periodo en adelante)
	return result[period-1:]
}
