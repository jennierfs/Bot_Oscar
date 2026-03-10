// ============================================
// Bot Oscar - Gráfico TradingView (Widget Avanzado)
// Embebe el gráfico real de TradingView con:
//   • Velas profesionales idénticas a tradingview.com
//   • Todos los indicadores integrados (RSI, MACD, BB, SMA, EMA...)
//   • Herramientas de dibujo (líneas, fibonacci, etc.)
//   • Todos los timeframes (1m a 1M)
//   • Datos en tiempo real
//   • Tema oscuro nativo
//   • Sin gaps raros de mercado
// ============================================
import { useEffect, useRef, memo } from 'react';

interface TradingViewChartProps {
  symbol: string;
  height?: number;
}

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

  // Convertir símbolo al formato TradingView
  const tvSymbol = SYMBOL_MAP[symbol] || symbol;

  useEffect(() => {
    if (!containerRef.current) return;

    // Limpiar widget anterior
    const container = containerRef.current;
    container.innerHTML = '';

    // Crear y cargar el script del widget
    // TradingView espera: el script dentro del contenedor,
    // y el iframe se crea automáticamente con las dimensiones del config
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
      style: '1',                    // 1 = Candlestick
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
      studies: [
        // === TENDENCIA ===
        // VWAP - Precio ponderado por volumen (clave institucional)
        'STD;VWAP',
        // === VOLATILIDAD ===
        // Bollinger Bands (20, 2)
        'STD;Bollinger_Bands',
        // ATR - Average True Range (volatilidad para SL/TP)
        'STD;Average_True_Range',
        // === OSCILADORES ===
        // RSI (14 periodos)
        'STD;RSI',
        // MACD (12, 26, 9)
        'STD;MACD',
      ],
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
  }, [tvSymbol, height]);

  return (
    <div
      className="tradingview-widget-container"
      ref={containerRef}
      style={{ height: `${height}px`, width: '100%' }}
    />
  );
}

// Memo para evitar re-renders innecesarios cuando cambian otros datos del dashboard
export default memo(TradingViewChart, (prev, next) => {
  return prev.symbol === next.symbol && prev.height === next.height;
});
