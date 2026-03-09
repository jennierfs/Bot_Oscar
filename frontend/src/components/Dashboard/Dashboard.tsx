// ============================================
// Bot Oscar - Dashboard principal
// Layout con tarjetas de estadísticas, gráficos y tablas
// ============================================
import { DollarSign, TrendingUp, BarChart3, Zap } from 'lucide-react';
import type { Asset, Price, Signal, Portfolio, IndicatorValues } from '../../types';
import StatCard from './StatCard';
import PriceChart from '../Charts/PriceChart';
import SignalPanel from '../Signals/SignalPanel';
import PortfolioPanel from '../Portfolio/Portfolio';
import AISignalPanel from '../AI/AISignalPanel';

interface DashboardProps {
  portfolio: Portfolio | null;
  signals: Signal[];
  prices: Price[];
  indicators: IndicatorValues | null;
  selectedAsset: Asset | null;
}

export default function Dashboard({
  portfolio,
  signals,
  prices,
  indicators,
  selectedAsset,
}: DashboardProps) {
  // Formatear números como moneda USD
  const formatUSD = (n: number) =>
    new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(n);

  // Determinar tendencia del retorno
  const returnTrend: 'up' | 'down' | 'neutral' =
    portfolio && portfolio.porcentajeRetorno > 0
      ? 'up'
      : portfolio && portfolio.porcentajeRetorno < 0
      ? 'down'
      : 'neutral';

  // Señales recientes del activo seleccionado
  const assetSignals = selectedAsset
    ? signals.filter(s => s.symbol === selectedAsset.symbol)
    : signals;

  // Última puntuación de indicadores
  const lastScore = indicators?.score ?? 0;
  const scoreColor = lastScore >= 65 ? 'green' : lastScore <= 35 ? 'red' : 'gold';

  return (
    <main className="flex-1 overflow-y-auto p-6 bg-gradient-dark">
      {/* Fila 1: Tarjetas de estadísticas */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <StatCard
          title="Capital Total"
          value={formatUSD(portfolio?.capital ?? 10000)}
          subtitle={`Inicial: ${formatUSD(portfolio?.capitalInicial ?? 10000)}`}
          icon={<DollarSign className="w-5 h-5 text-oscar-gold" />}
          accentColor="gold"
        />
        <StatCard
          title="Ganancia / Pérdida"
          value={formatUSD(portfolio?.gananciaPerdida ?? 0)}
          subtitle={`${(portfolio?.porcentajeRetorno ?? 0).toFixed(2)}%`}
          icon={<TrendingUp className="w-5 h-5 text-oscar-green" />}
          trend={returnTrend}
          accentColor={returnTrend === 'down' ? 'red' : 'green'}
        />
        <StatCard
          title="Operaciones Abiertas"
          value={String(portfolio?.operacionesAbiertas ?? 0)}
          subtitle={`Total: ${portfolio?.totalOperaciones ?? 0}`}
          icon={<BarChart3 className="w-5 h-5 text-oscar-gold" />}
          accentColor="gold"
        />
        <StatCard
          title="Puntuación Señal"
          value={`${lastScore}/100`}
          subtitle={indicators?.signal ?? 'Sin datos'}
          icon={<Zap className="w-5 h-5 text-oscar-gold" />}
          accentColor={scoreColor}
        />
      </div>

      {/* Fila 2: Gráfico de precios profesional */}
      <div className="glass-card p-5 mb-6 animate-fade-in">
        <div className="flex items-center justify-between mb-2">
          <div>
            <h2 className="text-lg font-bold text-white">
              {selectedAsset?.name ?? 'Selecciona un activo'}
            </h2>
            <p className="text-xs text-oscar-gray">
              {selectedAsset?.symbol ?? ''} • Velas diarias • Yahoo Finance
            </p>
          </div>
          {indicators && (
            <div className="flex gap-3 text-xs">
              <IndicatorBadge label="RSI" value={indicators.rsi.toFixed(1)} color={indicators.rsi > 70 ? 'red' : indicators.rsi < 30 ? 'green' : 'gold'} />
              <IndicatorBadge label="Score" value={`${indicators.score}`} color={indicators.score >= 65 ? 'green' : indicators.score <= 35 ? 'red' : 'gold'} />
              <IndicatorBadge label="Señal" value={indicators.signal} color={indicators.signal === 'COMPRA' ? 'green' : indicators.signal === 'VENTA' ? 'red' : 'gold'} />
            </div>
          )}
        </div>
        <PriceChart
          prices={prices}
          indicators={indicators}
          signals={assetSignals}
          height={520}
        />
      </div>

      {/* Fila 3: Señales IA (DeepSeek) - ancho completo */}
      <div className="mb-6">
        <AISignalPanel selectedAsset={selectedAsset} />
      </div>

      {/* Fila 4: Señales del motor y Portafolio lado a lado */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <SignalPanel signals={assetSignals.length > 0 ? assetSignals : signals} />
        <PortfolioPanel portfolio={portfolio} />
      </div>
    </main>
  );
}

// ---- Mini badge de indicador ----
function IndicatorBadge({ label, value, color = 'gold' }: { label: string; value: string; color?: string }) {
  const colorClasses: Record<string, string> = {
    gold: 'text-oscar-gold border-oscar-gold/30',
    green: 'text-green-400 border-green-500/30',
    red: 'text-red-400 border-red-500/30',
  };
  return (
    <div className={`bg-oscar-dark/60 border rounded-lg px-3 py-1.5 ${colorClasses[color] || colorClasses.gold}`}>
      <span className="text-oscar-gray">{label}: </span>
      <span className="font-mono font-bold">{value}</span>
    </div>
  );
}
