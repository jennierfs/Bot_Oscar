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
    const interval = setInterval(fetchSentiment, 300000);
    return () => clearInterval(interval);
  }, [selectedAsset?.symbol]);

  if (!selectedAsset) return null;

  return (
    <div className="glass-card p-4 animate-fade-in">
      {/* Header */}
      <div className="flex items-center gap-2 mb-4">
        <span className="text-base">📊</span>
        <h3 className="text-sm font-bold text-white">Sentimiento de Mercado</h3>
        <span className="text-[10px] text-oscar-gray">• {selectedAsset.symbol} • Yahoo Finance</span>
      </div>

      {loading && !data && (
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-2 border-oscar-gold border-t-transparent" />
        </div>
      )}

      {error && !data && (
        <p className="text-red-400 text-xs text-center py-4">{error}</p>
      )}

      {data && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
          <AnalystRatingsCard ratings={data.analystRatings} />
          <ShortInterestCard shortInterest={data.shortInterest} />
        </div>
      )}

      {data?.summary && (
        <p className="text-[9px] text-oscar-gray/50 mt-4 leading-relaxed border-t border-white/5 pt-3">
          {data.summary}
        </p>
      )}
    </div>
  );
}

// ============================================
// Analyst Ratings — Gauge semicircular + histograma + precio objetivo
// ============================================
function AnalystRatingsCard({ ratings }: { ratings: MarketSentiment['analystRatings'] }) {
  if (!ratings || ratings.total === 0) {
    return (
      <div className="bg-oscar-dark/40 rounded-xl p-4 border border-oscar-gray/10">
        <p className="text-[11px] font-semibold text-oscar-gray mb-2">📈 Recomendaciones de Analistas</p>
        <p className="text-xs text-oscar-gray/50 text-center py-4">Sin datos disponibles</p>
      </div>
    );
  }

  return (
    <div className="bg-oscar-dark/40 rounded-xl p-4 border border-oscar-gray/10">
      {/* Título */}
      <p className="text-[11px] font-semibold text-oscar-gray mb-3">📈 Recomendaciones de Analistas</p>

      {/* Gauge semicircular de consenso */}
      <div className="flex justify-center mb-3">
        <ConsensusGauge buyPercent={ratings.buyPercent} consensus={ratings.consensus} total={ratings.total} />
      </div>

      {/* Histograma de barras verticales */}
      <div className="mb-4">
        <RatingsHistogram ratings={ratings} />
      </div>

      {/* Precio objetivo visual */}
      {ratings.targetMean > 0 && (
        <PriceTargetRange ratings={ratings} />
      )}
    </div>
  );
}

