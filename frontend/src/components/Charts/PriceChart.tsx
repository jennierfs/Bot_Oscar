// ============================================
// Bot Oscar - Gráfico de Precios V3 (Profesional)
// TradingView Lightweight Charts con:
//   • Velas japonesas con colores vivos
//   • Barras de volumen (panel inferior)
//   • SMA50 / SMA200 dibujadas como líneas
//   • Bollinger Bands (área sombreada)
//   • EMA12 / EMA26 opcionales
//   • Leyenda dinámica OHLCV al mover el mouse
//   • Marcadores de señales COMPRA/VENTA
//   • Línea de precio actual
// ============================================
import { useEffect, useRef, useCallback, useState } from 'react';
import {
  createChart,
  ColorType,
  CrosshairMode,
  LineStyle,
  type IChartApi,
  type ISeriesApi,
  type CandlestickData,
  type HistogramData,
  type LineData,
  type Time,
} from 'lightweight-charts';
import type { Price, IndicatorValues, Signal } from '../../types';

interface PriceChartProps {
  prices: Price[];
  indicators?: IndicatorValues | null;
  signals?: Signal[];
  height?: number;
}

// Datos de la leyenda dinámica
interface LegendData {
  time: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
  change: number;
  changePercent: number;
}

export default function PriceChart({
  prices,
  indicators,
  signals,
  height = 500,
}: PriceChartProps) {
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const candleSeriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const [legend, setLegend] = useState<LegendData | null>(null);
  const [showVolume, setShowVolume] = useState(true);
  const [showSMA, setShowSMA] = useState(true);
  const [showBollinger, setShowBollinger] = useState(true);

  // Preprocesar datos de precios
  const processData = useCallback(() => {
    const dateMap = new Map<string, {
      time: string; open: number; high: number; low: number; close: number; volume: number;
    }>();

    for (const p of prices) {
      if (
        !p.date ||
        !Number.isFinite(p.open) || p.open <= 0 ||
        !Number.isFinite(p.high) || p.high <= 0 ||
        !Number.isFinite(p.low) || p.low <= 0 ||
        !Number.isFinite(p.close) || p.close <= 0
      ) continue;

      const dateKey = p.date.split('T')[0];
      dateMap.set(dateKey, {
        time: dateKey,
        open: p.open,
        high: p.high,
        low: p.low,
        close: p.close,
        volume: p.volume || 0,
      });
    }

    return Array.from(dateMap.values()).sort((a, b) => a.time.localeCompare(b.time));
  }, [prices]);

  // Calcular SMA sobre los datos
  const calculateSMA = useCallback((data: { time: string; close: number }[], period: number): LineData[] => {
    const result: LineData[] = [];
    for (let i = period - 1; i < data.length; i++) {
      let sum = 0;
      for (let j = 0; j < period; j++) {
        sum += data[i - j].close;
      }
      result.push({ time: data[i].time as Time, value: sum / period });
    }
    return result;
  }, []);

  // Calcular Bollinger Bands
  const calculateBollinger = useCallback((data: { time: string; close: number }[], period: number = 20, mult: number = 2) => {
    const upper: LineData[] = [];
    const middle: LineData[] = [];
    const lower: LineData[] = [];

    for (let i = period - 1; i < data.length; i++) {
      let sum = 0;
      for (let j = 0; j < period; j++) {
        sum += data[i - j].close;
      }
      const mean = sum / period;

      let sqSum = 0;
      for (let j = 0; j < period; j++) {
        sqSum += Math.pow(data[i - j].close - mean, 2);
      }
      const stdDev = Math.sqrt(sqSum / period);

      const t = data[i].time as Time;
      upper.push({ time: t, value: mean + mult * stdDev });
      middle.push({ time: t, value: mean });
      lower.push({ time: t, value: mean - mult * stdDev });
    }

    return { upper, middle, lower };
  }, []);

  // Calcular EMA
  const calculateEMA = useCallback((data: { time: string; close: number }[], period: number): LineData[] => {
    const result: LineData[] = [];
    const k = 2 / (period + 1);
    let ema = data[0].close;

    for (let i = 0; i < data.length; i++) {
      ema = data[i].close * k + ema * (1 - k);
      if (i >= period - 1) {
        result.push({ time: data[i].time as Time, value: ema });
      }
    }
    return result;
  }, []);

  useEffect(() => {
    if (!chartContainerRef.current) return;

    // Limpiar gráfico anterior
    if (chartRef.current) {
      chartRef.current.remove();
      chartRef.current = null;
    }

    const chartData = processData();
    if (chartData.length === 0) return;

    // ======== CREAR GRÁFICO ========
    const chart = createChart(chartContainerRef.current, {
      width: chartContainerRef.current.clientWidth,
      height,
      layout: {
        background: { type: ColorType.Solid, color: 'transparent' },
        textColor: '#64748B',
        fontSize: 11,
        fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      },
      grid: {
        vertLines: { color: 'rgba(30, 41, 59, 0.4)', style: LineStyle.Dotted },
        horzLines: { color: 'rgba(30, 41, 59, 0.4)', style: LineStyle.Dotted },
      },
      crosshair: {
        mode: CrosshairMode.Normal,
        vertLine: {
          color: 'rgba(240, 185, 11, 0.4)',
          width: 1,
          style: LineStyle.Dashed,
          labelBackgroundColor: '#F0B90B',
        },
        horzLine: {
          color: 'rgba(240, 185, 11, 0.4)',
          width: 1,
          style: LineStyle.Dashed,
          labelBackgroundColor: '#1E293B',
        },
      },
      rightPriceScale: {
        borderColor: 'rgba(30, 41, 59, 0.5)',
        scaleMargins: { top: 0.05, bottom: showVolume ? 0.28 : 0.05 },
      },
      timeScale: {
        borderColor: 'rgba(30, 41, 59, 0.5)',
        timeVisible: false,
        rightOffset: 5,
        barSpacing: 8,
        minBarSpacing: 4,
      },
      handleScroll: { mouseWheel: true, pressedMouseMove: true },
      handleScale: { mouseWheel: true, pinch: true },
    });

    // ======== VELAS JAPONESAS ========
    const candleSeries = chart.addCandlestickSeries({
      upColor: '#22C55E',
      downColor: '#EF4444',
      borderUpColor: '#16A34A',
      borderDownColor: '#DC2626',
      wickUpColor: '#22C55E',
      wickDownColor: '#EF4444',
    });

    const candleData: CandlestickData[] = chartData.map(d => ({
      time: d.time as Time,
      open: d.open,
      high: d.high,
      low: d.low,
      close: d.close,
    }));
    candleSeries.setData(candleData);
    candleSeriesRef.current = candleSeries;

    // Línea de último precio
    const lastPrice = chartData[chartData.length - 1];
    candleSeries.createPriceLine({
      price: lastPrice.close,
      color: lastPrice.close >= lastPrice.open ? '#22C55E' : '#EF4444',
      lineWidth: 1,
      lineStyle: LineStyle.Dotted,
      axisLabelVisible: true,
      title: '',
    });

    // ======== VOLUMEN ========
    if (showVolume) {
      const volumeSeries = chart.addHistogramSeries({
        priceFormat: { type: 'volume' },
        priceScaleId: 'volume',
      });

      chart.priceScale('volume').applyOptions({
        scaleMargins: { top: 0.8, bottom: 0 },
      });

      const maxVol = Math.max(...chartData.map(d => d.volume));
      const volumeData: HistogramData[] = chartData.map(d => ({
        time: d.time as Time,
        value: d.volume,
        color: d.close >= d.open
          ? `rgba(34, 197, 94, ${0.15 + (maxVol > 0 ? (d.volume / maxVol) * 0.45 : 0.15)})`
          : `rgba(239, 68, 68, ${0.15 + (maxVol > 0 ? (d.volume / maxVol) * 0.45 : 0.15)})`,
      }));
      volumeSeries.setData(volumeData);
    }

    // ======== SMA 50 & 200 ========
    if (showSMA && chartData.length >= 50) {
      const sma50Data = calculateSMA(chartData, 50);
      const sma50Series = chart.addLineSeries({
        color: '#F59E0B',
        lineWidth: 2,
        title: 'SMA50',
        crosshairMarkerVisible: false,
        lastValueVisible: false,
        priceLineVisible: false,
      });
      sma50Series.setData(sma50Data);

      if (chartData.length >= 200) {
        const sma200Data = calculateSMA(chartData, 200);
        const sma200Series = chart.addLineSeries({
          color: '#8B5CF6',
          lineWidth: 2,
          title: 'SMA200',
          crosshairMarkerVisible: false,
          lastValueVisible: false,
          priceLineVisible: false,
        });
        sma200Series.setData(sma200Data);
      }
    }

    // ======== EMA 12 & 26 ========
    if (chartData.length >= 26) {
      const ema12Data = calculateEMA(chartData, 12);
      chart.addLineSeries({
        color: '#06B6D4',
        lineWidth: 1,
        lineStyle: LineStyle.Dotted,
        crosshairMarkerVisible: false,
        lastValueVisible: false,
        priceLineVisible: false,
        title: 'EMA12',
      }).setData(ema12Data);

      const ema26Data = calculateEMA(chartData, 26);
      chart.addLineSeries({
        color: '#F472B6',
        lineWidth: 1,
        lineStyle: LineStyle.Dotted,
        crosshairMarkerVisible: false,
        lastValueVisible: false,
        priceLineVisible: false,
        title: 'EMA26',
      }).setData(ema26Data);
    }

    // ======== BOLLINGER BANDS ========
    if (showBollinger && chartData.length >= 20) {
      const bb = calculateBollinger(chartData, 20, 2);

      chart.addLineSeries({
        color: 'rgba(99, 102, 241, 0.5)',
        lineWidth: 1,
        lineStyle: LineStyle.Dashed,
        crosshairMarkerVisible: false,
        lastValueVisible: false,
        priceLineVisible: false,
        title: 'BB↑',
      }).setData(bb.upper);

      chart.addLineSeries({
        color: 'rgba(99, 102, 241, 0.3)',
        lineWidth: 1,
        crosshairMarkerVisible: false,
        lastValueVisible: false,
        priceLineVisible: false,
        title: 'BB',
      }).setData(bb.middle);

      chart.addLineSeries({
        color: 'rgba(99, 102, 241, 0.5)',
        lineWidth: 1,
        lineStyle: LineStyle.Dashed,
        crosshairMarkerVisible: false,
        lastValueVisible: false,
        priceLineVisible: false,
        title: 'BB↓',
      }).setData(bb.lower);
    }

    // ======== MARCADORES DE SEÑALES ========
    if (signals && signals.length > 0) {
      const markers = signals
        .filter(s => s.type === 'COMPRA' || s.type === 'VENTA')
        .map(s => {
          const signalDate = s.createdAt.split('T')[0];
          return {
            time: signalDate as Time,
            position: s.type === 'COMPRA' ? 'belowBar' as const : 'aboveBar' as const,
            color: s.type === 'COMPRA' ? '#22C55E' : '#EF4444',
            shape: s.type === 'COMPRA' ? 'arrowUp' as const : 'arrowDown' as const,
            text: `${s.type} ${s.strength}%`,
          };
        })
        .sort((a, b) => (a.time as string).localeCompare(b.time as string));

      if (markers.length > 0) {
        candleSeries.setMarkers(markers);
      }
    }

    // ======== LEYENDA DINÁMICA (CROSSHAIR) ========
    chart.subscribeCrosshairMove(param => {
      if (!param.time || !param.seriesData) {
        setLegend(null);
        return;
      }

      const candle = param.seriesData.get(candleSeries) as CandlestickData | undefined;
      if (!candle) {
        setLegend(null);
        return;
      }

      const matchingData = chartData.find(d => d.time === param.time);
      const change = candle.close - candle.open;
      const changePercent = ((change / candle.open) * 100);

      setLegend({
        time: param.time as string,
        open: candle.open,
        high: candle.high,
        low: candle.low,
        close: candle.close,
        volume: matchingData?.volume ?? 0,
        change,
        changePercent,
      });
    });

    // Ajustar vista
    chart.timeScale().fitContent();
    chartRef.current = chart;

    // Redimensionamiento
    const handleResize = () => {
      if (chartContainerRef.current && chartRef.current) {
        chartRef.current.applyOptions({ width: chartContainerRef.current.clientWidth });
      }
    };
    const resizeObserver = new ResizeObserver(handleResize);
    resizeObserver.observe(chartContainerRef.current);

    return () => {
      resizeObserver.disconnect();
      if (chartRef.current) {
        chartRef.current.remove();
        chartRef.current = null;
      }
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [prices, indicators, signals, showVolume, showSMA, showBollinger, height]);

  // Formatear volumen abreviado
  const fmtVol = (v: number) => {
    if (v >= 1_000_000_000) return (v / 1_000_000_000).toFixed(1) + 'B';
    if (v >= 1_000_000) return (v / 1_000_000).toFixed(1) + 'M';
    if (v >= 1_000) return (v / 1_000).toFixed(1) + 'K';
    return v.toFixed(0);
  };

  // Placeholder vacío
  if (prices.length === 0) {
    return (
      <div className="h-[500px] flex items-center justify-center">
        <div className="text-center text-oscar-gray">
          <p className="text-3xl mb-3">📊</p>
          <p className="text-sm font-medium">Sin datos de precios</p>
          <p className="text-xs text-oscar-gray/60 mt-1">Selecciona un activo o inicia el bot</p>
        </div>
      </div>
    );
  }

  const lastCandle = processData();
  const lastP = lastCandle[lastCandle.length - 1];
  const prevP = lastCandle.length > 1 ? lastCandle[lastCandle.length - 2] : lastP;
  const dayChange = lastP.close - prevP.close;
  const dayChangePct = (dayChange / prevP.close) * 100;
  const isUp = dayChange >= 0;

  return (
    <div className="relative">
      {/* ---- Header del gráfico con precio actual y leyenda ---- */}
      <div className="flex items-start justify-between mb-3">
        {/* Precio actual */}
        <div className="flex items-center gap-4">
          <div>
            <span className="text-2xl font-bold text-white font-mono">
              ${lastP.close.toFixed(2)}
            </span>
            <span className={`ml-2 text-sm font-mono font-bold ${isUp ? 'text-green-400' : 'text-red-400'}`}>
              {isUp ? '+' : ''}{dayChange.toFixed(2)} ({isUp ? '+' : ''}{dayChangePct.toFixed(2)}%)
            </span>
          </div>
        </div>

        {/* Controles de overlays */}
        <div className="flex items-center gap-1.5">
          <OverlayToggle label="VOL" active={showVolume} onClick={() => setShowVolume(!showVolume)} />
          <OverlayToggle label="SMA" active={showSMA} onClick={() => setShowSMA(!showSMA)} color="amber" />
          <OverlayToggle label="BB" active={showBollinger} onClick={() => setShowBollinger(!showBollinger)} color="indigo" />
        </div>
      </div>

      {/* ---- Leyenda OHLCV dinámica ---- */}
      <div className="h-5 mb-1">
        {legend ? (
          <div className="flex items-center gap-3 text-[11px] font-mono">
            <span className="text-oscar-gray">{legend.time}</span>
            <span className="text-oscar-gray">O:<span className="text-white ml-0.5">{legend.open.toFixed(2)}</span></span>
            <span className="text-oscar-gray">H:<span className="text-green-400 ml-0.5">{legend.high.toFixed(2)}</span></span>
            <span className="text-oscar-gray">L:<span className="text-red-400 ml-0.5">{legend.low.toFixed(2)}</span></span>
            <span className="text-oscar-gray">C:<span className={`ml-0.5 ${legend.close >= legend.open ? 'text-green-400' : 'text-red-400'}`}>{legend.close.toFixed(2)}</span></span>
            {legend.volume > 0 && (
              <span className="text-oscar-gray">V:<span className="text-purple-300 ml-0.5">{fmtVol(legend.volume)}</span></span>
            )}
            <span className={`font-bold ${legend.changePercent >= 0 ? 'text-green-400' : 'text-red-400'}`}>
              {legend.changePercent >= 0 ? '+' : ''}{legend.changePercent.toFixed(2)}%
            </span>
          </div>
        ) : (
          <div className="flex items-center gap-3 text-[11px] font-mono text-oscar-gray/40">
            <span>Mueve el cursor sobre el gráfico para ver OHLCV</span>
          </div>
        )}
      </div>

      {/* ---- Gráfico ---- */}
      <div ref={chartContainerRef} style={{ height: `${height}px` }} />

      {/* ---- Leyenda de indicadores ---- */}
      <div className="flex items-center gap-4 mt-2 text-[10px] font-mono">
        {showSMA && (
          <>
            <span className="flex items-center gap-1">
              <span className="w-3 h-0.5 bg-amber-400 inline-block rounded" /> SMA50
            </span>
            <span className="flex items-center gap-1">
              <span className="w-3 h-0.5 bg-purple-500 inline-block rounded" /> SMA200
            </span>
          </>
        )}
        <span className="flex items-center gap-1">
          <span className="w-3 h-0.5 bg-cyan-400 inline-block rounded" /> EMA12
        </span>
        <span className="flex items-center gap-1">
          <span className="w-3 h-0.5 bg-pink-400 inline-block rounded" /> EMA26
        </span>
        {showBollinger && (
          <span className="flex items-center gap-1">
            <span className="w-3 h-0.5 bg-indigo-400 inline-block rounded" /> Bollinger
          </span>
        )}
      </div>
    </div>
  );
}

// ---- Toggle de overlay ----
function OverlayToggle({
  label,
  active,
  onClick,
  color = 'gray',
}: {
  label: string;
  active: boolean;
  onClick: () => void;
  color?: 'gray' | 'amber' | 'indigo';
}) {
  const colors = {
    gray: active ? 'bg-gray-600/40 text-white border-gray-500/50' : 'bg-gray-800/30 text-gray-600 border-gray-800/30',
    amber: active ? 'bg-amber-500/15 text-amber-400 border-amber-500/30' : 'bg-gray-800/30 text-gray-600 border-gray-800/30',
    indigo: active ? 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30' : 'bg-gray-800/30 text-gray-600 border-gray-800/30',
  };

  return (
    <button
      onClick={onClick}
      className={`px-2 py-0.5 rounded text-[10px] font-bold font-mono border transition-all duration-200 ${colors[color]}`}
    >
      {label}
    </button>
  );
}
