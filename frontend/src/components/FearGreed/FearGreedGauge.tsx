// ============================================
// Bot Oscar - Medidor de Miedo & Codicia por Activo
// Gauge visual estilo CNN Fear & Greed Index
// Cada activo tiene su propio medidor independiente
// ============================================
import { useState, useEffect } from 'react';
import type { Asset, FearGreedResult } from '../../types';
import * as api from '../../services/api';

interface FearGreedGaugeProps {
  selectedAsset: Asset | null;
}

export default function FearGreedGauge({ selectedAsset }: FearGreedGaugeProps) {
  const [data, setData] = useState<FearGreedResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Cargar Fear & Greed cuando cambia el activo seleccionado
  useEffect(() => {
    if (!selectedAsset) {
      setData(null);
      return;
    }

    const fetchFearGreed = async () => {
      setLoading(true);
      setError(null);
      try {
        const result = await api.getFearGreed(selectedAsset.symbol);
        setData(result);
      } catch (err) {
        console.error('Error obteniendo Fear & Greed:', err);
        setError('Error calculando índice');
        setData(null);
      } finally {
        setLoading(false);
      }
    };

    fetchFearGreed();

    // Refrescar cada 30 segundos
    const interval = setInterval(fetchFearGreed, 30000);
    return () => clearInterval(interval);
  }, [selectedAsset?.symbol]);

  if (!selectedAsset) {
    return null;
  }

  return (
    <div className="glass-card p-3 animate-fade-in">
      {/* Header */}
      <div className="flex items-center gap-2 mb-2">
        <span className="text-sm">😨</span>
        <h3 className="text-xs font-bold text-white">Índice de Miedo & Codicia</h3>
        <span className="text-[10px] text-oscar-gray">• {selectedAsset.symbol}</span>
      </div>

      {loading && !data && (
        <div className="flex items-center justify-center py-4">
          <div className="animate-spin rounded-full h-6 w-6 border-2 border-oscar-gold border-t-transparent" />
        </div>
      )}

      {error && !data && (
        <p className="text-red-400 text-xs text-center py-2">{error}</p>
      )}

      {data && (
        <div className="flex gap-4 items-start">
          {/* Izquierda: Gauge compacto */}
          <div className="flex-shrink-0 w-[180px]">
            <GaugeMeter score={data.score} label={data.label} />
          </div>

          {/* Derecha: Factores */}
          <div className="flex-1 min-w-0 space-y-1">
            <p className="text-[10px] text-oscar-gray font-semibold mb-1">Desglose:</p>
            {data.components.map((comp) => (
              <ComponentBar key={comp.name} comp={comp} />
            ))}
            <p className="text-[9px] text-oscar-gray/50 pt-1 leading-snug">
              {data.description}
            </p>
          </div>
        </div>
      )}
    </div>
  );
}

