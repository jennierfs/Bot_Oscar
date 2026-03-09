// ============================================
// Bot Oscar - Gestión de Riesgo
// Implementa las reglas de un trader profesional:
// - Nunca arriesgar más del 1-2% del capital por operación
// - Siempre usar stop loss basado en ATR
// - Ratio riesgo/beneficio mínimo de 1:2
// - Control de tamaño de posición
// ============================================
package trading

import (
	"math"

	"bot-oscar/internal/models"
)

// CalculateATR calcula el Average True Range (Rango Verdadero Promedio)
// El ATR mide la volatilidad real de un activo
// Se usa para determinar dónde colocar el stop loss
func CalculateATR(prices []models.Price, period int) float64 {
	if len(prices) < period+1 || period <= 0 {
		return 0
	}

	var atrSum float64

	// Calcular True Range para cada periodo
	for i := len(prices) - period; i < len(prices); i++ {
		// True Range = max de:
		//   1. High - Low (rango del día)
		//   2. |High - Cierre anterior| (gap alcista)
		//   3. |Low - Cierre anterior| (gap bajista)
		tr := math.Max(
			prices[i].High-prices[i].Low,
			math.Max(
				math.Abs(prices[i].High-prices[i-1].Close),
				math.Abs(prices[i].Low-prices[i-1].Close),
			),
		)
		atrSum += tr
	}

	return atrSum / float64(period)
}

// CalculateStopLoss calcula el precio de stop loss basado en ATR
// Un trader profesional usa 2x ATR como distancia del stop loss
func CalculateStopLoss(entryPrice float64, signalType string, atr float64) float64 {
	multiplier := 2.0 // 2x ATR es estándar profesional

	if atr == 0 {
		// Si no hay ATR, usar 2% del precio como fallback
		atr = entryPrice * 0.02
	}

	if signalType == "COMPRA" {
		// Stop loss por debajo del precio de entrada
		return math.Round((entryPrice-atr*multiplier)*100) / 100
	}
	// Stop loss por encima del precio de entrada (posición corta)
	return math.Round((entryPrice+atr*multiplier)*100) / 100
}

// CalculateTakeProfit calcula el precio objetivo basado en el ratio riesgo/beneficio
// Con un ratio de 1:2, el beneficio potencial es el doble del riesgo
func CalculateTakeProfit(entryPrice, stopLoss, ratio float64) float64 {
	risk := math.Abs(entryPrice - stopLoss)

	if entryPrice > stopLoss {
		// Posición larga: take profit por encima
		return math.Round((entryPrice+risk*ratio)*100) / 100
	}
	// Posición corta: take profit por debajo
	return math.Round((entryPrice-risk*ratio)*100) / 100
}

// CalculatePositionSize calcula el tamaño de la posición
// Fórmula: Cantidad = (Capital × %Riesgo) / |Entrada - StopLoss|
// Esto asegura que si se ejecuta el stop loss, la pérdida máxima
// sea exactamente el porcentaje de riesgo definido
func CalculatePositionSize(capital, riskPercent, entryPrice, stopLoss float64) float64 {
	// Monto máximo que estamos dispuestos a perder
	riskAmount := capital * (riskPercent / 100)

	// Riesgo por unidad (distancia entre entrada y stop loss)
	riskPerUnit := math.Abs(entryPrice - stopLoss)

	if riskPerUnit == 0 {
		return 0
	}

	// Cantidad de unidades a comprar/vender
	quantity := riskAmount / riskPerUnit

	return math.Round(quantity*1000000) / 1000000
}
