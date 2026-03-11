// ============================================
// Bot Oscar - Tipos TypeScript
// Interfaces que reflejan exactamente los modelos del backend Go
// ============================================

// Activo financiero (commodity o acción de defensa)
export interface Asset {
  id: number;
  symbol: string;
  name: string;
  type: 'commodity' | 'accion';
  active: boolean;
  createdAt: string;
}

// Precio OHLCV (vela japonesa)
export interface Price {
  id: number;
  assetId: number;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
  date: string;
}

// Señal de trading generada por el motor de análisis
export interface Signal {
  id: number;
  assetId: number;
  symbol: string;
  assetName: string;
  type: 'COMPRA' | 'VENTA' | 'MANTENER';
  strength: number; // 0-100
  entryPrice: number;
  stopLoss: number;
  takeProfit: number;
  reason: string;
  createdAt: string;
}

// Operación (trade) ejecutada
export interface Operation {
  id: number;
  assetId: number;
  symbol: string;
  assetName: string;
  type: 'COMPRA' | 'VENTA';
  entryPrice: number;
  exitPrice: number | null;
  quantity: number;
  stopLoss: number;
  takeProfit: number;
  status: 'ABIERTA' | 'CERRADA' | 'CANCELADA';
  pnl: number | null;
  entryReason: string;
  exitReason: string | null;
  openedAt: string;
  closedAt: string | null;
}

// Resumen del portafolio
export interface Portfolio {
  capital: number;
  capitalInicial: number;
  gananciaPerdida: number;
  porcentajeRetorno: number;
  operacionesAbiertas: number;
  totalOperaciones: number;
  operaciones: Operation[];
}

// Estado del bot
export interface BotStatus {
  running: boolean;
  mode: string;
  lastAnalysis: string | null;
  assetsMonitored: number;
  activeSignals: number;
}

// Valores calculados de indicadores técnicos profesionales
export interface IndicatorValues {
  symbol: string;

  // Osciladores
  rsi: number;
  macd: {
    macd: number;
    signal: number;
    histogram: number;
  };

  // Medias Móviles Simples
  sma50: number;
  sma200: number;

  // Medias Móviles Exponenciales
  ema12: number;
  ema26: number;
  ema21: number;   // Pullbacks institucionales
  ema50: number;   // Tendencia intermedia
  ema200: number;  // Tendencia principal (la más importante)

  // Volatilidad
  bollinger: {
    upper: number;
    middle: number;
    lower: number;
  };
  atr: number;     // Average True Range - volatilidad real

  // Volumen
  vwap: number;         // Precio promedio ponderado por volumen
  volumenProm: number;  // Volumen promedio 20 días
  volumenHoy: number;   // Volumen del último día
  volumenRatio: number; // Ratio volumen hoy / promedio

  // Resultado
  score: number;
  signal: string;
}

// Señal generada por DeepSeek AI
export interface AISignal {
  symbol: string;
  assetName: string;
  signal: 'COMPRA' | 'VENTA' | 'MANTENER';
  confidence: number;     // 0-100
  entryPrice: number;
  stopLoss: number;
  takeProfit: number;
  timeframe: string;      // "corto", "medio", "largo"
  riskLevel: string;      // "bajo", "medio", "alto"
  analysis: string;       // Análisis detallado en español
  keyFactors: string[];   // Factores clave de la decisión
  disclaimer: string;     // Advertencia legal
  timestamp: string;
  model: string;          // Modelo de IA usado
  patterns?: PatternData; // Patrones de velas detectados
}

// Datos de patrones de velas japonesas detectados
export interface PatternData {
  detected: PatternItem[];     // Lista de patrones encontrados
  bullishCount: number;        // Total alcistas
  bearishCount: number;        // Total bajistas
  neutralCount: number;        // Total neutrales
  bias: string;                // ALCISTA/BAJISTA/NEUTRAL
  biasStrength: number;        // 0-100
  byTimeframe: Record<string, string>;  // Sesgo por timeframe
  confluences: string[];       // Confluencias multi-timeframe
}

// Un patrón individual detectado
export interface PatternItem {
  name: string;       // Nombre en español
  nameEN: string;     // Nombre en inglés
  type: string;       // ALCISTA/BAJISTA/NEUTRAL
  strength: number;   // 1-3 (estrellas)
  timeframe: string;  // 1day, 4h, 1h
  details: string;    // Descripción del patrón
}

// Índice de Miedo y Codicia por activo individual
export interface FearGreedResult {
  symbol: string;
  assetName: string;
  score: number;       // 0-100
  label: string;       // "Miedo Extremo", "Miedo", "Neutral", "Codicia", "Codicia Extrema"
  description: string; // Explicación en español
  components: FearGreedComponent[];
}

// Componente individual del índice Fear & Greed
export interface FearGreedComponent {
  name: string;   // Nombre del factor
  score: number;  // 0-100
  weight: number; // 0-1
  detail: string; // Explicación
}

// ============================================
// Sentimiento de Mercado (Analyst Ratings + Short Interest)
// ============================================

// Sentimiento de mercado completo de un activo
export interface MarketSentiment {
  symbol: string;
  assetName: string;
  analystRatings: AnalystRatings | null;
  shortInterest: ShortInterest | null;
  summary: string;
  updatedAt: string;
}

// Recomendaciones de analistas de Wall Street
export interface AnalystRatings {
  strongBuy: number;
  buy: number;
  hold: number;
  sell: number;
  strongSell: number;
  total: number;
  buyPercent: number;   // % compra
  sellPercent: number;  // % venta
  consensus: string;    // "Compra Fuerte", "Compra", "Mantener", "Venta", "Venta Fuerte"
  targetHigh: number;
  targetLow: number;
  targetMean: number;
  currentPrice: number;
  upsidePercent: number; // % potencial subida/bajada
}

// Datos de posiciones en corto
export interface ShortInterest {
  shortPercentOfFloat: number;
  shortRatio: number;
  sharesShort: number;
  sharesFloat: number;
  level: string; // "Bajo", "Moderado", "Alto", "Muy Alto"
}
