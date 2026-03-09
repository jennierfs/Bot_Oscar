// ============================================
// Bot Oscar - Panel de Señales IA (DeepSeek)
// Permite generar análisis inteligente con un botón
// y muestra el resultado detallado de DeepSeek
// ============================================
import { useState } from 'react';
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
} from 'lucide-react';
import type { AISignal, Asset } from '../../types';
import * as api from '../../services/api';

interface AISignalPanelProps {
  selectedAsset: Asset | null;
}

export default function AISignalPanel({ selectedAsset }: AISignalPanelProps) {
  const [aiSignal, setAiSignal] = useState<AISignal | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Generar señal IA al presionar el botón
  const handleGenerateSignal = async () => {
    if (!selectedAsset) return;

    setLoading(true);
    setError(null);
    setAiSignal(null);

    try {
      const signal = await api.generateAISignal(selectedAsset.symbol);
      setAiSignal(signal);
    } catch (err: unknown) {
      const errorMsg =
        err instanceof Error
          ? err.message
          : 'Error desconocido generando señal IA';
      // Extraer mensaje del backend si existe
      const axiosErr = err as { response?: { data?: { error?: string } } };
      setError(axiosErr.response?.data?.error || errorMsg);
    } finally {
      setLoading(false);
    }
  };

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
        onClick={handleGenerateSignal}
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
