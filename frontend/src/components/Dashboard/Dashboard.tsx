// ============================================
// Bot Oscar - Dashboard principal
// Layout con tarjetas de estadísticas, gráficos y tablas
// ============================================
import type { Asset, Price, IndicatorValues, AISignal } from '../../types';

import PriceChart from '../Charts/PriceChart';

import AISignalPanel from '../AI/AISignalPanel';
import FearGreedGauge from '../FearGreed/FearGreedGauge';
import SentimentPanel from '../Sentiment/SentimentPanel';

interface DashboardProps {
  prices: Price[];
  indicators: IndicatorValues | null;
  selectedAsset: Asset | null;
  aiSignal: AISignal | null;
  aiLoading: boolean;
  aiError: string | null;
  onGenerateSignal: () => void;
}

export default function Dashboard({
  prices,
  indicators,
  selectedAsset,
  aiSignal,
  aiLoading,
  aiError,
  onGenerateSignal,
}: DashboardProps) {

  return (
    <main className="flex-1 overflow-y-auto px-4 py-4 bg-gradient-dark">
      {/* Gráfico de precios profesional */}
      <div className="glass-card p-3 mb-6 animate-fade-in">
        <div className="flex items-center justify-between mb-1">
          <div>
            <h2 className="text-lg font-bold text-white">
              {selectedAsset?.name ?? 'Selecciona un activo'}
            </h2>
            <p className="text-xs text-oscar-gray">
              {selectedAsset?.symbol ?? ''} • Velas diarias • Yahoo Finance
            </p>
          </div>
          {indicators && aiSignal && (
            <div className="flex gap-3 text-xs">
              <IndicatorBadge label="RSI" value={indicators.rsi.toFixed(1)} color={indicators.rsi > 70 ? 'red' : indicators.rsi < 30 ? 'green' : 'gold'} />
              <IndicatorBadge label="Score" value={`${indicators.score}`} color={indicators.score >= 65 ? 'green' : indicators.score <= 35 ? 'red' : 'gold'} />
              <IndicatorBadge label="Señal" value={indicators.signal} color={indicators.signal === 'COMPRA' ? 'green' : indicators.signal === 'VENTA' ? 'red' : 'gold'} />
            </div>
          )}
        </div>
        <PriceChart
          symbol={selectedAsset?.symbol ?? ''}
          height={500}
        />
      </div>

      {/* Fila 2: Índice de Miedo & Codicia del activo seleccionado */}
      <div className="mb-6">
        <FearGreedGauge selectedAsset={selectedAsset} />
      </div>

      {/* Fila 3: Sentimiento de Mercado (Analyst Ratings + Short Interest) */}
      <div className="mb-6">
        <SentimentPanel selectedAsset={selectedAsset} />
      </div>

      {/* Fila 4: Señales IA (DeepSeek) - ancho completo */}
      <div className="mb-6">
        <AISignalPanel
          selectedAsset={selectedAsset}
          aiSignal={aiSignal}
          loading={aiLoading}
          error={aiError}
          onGenerateSignal={onGenerateSignal}
        />
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