// ---- Gauge semicircular de consenso (Venta Fuerte ← → Compra Fuerte) ----
function ConsensusGauge({ buyPercent, consensus, total }: { buyPercent: number; consensus: string; total: number }) {
  const cx = 100, cy = 85, r = 68;

  // buyPercent 0-100 mapea a ángulo -180 a 0
  const needleAngle = -180 + (buyPercent / 100) * 180;

  const polarToCartesian = (angle: number, radius: number) => ({
    x: cx + radius * Math.cos((angle * Math.PI) / 180),
    y: cy + radius * Math.sin((angle * Math.PI) / 180),
  });

  const describeArc = (startA: number, endA: number, radius: number) => {
    const start = polarToCartesian(endA, radius);
    const end = polarToCartesian(startA, radius);
    const largeArc = endA - startA > 180 ? 1 : 0;
    return `M ${start.x} ${start.y} A ${radius} ${radius} 0 ${largeArc} 0 ${end.x} ${end.y}`;
  };

  // 5 segmentos de color
  const segments = [
    { start: -180, end: -144, color: '#ef4444' }, // Venta fuerte
    { start: -144, end: -108, color: '#f97316' }, // Venta
    { start: -108, end: -72,  color: '#eab308' }, // Mantener
    { start: -72,  end: -36,  color: '#84cc16' }, // Compra
    { start: -36,  end: 0,    color: '#22c55e' }, // Compra fuerte
  ];

  const needleTip = polarToCartesian(needleAngle, r - 8);
  const consensusColor = getConsensusHex(consensus);

  return (
    <div className="flex flex-col items-center">
      <svg viewBox="0 0 200 110" className="w-[200px]">
        {/* Segmentos de fondo */}
        {segments.map((seg, i) => (
          <path key={i} d={describeArc(seg.start, seg.end, r)}
            fill="none" stroke={seg.color} strokeWidth="14" strokeLinecap="round" opacity="0.25" />
        ))}
        {/* Arco activo */}
        <path d={describeArc(-180, -180 + (buyPercent / 100) * 180, r)}
          fill="none" stroke={consensusColor} strokeWidth="14" strokeLinecap="round"
          style={{ filter: `drop-shadow(0 0 6px ${consensusColor}60)` }} />
        {/* Labels */}
        <text x="18" y="95" fill="#9ca3af" fontSize="7" textAnchor="middle">Venta</text>
        <text x="182" y="95" fill="#9ca3af" fontSize="7" textAnchor="middle">Compra</text>
        {/* Aguja */}
        <line x1={cx} y1={cy} x2={needleTip.x} y2={needleTip.y}
          stroke={consensusColor} strokeWidth="2.5" strokeLinecap="round"
          style={{ filter: `drop-shadow(0 0 4px ${consensusColor}90)`, transition: 'all 1s ease-out' }} />
        <circle cx={cx} cy={cy} r="5" fill={consensusColor} opacity="0.9" />
        <circle cx={cx} cy={cy} r="2.5" fill="#1a1a2e" />
        {/* Score central */}
        <text x={cx} y={cy + 20} fill={consensusColor} fontSize="20" fontWeight="bold"
          textAnchor="middle" fontFamily="monospace"
          style={{ filter: `drop-shadow(0 0 6px ${consensusColor}50)` }}>
          {Math.round(buyPercent)}%
        </text>
      </svg>
      {/* Etiqueta */}
      <div className="flex items-center gap-2 -mt-1">
        <span className="text-xs font-bold px-3 py-1 rounded-full"
          style={{ backgroundColor: `${consensusColor}18`, border: `1px solid ${consensusColor}40`, color: consensusColor }}>
          {consensus}
        </span>
        <span className="text-[10px] text-oscar-gray">{total} analistas</span>
      </div>
    </div>
  );
}

// ---- Histograma vertical de ratings ----
function RatingsHistogram({ ratings }: { ratings: NonNullable<MarketSentiment['analystRatings']> }) {
  const maxCount = Math.max(ratings.strongBuy, ratings.buy, ratings.hold, ratings.sell, ratings.strongSell, 1);

  const bars = [
    { label: 'Venta+', count: ratings.strongSell, color: '#ef4444' },
    { label: 'Venta',  count: ratings.sell,       color: '#f97316' },
    { label: 'Neutro', count: ratings.hold,       color: '#eab308' },
    { label: 'Compra', count: ratings.buy,        color: '#84cc16' },
    { label: 'Compra+',count: ratings.strongBuy,  color: '#22c55e' },
  ];

  return (
    <div className="flex items-end justify-center gap-2 h-[60px]">
      {bars.map((bar) => {
        const heightPct = (bar.count / maxCount) * 100;
        return (
          <div key={bar.label} className="flex flex-col items-center gap-1 flex-1 max-w-[48px]">
            {/* Número encima */}
            <span className="text-[10px] font-mono font-bold" style={{ color: bar.color }}>
              {bar.count}
            </span>
            {/* Barra */}
            <div className="w-full bg-white/5 rounded-t-md overflow-hidden" style={{ height: '36px' }}>
              <div
                className="w-full rounded-t-md transition-all duration-700 ease-out"
                style={{
                  height: `${Math.max(heightPct, 4)}%`,
                  backgroundColor: bar.color,
                  opacity: 0.75,
                  marginTop: `${100 - Math.max(heightPct, 4)}%`,
                  boxShadow: `0 0 8px ${bar.color}30`,
                }}
              />
            </div>
            {/* Etiqueta */}
            <span className="text-[7px] text-oscar-gray/60 text-center leading-tight">{bar.label}</span>
          </div>
        );
      })}
    </div>
  );
}

