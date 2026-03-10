// ============================================
// Bot Oscar - Panel de Señales IA (DeepSeek)
// Permite generar análisis inteligente con un botón
// y muestra el resultado detallado de DeepSeek
// ============================================
import {
  Brain,
  Sparkles,
  ArrowUpCircle,
  ArrowDownCircle,
  MinusCircle,
  AlertTriangle,
  Clock,
  Shield,
  Target,
  TrendingUp,
  TrendingDown,
  Loader2,
  CheckCircle,
  CandlestickChart,
} from 'lucide-react';
import type { AISignal, Asset, PatternItem } from '../../types';

interface AISignalPanelProps {
  selectedAsset: Asset | null;
  aiSignal: AISignal | null;
  loading: boolean;
  error: string | null;
  onGenerateSignal: () => void;
}

export default function AISignalPanel({
  selectedAsset,
  aiSignal,
  loading,
  error,
  onGenerateSignal,
}: AISignalPanelProps) {

  // Ícono según tipo de señal
  const getSignalIcon = (signal: string) => {
    switch (signal) {
      case 'COMPRA': return <ArrowUpCircle className="w-6 h-6 text-oscar-green" />;
      case 'VENTA': return <ArrowDownCircle className="w-6 h-6 text-oscar-red" />;
      default: return <MinusCircle className="w-6 h-6 text-oscar-gold" />;
    }
  };

  // Color de fondo según señal
  const getSignalBg = (signal: string) => {
    switch (signal) {
      case 'COMPRA': return 'border-oscar-green/40 bg-oscar-green/5';
      case 'VENTA': return 'border-oscar-red/40 bg-oscar-red/5';
      default: return 'border-oscar-gold/40 bg-oscar-gold/5';
    }
  };

  // Badge de señal
  const getSignalBadge = (signal: string) => {
    switch (signal) {
      case 'COMPRA': return 'badge-compra';
      case 'VENTA': return 'badge-venta';
      default: return 'badge-mantener';
    }
  };

  // Color del nivel de riesgo
  const getRiskColor = (risk: string) => {
    switch (risk) {
      case 'bajo': return 'text-oscar-green';
      case 'alto': return 'text-oscar-red';
      default: return 'text-oscar-gold';
    }
  };

  // Color de la barra de confianza
  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 70) return 'bg-oscar-green';
    if (confidence >= 40) return 'bg-oscar-gold';
    return 'bg-oscar-red';
  };

  return (
    <div className="glass-card p-5 animate-fade-in">
      {/* Encabezado */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Brain className="w-5 h-5 text-purple-400" />
          <h3 className="text-lg font-bold">Análisis IA</h3>
          <span className="text-[10px] bg-purple-500/20 text-purple-300 border border-purple-500/30 px-2 py-0.5 rounded-full font-mono">
            DeepSeek
          </span>
        </div>
      </div>

      {/* Botón de generar señal */}
      <button
        onClick={onGenerateSignal}
        disabled={loading || !selectedAsset}
        className={`w-full mb-4 flex items-center justify-center gap-2 py-3 px-4 rounded-lg font-bold text-sm transition-all duration-300 ${
          loading
            ? 'bg-purple-500/20 text-purple-300 cursor-wait border border-purple-500/30'
            : !selectedAsset
            ? 'bg-gray-800/50 text-gray-500 cursor-not-allowed border border-gray-700/30'
            : 'bg-gradient-to-r from-purple-600 to-purple-500 text-white hover:from-purple-500 hover:to-purple-400 hover:shadow-lg hover:shadow-purple-500/20 active:scale-[0.98]'
        }`}
      >
        {loading ? (
          <>
            <Loader2 className="w-4 h-4 animate-spin" />
            DeepSeek analizando {selectedAsset?.symbol}...
          </>
        ) : (
          <>
            <Sparkles className="w-4 h-4" />
            Generar Señal IA — {selectedAsset?.symbol ?? 'Selecciona un activo'}
          </>
        )}
      </button>

      {/* Error */}
      {error && (
        <div className="mb-4 p-3 bg-oscar-red/10 border border-oscar-red/30 rounded-lg flex items-start gap-2">
          <AlertTriangle className="w-4 h-4 text-oscar-red flex-shrink-0 mt-0.5" />
          <p className="text-xs text-oscar-red">{error}</p>
        </div>
      )}

      {/* Resultado de la señal IA */}
      {aiSignal && (
        <div className="space-y-4 animate-fade-in">
          {/* Card principal de señal */}
          <div className={`border rounded-xl p-4 ${getSignalBg(aiSignal.signal)}`}>
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-3">
                {getSignalIcon(aiSignal.signal)}
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-white font-bold text-lg">{aiSignal.assetName}</span>
                    <span className={getSignalBadge(aiSignal.signal)}>{aiSignal.signal}</span>
                  </div>
                  <span className="text-xs text-oscar-gray font-mono">{aiSignal.symbol}</span>
                </div>
              </div>
              <div className="text-right">
                <div className="flex items-center gap-1">
                  <CheckCircle className="w-3 h-3 text-purple-400" />
                  <span className="text-[10px] text-purple-300 font-mono">{aiSignal.model}</span>
                </div>
              </div>
            </div>

            {/* Barra de confianza */}
            <div className="mb-3">
              <div className="flex items-center justify-between text-xs mb-1">
                <span className="text-oscar-gray">Confianza IA</span>
                <span className="font-bold text-white">{aiSignal.confidence}%</span>
              </div>
              <div className="w-full h-2 bg-gray-800 rounded-full overflow-hidden">
                <div
                  className={`h-full rounded-full transition-all duration-1000 ${getConfidenceColor(aiSignal.confidence)}`}
                  style={{ width: `${aiSignal.confidence}%` }}
                />
              </div>
            </div>

            {/* Precios: Entrada, SL, TP */}
            <div className="grid grid-cols-3 gap-3">
              <div className="bg-oscar-dark/60 rounded-lg p-2.5 text-center">
                <Target className="w-3.5 h-3.5 text-oscar-gold mx-auto mb-1" />
                <span className="text-[10px] text-oscar-gray block">Entrada</span>
                <span className="text-sm text-white font-mono font-bold">
                  ${aiSignal.entryPrice > 0 ? aiSignal.entryPrice.toFixed(2) : '—'}
                </span>
              </div>
              <div className="bg-oscar-dark/60 rounded-lg p-2.5 text-center">
                <TrendingDown className="w-3.5 h-3.5 text-oscar-red mx-auto mb-1" />
                <span className="text-[10px] text-oscar-gray block">Stop Loss</span>
                <span className="text-sm text-oscar-red font-mono font-bold">
                  ${aiSignal.stopLoss > 0 ? aiSignal.stopLoss.toFixed(2) : '—'}
                </span>
              </div>
              <div className="bg-oscar-dark/60 rounded-lg p-2.5 text-center">
                <TrendingUp className="w-3.5 h-3.5 text-oscar-green mx-auto mb-1" />
                <span className="text-[10px] text-oscar-gray block">Take Profit</span>
                <span className="text-sm text-oscar-green font-mono font-bold">
                  ${aiSignal.takeProfit > 0 ? aiSignal.takeProfit.toFixed(2) : '—'}
                </span>
              </div>
            </div>
          </div>

          {/* Meta info: Timeframe + Riesgo */}
          <div className="grid grid-cols-2 gap-3">
            <div className="bg-oscar-dark/40 border border-gray-800/50 rounded-lg p-3 flex items-center gap-2">
              <Clock className="w-4 h-4 text-oscar-gold" />
              <div>
                <span className="text-[10px] text-oscar-gray block">Plazo</span>
                <span className="text-sm text-white font-bold capitalize">{aiSignal.timeframe}</span>
              </div>
            </div>
            <div className="bg-oscar-dark/40 border border-gray-800/50 rounded-lg p-3 flex items-center gap-2">
              <Shield className="w-4 h-4 text-oscar-gold" />
              <div>
                <span className="text-[10px] text-oscar-gray block">Riesgo</span>
                <span className={`text-sm font-bold capitalize ${getRiskColor(aiSignal.riskLevel)}`}>
                  {aiSignal.riskLevel}
                </span>
              </div>
            </div>
          </div>

          {/* Análisis detallado */}
          <div className="bg-oscar-dark/40 border border-gray-800/50 rounded-lg p-4">
            <h4 className="text-sm font-bold text-white mb-2 flex items-center gap-1.5">
              <Brain className="w-3.5 h-3.5 text-purple-400" />
              Análisis DeepSeek
            </h4>
            <p className="text-xs text-oscar-gray leading-relaxed whitespace-pre-line">
              {aiSignal.analysis}
            </p>
          </div>

          {/* ═══════════════════════════════════════════════════ */}
          {/* Sección de Patrones de Velas Japonesas             */}
          {/* ═══════════════════════════════════════════════════ */}
          {aiSignal.patterns && aiSignal.patterns.detected.length > 0 && (
            <div className="bg-oscar-dark/40 border border-gray-800/50 rounded-lg p-4 animate-fade-in">
              {/* Cabecera de patrones */}
              <div className="flex items-center justify-between mb-3">
                <h4 className="text-sm font-bold text-white flex items-center gap-1.5">
                  <CandlestickChart className="w-3.5 h-3.5 text-orange-400" />
                  Patrones de Velas Detectados
                </h4>
                <div className={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${
                  aiSignal.patterns.bias === 'ALCISTA'
                    ? 'bg-oscar-green/10 text-oscar-green border-oscar-green/30'
                    : aiSignal.patterns.bias === 'BAJISTA'
                    ? 'bg-oscar-red/10 text-oscar-red border-oscar-red/30'
                    : 'bg-oscar-gold/10 text-oscar-gold border-oscar-gold/30'
                }`}>
                  Sesgo {aiSignal.patterns.bias} — {aiSignal.patterns.biasStrength}/100
                </div>
              </div>

              {/* Barra visual de sesgo */}
              <div className="mb-3">
                <div className="flex items-center justify-between text-[10px] mb-1">
                  <span className="text-oscar-red">Bajista ({aiSignal.patterns.bearishCount})</span>
                  <span className="text-oscar-gray">Neutral ({aiSignal.patterns.neutralCount})</span>
                  <span className="text-oscar-green">Alcista ({aiSignal.patterns.bullishCount})</span>
                </div>
                <div className="w-full h-1.5 bg-gray-800 rounded-full overflow-hidden flex">
                  {(() => {
                    const total = aiSignal.patterns.bullishCount + aiSignal.patterns.bearishCount + aiSignal.patterns.neutralCount;
                    if (total === 0) return null;
                    const bullPct = (aiSignal.patterns.bullishCount / total) * 100;
                    const bearPct = (aiSignal.patterns.bearishCount / total) * 100;
                    const neutPct = (aiSignal.patterns.neutralCount / total) * 100;
                    return (
                      <>
                        {bearPct > 0 && <div className="h-full bg-oscar-red" style={{ width: `${bearPct}%` }} />}
                        {neutPct > 0 && <div className="h-full bg-oscar-gold/50" style={{ width: `${neutPct}%` }} />}
                        {bullPct > 0 && <div className="h-full bg-oscar-green" style={{ width: `${bullPct}%` }} />}
                      </>
                    );
                  })()}
                </div>
              </div>

              {/* Patrones agrupados por timeframe */}
              {['1day', '4h', '1h'].map(tf => {
                const tfPatterns = aiSignal.patterns!.detected.filter((p: PatternItem) => p.timeframe === tf);
                if (tfPatterns.length === 0) return null;
                const tfLabel = tf === '1day' ? 'Diario' : tf === '4h' ? '4 Horas' : '1 Hora';
                const tfBias = aiSignal.patterns!.byTimeframe[tf];
                return (
                  <div key={tf} className="mb-2 last:mb-0">
                    <div className="flex items-center gap-2 mb-1.5">
                      <span className="text-[10px] font-mono font-bold text-oscar-gold bg-oscar-gold/10 px-1.5 py-0.5 rounded">
                        {tfLabel}
                      </span>
                      {tfBias && tfBias !== 'NEUTRAL' && (
                        <span className={`text-[9px] ${tfBias === 'ALCISTA' ? 'text-oscar-green' : 'text-oscar-red'}`}>
                          {tfBias === 'ALCISTA' ? '↑' : '↓'} {tfBias}
                        </span>
                      )}
                    </div>
                    <div className="flex flex-wrap gap-1.5">
                      {tfPatterns.map((p: PatternItem, idx: number) => (
                        <div
                          key={`${tf}-${idx}`}
                          className={`group relative text-[10px] px-2 py-1 rounded-md border cursor-default transition-all ${
                            p.type === 'ALCISTA'
                              ? 'bg-oscar-green/8 border-oscar-green/25 text-oscar-green hover:bg-oscar-green/15'
                              : p.type === 'BAJISTA'
                              ? 'bg-oscar-red/8 border-oscar-red/25 text-oscar-red hover:bg-oscar-red/15'
                              : 'bg-oscar-gold/8 border-oscar-gold/25 text-oscar-gold hover:bg-oscar-gold/15'
                          }`}
                        >
                          <span className="font-medium">{p.name}</span>
                          <span className="ml-1 opacity-70">
                            {'★'.repeat(p.strength)}{'☆'.repeat(3 - p.strength)}
                          </span>
                          {/* Tooltip con detalles al hacer hover */}
                          <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 px-2 py-1 bg-gray-900 border border-gray-700 rounded text-[9px] text-oscar-gray whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-10">
                            {p.details}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                );
              })}

              {/* Confluencias multi-timeframe */}
              {aiSignal.patterns.confluences.length > 0 && (
                <div className="mt-3 pt-2 border-t border-gray-800/50">
                  <span className="text-[10px] font-bold text-white">Confluencias:</span>
                  {aiSignal.patterns.confluences.map((c, i) => (
                    <div key={i} className="text-[10px] text-oscar-gold mt-0.5 flex items-center gap-1">
                      <span>⚡</span>
                      <span>{c}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Factores clave */}
          {aiSignal.keyFactors && aiSignal.keyFactors.length > 0 && (
            <div className="bg-oscar-dark/40 border border-gray-800/50 rounded-lg p-4">
              <h4 className="text-sm font-bold text-white mb-2">📌 Factores Clave</h4>
              <ul className="space-y-1.5">
                {aiSignal.keyFactors.map((factor, i) => (
                  <li key={i} className="text-xs text-oscar-gray flex items-start gap-2">
                    <span className="text-oscar-gold mt-0.5">•</span>
                    {factor}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Disclaimer */}
          <div className="bg-yellow-500/5 border border-yellow-500/20 rounded-lg p-3 flex items-start gap-2">
            <AlertTriangle className="w-4 h-4 text-yellow-500 flex-shrink-0 mt-0.5" />
            <p className="text-[10px] text-yellow-500/80 leading-relaxed">
              {aiSignal.disclaimer}
            </p>
          </div>

          {/* Timestamp */}
          <div className="text-center">
            <span className="text-[10px] text-oscar-gray">
              Generado: {new Date(aiSignal.timestamp).toLocaleString('es-ES')}
            </span>
          </div>
        </div>
      )}

      {/* Estado vacío */}
      {!aiSignal && !loading && !error && (
        <div className="text-center py-6 text-oscar-gray">
          <Brain className="w-10 h-10 mx-auto mb-3 text-purple-400/30" />
          <p className="text-sm mb-1">Análisis inteligente con IA</p>
          <p className="text-[11px] text-oscar-gray/60 max-w-[280px] mx-auto">
            Presiona el botón para que DeepSeek analice los indicadores técnicos reales de{' '}
            <span className="text-oscar-gold">{selectedAsset?.symbol ?? 'un activo'}</span>{' '}
            y genere una señal profesional
          </p>
        </div>
      )}
    </div>
  );
}
