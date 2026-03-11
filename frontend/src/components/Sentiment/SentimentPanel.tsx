// ============================================
// Bot Oscar - Panel de Sentimiento de Mercado
// Muestra Analyst Ratings (Buy/Sell) y Short Interest
// para el activo seleccionado, datos reales de Yahoo Finance
// ============================================
import { useState, useEffect } from 'react';
import type { Asset, MarketSentiment } from '../../types';
import * as api from '../../services/api';

interface SentimentPanelProps {
  selectedAsset: Asset | null;
}

export default function SentimentPanel({ selectedAsset }: SentimentPanelProps) {
  const [data, setData] = useState<MarketSentiment | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!selectedAsset) {
      setData(null);
      return;
    }

    const fetchSentiment = async () => {
      setLoading(true);
      setError(null);
      try {
        const result = await api.getSentiment(selectedAsset.symbol);
        setData(result);
      } catch (err) {
        console.error('Error obteniendo sentimiento:', err);
        setError('Error obteniendo datos de sentimiento');
        setData(null);
      } finally {
        setLoading(false);
      }
    };

    fetchSentiment();

    // Refrescar cada 5 minutos (los datos no cambian tan rápido)
    const interval = setInterval(fetchSentiment, 300000);
    return () => clearInterval(interval);
  }, [selectedAsset?.symbol]);

  if (!selectedAsset) return null;

  return (
    <div className="glass-card p-3 animate-fade-in">
      {/* Header */}
      <div className="flex items-center gap-2 mb-3">
        <span className="text-sm">📊</span>
        <h3 className="text-xs font-bold text-white">Sentimiento de Mercado</h3>
        <span className="text-[10px] text-oscar-gray">• {selectedAsset.symbol} • Yahoo Finance</span>
      </div>

      {loading && !data && (
        <div className="flex items-center justify-center py-6">
          <div className="animate-spin rounded-full h-6 w-6 border-2 border-oscar-gold border-t-transparent" />
        </div>
      )}

      {error && !data && (
        <p className="text-red-400 text-xs text-center py-3">{error}</p>
      )}

      {data && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Columna 1: Analyst Ratings */}
          <AnalystRatingsCard ratings={data.analystRatings} />

          {/* Columna 2: Short Interest */}
          <ShortInterestCard shortInterest={data.shortInterest} />
        </div>
      )}

      {/* Resumen */}
      {data?.summary && (
        <p className="text-[9px] text-oscar-gray/60 mt-3 leading-relaxed">
          {data.summary}
        </p>
      )}
    </div>
  );
}