// ---- Rango visual de precio objetivo ----
function PriceTargetRange({ ratings }: { ratings: NonNullable<MarketSentiment['analystRatings']> }) {
  const { targetLow, targetHigh, targetMean, currentPrice, upsidePercent } = ratings;
  const range = targetHigh - targetLow;
  if (range <= 0) return null;

  // Posición del precio actual y target medio en la barra (0-100%)
  const clamp = (v: number) => Math.max(0, Math.min(100, v));
  const currentPos = clamp(((currentPrice - targetLow) / range) * 100);
  const meanPos = clamp(((targetMean - targetLow) / range) * 100);

  const upsideColor = upsidePercent >= 0 ? '#22c55e' : '#ef4444';
  const upsideIcon = upsidePercent >= 0 ? '▲' : '▼';

  return (
    <div className="bg-oscar-black/30 rounded-lg p-3">
      <div className="flex items-center justify-between mb-2">
        <span className="text-[10px] text-oscar-gray font-semibold">🎯 Precio Objetivo</span>
        <span className="text-sm font-mono font-bold text-oscar-gold" style={{ textShadow: '0 0 8px rgba(234,179,8,0.3)' }}>
          ${targetMean.toFixed(2)}
        </span>
      </div>

      {/* Barra de rango visual */}
      <div className="relative h-3 bg-white/5 rounded-full mt-1 mb-2">
        {/* Zona del rango target (toda la barra) */}
        <div className="absolute inset-0 rounded-full bg-gradient-to-r from-red-500/20 via-yellow-500/20 to-green-500/20" />

        {/* Marcador precio actual */}
        <div className="absolute top-[-4px] transition-all duration-500"
          style={{ left: `${currentPos}%`, transform: 'translateX(-50%)' }}>
          <div className="w-[3px] h-[20px] bg-white rounded-full shadow-lg" style={{ boxShadow: '0 0 6px rgba(255,255,255,0.5)' }} />
        </div>

        {/* Marcador target medio */}
        <div className="absolute top-[-4px] transition-all duration-500"
          style={{ left: `${meanPos}%`, transform: 'translateX(-50%)' }}>
          <div className="w-[3px] h-[20px] bg-oscar-gold rounded-full" style={{ boxShadow: '0 0 6px rgba(234,179,8,0.6)' }} />
        </div>
      </div>

      {/* Leyenda debajo de la barra */}
      <div className="flex justify-between items-center text-[8px] mb-2">
        <span className="text-oscar-gray">${targetLow.toFixed(0)}</span>
        <span className="text-oscar-gray">${targetHigh.toFixed(0)}</span>
      </div>

      {/* Leyenda marcadores */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <div className="w-2 h-2 bg-white rounded-full" />
          <span className="text-[9px] text-oscar-gray">Actual: <span className="text-white font-mono font-bold">${currentPrice.toFixed(2)}</span></span>
        </div>
        <span className="text-[11px] font-bold font-mono" style={{ color: upsideColor, textShadow: `0 0 8px ${upsideColor}40` }}>
          {upsideIcon} {Math.abs(upsidePercent).toFixed(1)}%
        </span>
      </div>
    </div>
  );
}

// ============================================
// Short Interest — Gauge circular + detalles
// ============================================
function ShortInterestCard({ shortInterest }: { shortInterest: MarketSentiment['shortInterest'] }) {
  if (!shortInterest) {
    return (
      <div className="bg-oscar-dark/40 rounded-xl p-4 border border-oscar-gray/10">
        <p className="text-[11px] font-semibold text-oscar-gray mb-2">🩳 Short Interest</p>
        <p className="text-xs text-oscar-gray/50 text-center py-4">Sin datos disponibles</p>
      </div>
    );
  }

  return (
    <div className="bg-oscar-dark/40 rounded-xl p-4 border border-oscar-gray/10">
      <p className="text-[11px] font-semibold text-oscar-gray mb-3">🩳 Short Interest</p>

      {/* Gauge circular del short % */}
      <div className="flex justify-center mb-3">
        <ShortCircleGauge percent={shortInterest.shortPercentOfFloat} level={shortInterest.level} />
      </div>

      {/* Detalles con iconos */}
      <div className="space-y-2">
        <ShortDetailRow icon="📅" label="Días para cubrir" value={`${shortInterest.shortRatio.toFixed(1)} días`} />
        {shortInterest.sharesShort > 0 && (
          <ShortDetailRow icon="📉" label="Acciones en corto" value={formatNumber(shortInterest.sharesShort)} />
        )}
        {shortInterest.sharesFloat > 0 && (
          <ShortDetailRow icon="🏦" label="Float total" value={formatNumber(shortInterest.sharesFloat)} />
        )}
      </div>

      {/* Explicación contextual */}
      <div className="mt-3 rounded-lg px-3 py-2" style={{
        backgroundColor: shortInterest.shortPercentOfFloat >= 10 ? 'rgba(239,68,68,0.08)' :
          shortInterest.shortPercentOfFloat >= 5 ? 'rgba(234,179,8,0.06)' : 'rgba(34,197,94,0.06)',
        border: `1px solid ${shortInterest.shortPercentOfFloat >= 10 ? 'rgba(239,68,68,0.15)' :
          shortInterest.shortPercentOfFloat >= 5 ? 'rgba(234,179,8,0.12)' : 'rgba(34,197,94,0.12)'}`,
      }}>
        <p className="text-[9px] leading-relaxed" style={{
          color: shortInterest.shortPercentOfFloat >= 10 ? '#fca5a5' :
            shortInterest.shortPercentOfFloat >= 5 ? '#fde68a' : '#86efac',
        }}>
          {shortInterest.shortPercentOfFloat >= 20
            ? '🔴 Short interest muy alto — alto riesgo de short squeeze si el precio sube con volumen'
            : shortInterest.shortPercentOfFloat >= 10
            ? '⚠️ Short interest elevado — posible short squeeze si hay catalizador alcista'
            : shortInterest.shortPercentOfFloat >= 5
            ? '🟡 Nivel moderado de posiciones en corto — presión bajista presente pero controlada'
            : '🟢 Nivel bajo de posiciones en corto — sentimiento generalmente positivo'}
        </p>
      </div>
    </div>
  );
}

// ---- Gauge circular de Short Interest ----
function ShortCircleGauge({ percent, level }: { percent: number; level: string }) {
  const size = 130;
  const strokeWidth = 12;
  const cx = size / 2, cy = size / 2;
  const r = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * r;

  // Mapear 0-30% al arco completo
  const cappedPct = Math.min(percent, 30);
  const progress = cappedPct / 30;
  const dashOffset = circumference * (1 - progress);

  const color = getShortColor(percent);

  return (
    <div className="relative flex items-center justify-center" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90">
        {/* Fondo */}
        <circle cx={cx} cy={cy} r={r} fill="none" stroke="rgba(255,255,255,0.04)" strokeWidth={strokeWidth} />
        {/* Arco de progreso */}
        <circle cx={cx} cy={cy} r={r} fill="none"
          stroke={color} strokeWidth={strokeWidth} strokeLinecap="round"
          strokeDasharray={circumference} strokeDashoffset={dashOffset}
          style={{ transition: 'stroke-dashoffset 1s ease-out', filter: `drop-shadow(0 0 6px ${color}50)` }} />
      </svg>
      {/* Centro */}
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="text-xl font-mono font-bold" style={{ color, textShadow: `0 0 10px ${color}40` }}>
          {percent.toFixed(1)}%
        </span>
        <span className="text-[9px] font-semibold mt-0.5 px-2 py-0.5 rounded-full"
          style={{ backgroundColor: `${color}15`, color, border: `1px solid ${color}30` }}>
          {level}
        </span>
      </div>
    </div>
  );
}

// ---- Fila de detalle con icono ----
function ShortDetailRow({ icon, label, value }: { icon: string; label: string; value: string }) {
  return (
    <div className="flex items-center justify-between bg-oscar-black/25 rounded-lg px-3 py-1.5">
      <div className="flex items-center gap-1.5">
        <span className="text-[10px]">{icon}</span>
        <span className="text-[10px] text-oscar-gray">{label}</span>
      </div>
      <span className="text-[11px] font-mono font-bold text-white">{value}</span>
    </div>
  );
}

// ============================================
// Helpers
// ============================================

function getConsensusHex(consensus: string): string {
  switch (consensus) {
    case 'Compra Fuerte': return '#22c55e';
    case 'Compra': return '#84cc16';
    case 'Mantener': return '#eab308';
    case 'Venta': return '#f97316';
    case 'Venta Fuerte': return '#ef4444';
    default: return '#9ca3af';
  }
}

function getShortColor(percent: number): string {
  if (percent >= 20) return '#ef4444';
  if (percent >= 10) return '#f97316';
  if (percent >= 5) return '#eab308';
  return '#22c55e';
}

function formatNumber(n: number): string {
  if (n >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(2)}B`;
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return n.toString();
}
