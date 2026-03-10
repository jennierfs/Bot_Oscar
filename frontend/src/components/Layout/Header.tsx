// ============================================
// Bot Oscar - Header (Encabezado)
// Logo + Selector de activo + Estado del bot + Controles
// ============================================
import { Play, Square, Activity, Clock, BarChart3, Sparkles, Loader2 } from 'lucide-react';
import type { Asset, BotStatus } from '../../types';
import AssetSelector from './AssetSelector';

interface HeaderProps {
  botStatus: BotStatus | null;
  onStartBot: () => void;
  onStopBot: () => void;
  assets: Asset[];
  selectedAsset: Asset | null;
  onSelectAsset: (asset: Asset) => void;
  onGenerateSignal: () => void;
  aiLoading: boolean;
}

export default function Header({
  botStatus,
  onStartBot,
  onStopBot,
  assets,
  selectedAsset,
  onSelectAsset,
  onGenerateSignal,
  aiLoading,
}: HeaderProps) {
  const isRunning = botStatus?.running ?? false;

  const lastAnalysis = botStatus?.lastAnalysis
    ? new Date(botStatus.lastAnalysis).toLocaleTimeString('es-ES')
    : 'Sin datos';

  return (
    <header className="h-16 bg-oscar-dark/30 border-b border-gray-800/50 flex items-center justify-between px-6">
      {/* Lado izquierdo: Logo + Selector */}
      <div className="flex items-center gap-5">
        {/* Logo */}
        <div className="flex items-center gap-2.5">
          <div className="w-9 h-9 bg-oscar-gold/20 rounded-xl flex items-center justify-center">
            <BarChart3 className="w-5 h-5 text-oscar-gold" />
          </div>
          <div className="hidden sm:block">
            <h1 className="text-sm font-bold text-white leading-tight">Bot Oscar</h1>
            <p className="text-[10px] text-oscar-gray leading-tight">Trading Auto</p>
          </div>
        </div>

        {/* Separador */}
        <div className="w-px h-8 bg-gray-700/50" />

        {/* Selector de activo */}
        <AssetSelector
          assets={assets}
          selectedAsset={selectedAsset}
          onSelectAsset={onSelectAsset}
        />

        {/* Separador */}
        <div className="w-px h-8 bg-gray-700/50" />

        {/* Botón Generar Señal IA */}
        <button
          onClick={onGenerateSignal}
          disabled={aiLoading || !selectedAsset}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg font-bold text-xs transition-all duration-300 ${
            aiLoading
              ? 'bg-purple-500/20 text-purple-300 cursor-wait border border-purple-500/30'
              : !selectedAsset
              ? 'bg-gray-800/50 text-gray-500 cursor-not-allowed border border-gray-700/30'
              : 'bg-gradient-to-r from-purple-600 to-purple-500 text-white hover:from-purple-500 hover:to-purple-400 hover:shadow-lg hover:shadow-purple-500/20 active:scale-[0.98]'
          }`}
        >
          {aiLoading ? (
            <>
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
              <span className="hidden md:inline">Analizando...</span>
            </>
          ) : (
            <>
              <Sparkles className="w-3.5 h-3.5" />
              <span className="hidden md:inline">Generar Señal IA</span>
            </>
          )}
        </button>
      </div>

      {/* Centro: Info del bot */}
      <div className="hidden lg:flex items-center gap-5">
        <div className="flex items-center gap-2">
          <div
            className={`w-2.5 h-2.5 rounded-full ${
              isRunning ? 'bg-oscar-green animate-pulse-dot' : 'bg-oscar-red'
            }`}
          />
          <span className="text-xs font-medium">
            {isRunning ? 'Activo' : 'Detenido'}
          </span>
        </div>

        <div className="flex items-center gap-1.5 text-oscar-gray">
          <Activity className="w-3.5 h-3.5" />
          <span className="text-[11px] uppercase tracking-wider">
            {botStatus?.mode ?? 'paper'}
          </span>
        </div>

        <div className="flex items-center gap-1.5 text-oscar-gray">
          <Clock className="w-3.5 h-3.5" />
          <span className="text-[11px]">Análisis: {lastAnalysis}</span>
        </div>


      </div>

      {/* Lado derecho: Controles */}
      <div className="flex items-center gap-3">
        {isRunning ? (
          <button
            onClick={onStopBot}
            className="btn-danger flex items-center gap-2 text-sm"
          >
            <Square className="w-4 h-4" />
            <span className="hidden sm:inline">Detener</span>
          </button>
        ) : (
          <button
            onClick={onStartBot}
            className="btn-gold flex items-center gap-2 text-sm"
          >
            <Play className="w-4 h-4" />
            <span className="hidden sm:inline">Iniciar</span>
          </button>
        )}
      </div>
    </header>
  );
}