// ---- Tarjeta de Analyst Ratings ----
function AnalystRatingsCard({ ratings }: { ratings: MarketSentiment['analystRatings'] }) {
  if (!ratings || ratings.total === 0) {
    return (
      <div className="bg-oscar-dark/40 rounded-lg p-3 border border-oscar-gray/10">
        <p className="text-[10px] font-semibold text-oscar-gray mb-2">📈 Recomendaciones de Analistas</p>
        <p className="text-xs text-oscar-gray/50 text-center py-2">Sin datos disponibles para este activo</p>
      </div>
    );
  }

  const consensusColor = getConsensusColor(ratings.consensus);
  const upsideColor = ratings.upsidePercent >= 0 ? 'text-green-400' : 'text-red-400';
  const upsideIcon = ratings.upsidePercent >= 0 ? '▲' : '▼';

  return (
    <div className="bg-oscar-dark/40 rounded-lg p-3 border border-oscar-gray/10">
      {/* Título + Consenso */}
      <div className="flex items-center justify-between mb-3">
        <p className="text-[10px] font-semibold text-oscar-gray">📈 Recomendaciones de Analistas</p>
        <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${consensusColor}`}>
          {ratings.consensus}
        </span>
      </div>

      {/* Barra de distribución Buy/Hold/Sell */}
      <div className="mb-3">
        <RatingsBar ratings={ratings} />
      </div>

      {/* Desglose numérico */}
      <div className="grid grid-cols-5 gap-1 mb-3">
        <RatingCell label="Compra+" count={ratings.strongBuy} color="text-green-400" />
        <RatingCell label="Compra" count={ratings.buy} color="text-green-300" />
        <RatingCell label="Mantener" count={ratings.hold} color="text-yellow-400" />
        <RatingCell label="Venta" count={ratings.sell} color="text-red-300" />
        <RatingCell label="Venta+" count={ratings.strongSell} color="text-red-400" />
      </div>

      {/* Precio objetivo */}
      {ratings.targetMean > 0 && (
        <div className="bg-oscar-black/30 rounded-md p-2">
          <div className="flex items-center justify-between">
            <span className="text-[9px] text-oscar-gray">Precio objetivo medio:</span>
            <span className="text-xs font-mono font-bold text-oscar-gold">
              ${ratings.targetMean.toFixed(2)}
            </span>
          </div>
          <div className="flex items-center justify-between mt-1">
            <span className="text-[9px] text-oscar-gray">
              Rango: ${ratings.targetLow.toFixed(2)} — ${ratings.targetHigh.toFixed(2)}
            </span>
            <span className={`text-[10px] font-bold ${upsideColor}`}>
              {upsideIcon} {Math.abs(ratings.upsidePercent).toFixed(1)}%
            </span>
          </div>
        </div>
      )}
    </div>
  );
}

// ---- Barra horizontal de distribución ----
function RatingsBar({ ratings }: { ratings: NonNullable<MarketSentiment['analystRatings']> }) {
  const total = ratings.total;
  if (total === 0) return null;

  const segments = [
    { count: ratings.strongBuy, color: 'bg-green-500', label: 'Compra+' },
    { count: ratings.buy, color: 'bg-green-400', label: 'Compra' },
    { count: ratings.hold, color: 'bg-yellow-500', label: 'Mantener' },
    { count: ratings.sell, color: 'bg-red-400', label: 'Venta' },
    { count: ratings.strongSell, color: 'bg-red-500', label: 'Venta+' },
  ];

  return (
    <div>
      {/* Barra */}
      <div className="flex h-3 rounded-full overflow-hidden gap-[1px]">
        {segments.map((seg, i) => {
          const pct = (seg.count / total) * 100;
          if (pct === 0) return null;
          return (
            <div
              key={i}
              className={`${seg.color} transition-all duration-500`}
              style={{ width: `${pct}%` }}
              title={`${seg.label}: ${seg.count} (${pct.toFixed(0)}%)`}
            />
          );
        })}
      </div>
      {/* Porcentajes debajo */}
      <div className="flex justify-between mt-1">
        <span className="text-[9px] text-green-400 font-bold">{ratings.buyPercent}% Compra</span>
        <span className="text-[9px] text-oscar-gray">{ratings.total} analistas</span>
        <span className="text-[9px] text-red-400 font-bold">{ratings.sellPercent}% Venta</span>
      </div>
    </div>
  );
}

// ---- Celda individual de rating ----
function RatingCell({ label, count, color }: { label: string; count: number; color: string }) {
  return (
    <div className="text-center">
      <div className={`text-sm font-mono font-bold ${color}`}>{count}</div>
      <div className="text-[8px] text-oscar-gray/60">{label}</div>
    </div>
  );
}

// ---- Tarjeta de Short Interest ----
function ShortInterestCard({ shortInterest }: { shortInterest: MarketSentiment['shortInterest'] }) {
  if (!shortInterest) {
    return (
      <div className="bg-oscar-dark/40 rounded-lg p-3 border border-oscar-gray/10">
        <p className="text-[10px] font-semibold text-oscar-gray mb-2">🩳 Short Interest</p>
        <p className="text-xs text-oscar-gray/50 text-center py-2">Sin datos disponibles para este activo</p>
      </div>
    );
  }

  const levelColor = getShortLevelColor(shortInterest.level);
  const levelEmoji = getShortLevelEmoji(shortInterest.level);

  return (
    <div className="bg-oscar-dark/40 rounded-lg p-3 border border-oscar-gray/10">
      {/* Título + Nivel */}
      <div className="flex items-center justify-between mb-3">
        <p className="text-[10px] font-semibold text-oscar-gray">🩳 Short Interest</p>
        <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${levelColor}`}>
          {levelEmoji} {shortInterest.level}
        </span>
      </div>

      {/* Gauge visual del short % */}
      <div className="mb-3">
        <ShortGauge percent={shortInterest.shortPercentOfFloat} />
      </div>

      {/* Detalles */}
      <div className="space-y-1.5">
        <ShortDetailRow label="% del Float en corto" value={`${shortInterest.shortPercentOfFloat.toFixed(2)}%`} />
        <ShortDetailRow label="Días para cubrir" value={`${shortInterest.shortRatio.toFixed(1)} días`} />
        {shortInterest.sharesShort > 0 && (
          <ShortDetailRow label="Acciones en corto" value={formatNumber(shortInterest.sharesShort)} />
        )}
        {shortInterest.sharesFloat > 0 && (
          <ShortDetailRow label="Float total" value={formatNumber(shortInterest.sharesFloat)} />
        )}
      </div>

      {/* Explicación */}
      <p className="text-[8px] text-oscar-gray/40 mt-2 leading-snug">
        {shortInterest.shortPercentOfFloat >= 10
          ? '⚠️ Short interest alto — posible short squeeze si el precio sube'
          : shortInterest.shortPercentOfFloat >= 5
          ? 'Nivel moderado de posiciones en corto'
          : 'Nivel bajo de posiciones en corto — sentimiento generalmente positivo'}
      </p>
    </div>
  );
}

