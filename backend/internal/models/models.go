// ============================================
// Bot Oscar - Modelos de datos
// Estructuras que representan las entidades del sistema
// ============================================
package models

import "time"

// Asset representa un activo financiero (commodity o acción de defensa)
type Asset struct {
	ID        int       `json:"id"`
	Symbol    string    `json:"symbol"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // "commodity" o "accion"
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
}

// Price representa una vela OHLCV (Open, High, Low, Close, Volume)
type Price struct {
	ID      int       `json:"id"`
	AssetID int       `json:"assetId"`
	Open    float64   `json:"open"`
	High    float64   `json:"high"`
	Low     float64   `json:"low"`
	Close   float64   `json:"close"`
	Volume  int64     `json:"volume"`
	Date    time.Time `json:"date"`
}

// Signal representa una señal de trading generada por el análisis
type Signal struct {
	ID         int       `json:"id"`
	AssetID    int       `json:"assetId"`
	Symbol     string    `json:"symbol"`
	AssetName  string    `json:"assetName"`
	Type       string    `json:"type"`     // "COMPRA", "VENTA", "MANTENER"
	Strength   int       `json:"strength"` // Puntuación de 0 a 100
	EntryPrice float64   `json:"entryPrice"`
	StopLoss   float64   `json:"stopLoss"`
	TakeProfit float64   `json:"takeProfit"`
	Reason     string    `json:"reason"` // Explicación legible de la señal
	CreatedAt  time.Time `json:"createdAt"`
}

// Operation representa una operación (trade) ejecutada
type Operation struct {
	ID          int        `json:"id"`
	AssetID     int        `json:"assetId"`
	Symbol      string     `json:"symbol"`
	AssetName   string     `json:"assetName"`
	Type        string     `json:"type"` // "COMPRA" o "VENTA"
	EntryPrice  float64    `json:"entryPrice"`
	ExitPrice   *float64   `json:"exitPrice"` // nil si aún está abierta
	Quantity    float64    `json:"quantity"`
	StopLoss    float64    `json:"stopLoss"`
	TakeProfit  float64    `json:"takeProfit"`
	Status      string     `json:"status"` // "ABIERTA", "CERRADA", "CANCELADA"
	PnL         *float64   `json:"pnl"`    // Ganancia/Pérdida, nil si abierta
	EntryReason string     `json:"entryReason"`
	ExitReason  *string    `json:"exitReason"`
	OpenedAt    time.Time  `json:"openedAt"`
	ClosedAt    *time.Time `json:"closedAt"`
}

// Portfolio representa el estado general del portafolio
type Portfolio struct {
	Capital             float64     `json:"capital"`
	InitialCapital      float64     `json:"capitalInicial"`
	PnL                 float64     `json:"gananciaPerdida"`
	ReturnPercent       float64     `json:"porcentajeRetorno"`
	OpenOperationsCount int         `json:"operacionesAbiertas"`
	TotalOperations     int         `json:"totalOperaciones"`
	Operations          []Operation `json:"operaciones"`
}

// BotStatus representa el estado actual del bot
type BotStatus struct {
	Running         bool       `json:"running"`
	Mode            string     `json:"mode"` // "paper" o "real"
	LastAnalysis    *time.Time `json:"lastAnalysis"`
	AssetsMonitored int        `json:"assetsMonitored"`
	ActiveSignals   int        `json:"activeSignals"`
}

// IndicatorValues contiene los valores calculados de todos los indicadores
// Un trader profesional necesita TODOS estos datos para tomar decisiones
type IndicatorValues struct {
	Symbol string `json:"symbol"`

	// === OSCILADORES ===
	RSI  float64    `json:"rsi"`  // RSI(14) - Fuerza relativa
	MACD MACDValues `json:"macd"` // MACD(12,26,9) - Momentum

	// === MEDIAS MÓVILES SIMPLES ===
	SMA50  float64 `json:"sma50"`  // Tendencia intermedia
	SMA200 float64 `json:"sma200"` // Tendencia principal

	// === MEDIAS MÓVILES EXPONENCIALES ===
	EMA12  float64 `json:"ema12"`  // Cruce rápido (componente MACD)
	EMA26  float64 `json:"ema26"`  // Cruce lento (componente MACD)
	EMA21  float64 `json:"ema21"`  // Pullbacks - muy usada por institucionales
	EMA50  float64 `json:"ema50"`  // Tendencia intermedia (más reactiva que SMA50)
	EMA200 float64 `json:"ema200"` // Tendencia principal (la más importante)

	// === VOLATILIDAD ===
	Bollinger BollingerValues `json:"bollinger"` // Bandas de Bollinger(20,2)
	ATR       float64         `json:"atr"`       // Average True Range(14) - Volatilidad real

	// === VOLUMEN ===
	VWAP         float64 `json:"vwap"`         // Precio promedio ponderado por volumen
	VolumenProm  int64   `json:"volumenProm"`  // Volumen promedio de 20 días
	VolumenHoy   int64   `json:"volumenHoy"`   // Volumen del último día
	VolumenRatio float64 `json:"volumenRatio"` // Ratio volumen hoy / promedio (>1.5 = pico)

	// === RESULTADO ===
	Score  int    `json:"score"`  // Puntuación de confluencia 0-100
	Signal string `json:"signal"` // "COMPRA", "VENTA", "MANTENER"
}

// MACDValues contiene las 3 líneas del MACD
type MACDValues struct {
	MACD      float64 `json:"macd"`
	Signal    float64 `json:"signal"`
	Histogram float64 `json:"histogram"`
}

// BollingerValues contiene las 3 bandas de Bollinger
type BollingerValues struct {
	Upper  float64 `json:"upper"`
	Middle float64 `json:"middle"`
	Lower  float64 `json:"lower"`
}
