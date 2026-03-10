// ============================================
// Bot Oscar - Gráfico TradingView (Widget Avanzado)
// Embebe el gráfico real de TradingView con:
//   • Velas profesionales idénticas a tradingview.com
//   • Indicadores activables por el usuario (RSI, MACD, BB, VWAP, ATR)
//   • Herramientas de dibujo (líneas, fibonacci, etc.)
//   • Todos los timeframes (1m a 1M)
//   • Datos en tiempo real
//   • Tema oscuro nativo
// ============================================
import { useEffect, useRef, useState, memo } from 'react';

interface TradingViewChartProps {
  symbol: string;
  height?: number;
}

// Indicadores disponibles para activar/desactivar
interface IndicatorToggle {
  id: string;
  label: string;
  studyId: string;
  color: string;       // Color del botón activo
  description: string; // Tooltip
}

const AVAILABLE_INDICATORS: IndicatorToggle[] = [
  { id: 'vwap',      label: 'VWAP',      studyId: 'STD;VWAP',            color: '#a78bfa', description: 'Precio ponderado por volumen' },
  { id: 'bollinger', label: 'Bollinger',  studyId: 'STD;Bollinger_Bands', color: '#38bdf8', description: 'Bandas de Bollinger (20, 2)' },
  { id: 'rsi',       label: 'RSI',        studyId: 'STD;RSI',             color: '#fbbf24', description: 'Índice de Fuerza Relativa (14)' },
  { id: 'macd',      label: 'MACD',       studyId: 'STD;MACD',            color: '#34d399', description: 'MACD (12, 26, 9)' },
  { id: 'atr',       label: 'ATR',        studyId: 'STD;Average_True_Range', color: '#f87171', description: 'Volatilidad real (14)' },
  { id: 'sma50',     label: 'SMA 50',     studyId: 'STD;SMA',             color: '#fb923c', description: 'Media móvil simple 50' },
  { id: 'sma200',    label: 'SMA 200',    studyId: 'STD;SMA',             color: '#e879f9', description: 'Media móvil simple 200' },
];

// Mapeo de símbolos internos → símbolos de TradingView
const SYMBOL_MAP: Record<string, string> = {
  // ETFs de Commodities (antes eran futuros, ahora ETFs para compatibilidad total)
  'GLD': 'AMEX:GLD',        // ETF Oro (SPDR Gold Trust)
  'SLV': 'AMEX:SLV',        // ETF Plata (iShares Silver)
  'USO': 'AMEX:USO',        // ETF Petróleo (US Oil Fund)
  'UNG': 'AMEX:UNG',        // ETF Gas Natural (US Natural Gas)
  // Acciones de defensa (NYSE/NASDAQ)
  'LMT': 'NYSE:LMT',
  'RTX': 'NYSE:RTX',
  'NOC': 'NYSE:NOC',
  'GD': 'NYSE:GD',
  'BA': 'NYSE:BA',
  'LHX': 'NYSE:LHX',
  'HII': 'NYSE:HII',
  'KTOS': 'NASDAQ:KTOS',
  'AVAV': 'NASDAQ:AVAV',
  'PLTR': 'NYSE:PLTR',
  'LDOS': 'NYSE:LDOS',
  'SAIC': 'NYSE:SAIC',
  'MRCY': 'NASDAQ:MRCY',
  'TXT': 'NYSE:TXT',
  'HEI': 'NYSE:HEI',
  'RKLB': 'NASDAQ:RKLB',
};