// ---- Gauge visual del short interest ----
function ShortGauge({ percent }: { percent: number }) {
  // Cap at 30% for visual purposes
  const cappedPct = Math.min(percent, 30);
  const width = (cappedPct / 30) * 100;

  const getColor = (p: number) => {
    if (p >= 20) return 'bg-red-500';
    if (p >= 10) return 'bg-orange-500';
    if (p >= 5) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  return (
    <div>
      <div className="flex h-2.5 bg-oscar-black/40 rounded-full overflow-hidden">
        <div
          className={`${getColor(percent)} transition-all duration-700 rounded-full`}
          style={{ width: `${width}%` }}
        />
      </div>
      <div className="flex justify-between mt-0.5">
        <span className="text-[8px] text-oscar-gray/50">0%</span>
        <span className="text-[8px] text-oscar-gray/50">30%+</span>
      </div>
    </div>
  );
}

// ---- Fila de detalle ----
function ShortDetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between bg-oscar-black/20 rounded px-2 py-1">
      <span className="text-[9px] text-oscar-gray">{label}</span>
      <span className="text-[10px] font-mono font-bold text-white">{value}</span>
    </div>
  );
}

// ============================================
// Helpers
// ============================================

function getConsensusColor(consensus: string): string {
  switch (consensus) {
    case 'Compra Fuerte': return 'bg-green-500/20 text-green-400 border border-green-500/30';
    case 'Compra': return 'bg-green-500/10 text-green-300 border border-green-500/20';
    case 'Mantener': return 'bg-yellow-500/10 text-yellow-400 border border-yellow-500/20';
    case 'Venta': return 'bg-red-500/10 text-red-300 border border-red-500/20';
    case 'Venta Fuerte': return 'bg-red-500/20 text-red-400 border border-red-500/30';
    default: return 'bg-oscar-gray/10 text-oscar-gray border border-oscar-gray/20';
  }
}

function getShortLevelColor(level: string): string {
  switch (level) {
    case 'Muy Alto': return 'bg-red-500/20 text-red-400 border border-red-500/30';
    case 'Alto': return 'bg-orange-500/20 text-orange-400 border border-orange-500/30';
    case 'Moderado': return 'bg-yellow-500/15 text-yellow-400 border border-yellow-500/25';
    case 'Bajo': return 'bg-green-500/15 text-green-400 border border-green-500/25';
    default: return 'bg-oscar-gray/10 text-oscar-gray border border-oscar-gray/20';
  }
}

function getShortLevelEmoji(level: string): string {
  switch (level) {
    case 'Muy Alto': return '🔴';
    case 'Alto': return '🟠';
    case 'Moderado': return '🟡';
    case 'Bajo': return '🟢';
    default: return '⚪';
  }
}

function formatNumber(n: number): string {
  if (n >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(2)}B`;
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return n.toString();
}