// ---- Gauge semicircular con aguja ----
function GaugeMeter({ score, label }: { score: number; label: string }) {
  // Colores por zona
  const getColor = (s: number) => {
    if (s <= 20) return '#ef4444'; // rojo - miedo extremo
    if (s <= 40) return '#f97316'; // naranja - miedo
    if (s <= 60) return '#eab308'; // amarillo - neutral
    if (s <= 80) return '#84cc16'; // verde claro - codicia
    return '#22c55e'; // verde - codicia extrema
  };

  const getEmoji = (s: number) => {
    if (s <= 20) return '😱';
    if (s <= 40) return '😰';
    if (s <= 60) return '😐';
    if (s <= 80) return '🤑';
    return '🔥';
  };

  const color = getColor(score);
  const emoji = getEmoji(score);

  // Parámetros del arco SVG
  const cx = 90, cy = 82, r = 65;
  const startAngle = -180;
  const endAngle = 0;
  const needleAngle = startAngle + (score / 100) * (endAngle - startAngle);

  // Función para punto en arco
  const polarToCartesian = (angle: number, radius: number) => ({
    x: cx + radius * Math.cos((angle * Math.PI) / 180),
    y: cy + radius * Math.sin((angle * Math.PI) / 180),
  });

  // Crear path del arco de fondo con segmentos de color
  const segments = [
    { start: -180, end: -144, color: '#ef4444' },
    { start: -144, end: -108, color: '#f97316' },
    { start: -108, end: -72,  color: '#eab308' },
    { start: -72,  end: -36,  color: '#84cc16' },
    { start: -36,  end: 0,    color: '#22c55e' },
  ];

  const describeArc = (startA: number, endA: number, radius: number) => {
    const start = polarToCartesian(endA, radius);
    const end = polarToCartesian(startA, radius);
    const largeArc = endA - startA > 180 ? 1 : 0;
    return `M ${start.x} ${start.y} A ${radius} ${radius} 0 ${largeArc} 0 ${end.x} ${end.y}`;
  };

  const needleTip = polarToCartesian(needleAngle, r - 6);

  return (
    <div className="flex flex-col items-center">
      <svg viewBox="0 0 180 105" className="w-full">
        {segments.map((seg, i) => (
          <path
            key={i}
            d={describeArc(seg.start, seg.end, r)}
            fill="none"
            stroke={seg.color}
            strokeWidth="11"
            strokeLinecap="round"
            opacity="0.3"
          />
        ))}
        <path
          d={describeArc(-180, -180 + (score / 100) * 180, r)}
          fill="none"
          stroke={color}
          strokeWidth="11"
          strokeLinecap="round"
          style={{ filter: `drop-shadow(0 0 4px ${color}80)` }}
        />
        <text x="16" y="90" fill="#9ca3af" fontSize="7" textAnchor="middle">0</text>
        <text x="90" y="17" fill="#9ca3af" fontSize="7" textAnchor="middle">50</text>
        <text x="164" y="90" fill="#9ca3af" fontSize="7" textAnchor="middle">100</text>
        <line x1={cx} y1={cy} x2={needleTip.x} y2={needleTip.y}
          stroke={color} strokeWidth="2" strokeLinecap="round"
          style={{ filter: `drop-shadow(0 0 3px ${color}90)`, transition: 'all 1s ease-out' }}
        />
        <circle cx={cx} cy={cy} r="4" fill={color} opacity="0.9" />
        <circle cx={cx} cy={cy} r="2" fill="#1a1a2e" />
        <text x={cx} y={cy + 18} fill={color} fontSize="22" fontWeight="bold"
          textAnchor="middle" fontFamily="monospace"
          style={{ filter: `drop-shadow(0 0 6px ${color}60)` }}>
          {score}
        </text>
      </svg>
      {/* Etiqueta */}
      <div className="flex items-center gap-1.5 px-3 py-1 rounded-full -mt-1"
        style={{ backgroundColor: `${color}15`, border: `1px solid ${color}40` }}>
        <span className="text-sm">{emoji}</span>
        <span className="text-xs font-bold" style={{ color }}>{label}</span>
      </div>
    </div>
  );
}

// ---- Barra de componente individual ----
function ComponentBar({ comp }: { comp: { name: string; score: number; weight: number; detail: string } }) {
  const getBarColor = (s: number) => {
    if (s <= 20) return 'bg-red-500';
    if (s <= 40) return 'bg-orange-500';
    if (s <= 60) return 'bg-yellow-500';
    if (s <= 80) return 'bg-lime-500';
    return 'bg-green-500';
  };

  const getTextColor = (s: number) => {
    if (s <= 20) return 'text-red-400';
    if (s <= 40) return 'text-orange-400';
    if (s <= 60) return 'text-yellow-400';
    if (s <= 80) return 'text-lime-400';
    return 'text-green-400';
  };

  return (
    <div className="group/bar flex items-center gap-2">
      <span className="text-[9px] text-oscar-gray truncate w-[90px] flex-shrink-0" title={comp.detail}>
        {comp.name}
      </span>
      <div className="flex-1 h-1 bg-white/5 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full ${getBarColor(comp.score)} transition-all duration-700 ease-out`}
          style={{ width: `${comp.score}%`, opacity: 0.8 }}
        />
      </div>
      <span className={`text-[9px] font-mono font-bold w-[20px] text-right ${getTextColor(comp.score)}`}>
        {Math.round(comp.score)}
      </span>
    </div>
  );
}