function TradingViewChart({ symbol, height = 700 }: TradingViewChartProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [activeIndicators, setActiveIndicators] = useState<Set<string>>(new Set());

  // Convertir símbolo al formato TradingView
  const tvSymbol = SYMBOL_MAP[symbol] || symbol;

  // Construir la lista de studies según los indicadores activos
  const activeStudies = AVAILABLE_INDICATORS
    .filter(ind => activeIndicators.has(ind.id))
    .map(ind => ind.studyId);

  // Serializar para usar como dependencia del useEffect
  const studiesKey = Array.from(activeIndicators).sort().join(',');

  const toggleIndicator = (id: string) => {
    setActiveIndicators(prev => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  useEffect(() => {
    if (!containerRef.current) return;

    // Limpiar widget anterior
    const container = containerRef.current;
    container.innerHTML = '';

    // Crear y cargar el script del widget
    const script = document.createElement('script');
    script.src = 'https://s3.tradingview.com/external-embedding/embed-widget-advanced-chart.js';
    script.type = 'text/javascript';
    script.async = true;
    script.innerHTML = JSON.stringify({
      width: '100%',
      height: height,
      symbol: tvSymbol,
      interval: 'D',
      timezone: 'America/New_York',
      theme: 'dark',
      style: '1',
      locale: 'es',
      backgroundColor: 'rgba(13, 13, 13, 1)',
      gridColor: 'rgba(30, 41, 59, 0.3)',
      hide_top_toolbar: false,
      hide_legend: false,
      allow_symbol_change: false,
      save_image: true,
      calendar: false,
      hide_volume: false,
      support_host: 'https://www.tradingview.com',
      // Solo cargar los indicadores que el usuario activó
      studies: activeStudies,
      overrides: {
        'mainSeriesProperties.candleStyle.upColor': '#22C55E',
        'mainSeriesProperties.candleStyle.downColor': '#EF4444',
        'mainSeriesProperties.candleStyle.borderUpColor': '#16A34A',
        'mainSeriesProperties.candleStyle.borderDownColor': '#DC2626',
        'mainSeriesProperties.candleStyle.wickUpColor': '#22C55E',
        'mainSeriesProperties.candleStyle.wickDownColor': '#EF4444',
        'paneProperties.backgroundType': 'solid',
        'paneProperties.background': 'rgba(13, 13, 13, 1)',
        'scalesProperties.backgroundColor': 'rgba(13, 13, 13, 1)',
      },
    });

    container.appendChild(script);

    return () => {
      if (container) {
        container.innerHTML = '';
      }
    };
  }, [tvSymbol, height, studiesKey]);

  return (
    <div>
      {/* Barra de indicadores toggle */}
      <div className="flex flex-wrap items-center gap-1.5 mb-2">
        <span className="text-[10px] text-oscar-gray mr-1">📊 Indicadores:</span>
        {AVAILABLE_INDICATORS.map(ind => {
          const isActive = activeIndicators.has(ind.id);
          return (
            <button
              key={ind.id}
              onClick={() => toggleIndicator(ind.id)}
              title={ind.description}
              className={`
                px-2 py-0.5 rounded text-[10px] font-semibold
                transition-all duration-200 border
                ${isActive
                  ? 'text-white border-opacity-60'
                  : 'text-oscar-gray border-white/10 bg-white/[0.02] hover:bg-white/[0.05] hover:text-white/80'
                }
              `}
              style={isActive ? {
                backgroundColor: `${ind.color}20`,
                borderColor: `${ind.color}60`,
                color: ind.color,
                boxShadow: `0 0 8px ${ind.color}25`,
              } : undefined}
            >
              {ind.label}
            </button>
          );
        })}
        {activeIndicators.size > 0 && (
          <button
            onClick={() => setActiveIndicators(new Set())}
            className="px-2 py-0.5 rounded text-[10px] text-red-400/70 border border-red-500/20 hover:bg-red-500/10 hover:text-red-400 transition-all ml-1"
            title="Quitar todos los indicadores"
          >
            ✕ Limpiar
          </button>
        )}
      </div>

      {/* Gráfico TradingView */}
      <div
        className="tradingview-widget-container"
        ref={containerRef}
        style={{ height: `${height}px`, width: '100%' }}
      />
    </div>
  );
}

// Memo para evitar re-renders innecesarios cuando cambian otros datos del dashboard
export default memo(TradingViewChart, (prev, next) => {
  return prev.symbol === next.symbol && prev.height === next.height;
});
