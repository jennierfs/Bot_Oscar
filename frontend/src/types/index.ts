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

// Valores calculados de indicadores
export interface IndicatorValues {
  symbol: string;
  rsi: number;
  macd: {
    macd: number;
    signal: number;
    histogram: number;
  };
  sma50: number;
  sma200: number;
  ema12: number;
  ema26: number;
  bollinger: {
    upper: number;
    middle: number;
    lower: number;
  };
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
}
