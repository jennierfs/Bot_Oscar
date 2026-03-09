// ============================================
// Bot Oscar - Sidebar (Barra lateral)
// Muestra la lista de activos y permite seleccionar uno
// ============================================
import { BarChart3, TrendingUp, Shield } from 'lucide-react';
import type { Asset } from '../../types';

interface SidebarProps {
  assets: Asset[];
  selectedAsset: Asset | null;
  onSelectAsset: (asset: Asset) => void;
}

export default function Sidebar({ assets, selectedAsset, onSelectAsset }: SidebarProps) {
  // Separar activos por tipo
  const commodities = assets.filter(a => a.type === 'commodity');
  const stocks = assets.filter(a => a.type === 'accion');

  return (
    <aside className="w-64 bg-oscar-dark/50 border-r border-gray-800/50 flex flex-col overflow-hidden">
      {/* Logo y título */}
      <div className="p-5 border-b border-gray-800/50">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-oscar-gold/20 rounded-xl flex items-center justify-center">
            <BarChart3 className="w-6 h-6 text-oscar-gold" />
          </div>
          <div>
            <h1 className="text-lg font-bold text-white">Bot Oscar</h1>
            <p className="text-xs text-oscar-gray">Trading Automático</p>
          </div>
        </div>
      </div>

      {/* Lista de activos */}
      <div className="flex-1 overflow-y-auto p-3">
        {/* Commodities */}
        <div className="mb-4">
          <div className="flex items-center gap-2 px-2 mb-2">
            <TrendingUp className="w-4 h-4 text-oscar-gold" />
            <span className="text-xs font-semibold text-oscar-gold uppercase tracking-wider">
              Commodities
            </span>
          </div>
          {commodities.map(asset => (
            <AssetItem
              key={asset.id}
              asset={asset}
              isSelected={selectedAsset?.id === asset.id}
              onClick={() => onSelectAsset(asset)}
            />
          ))}
        </div>

        {/* Acciones de defensa */}
        <div>
          <div className="flex items-center gap-2 px-2 mb-2">
            <Shield className="w-4 h-4 text-oscar-gold" />
            <span className="text-xs font-semibold text-oscar-gold uppercase tracking-wider">
              Defensa
            </span>
          </div>
          {stocks.map(asset => (
            <AssetItem
              key={asset.id}
              asset={asset}
              isSelected={selectedAsset?.id === asset.id}
              onClick={() => onSelectAsset(asset)}
            />
          ))}
        </div>
      </div>

      {/* Pie del sidebar */}
      <div className="p-4 border-t border-gray-800/50">
        <p className="text-xs text-oscar-gray text-center">
          v1.0.0 • Modo Paper
        </p>
      </div>
    </aside>
  );
}

// ---- Componente de cada activo en la lista ----
function AssetItem({
  asset,
  isSelected,
  onClick,
}: {
  asset: Asset;
  isSelected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`w-full text-left px-3 py-2.5 rounded-lg mb-1 transition-all duration-200
        ${isSelected
          ? 'bg-oscar-gold/15 border border-oscar-gold/30 text-white'
          : 'hover:bg-gray-800/50 text-oscar-gray hover:text-white border border-transparent'
        }`}
    >
      <div className="flex items-center justify-between">
        <div>
          <p className={`text-sm font-medium ${isSelected ? 'text-oscar-gold' : ''}`}>
            {asset.symbol.replace('=F', '')}
          </p>
          <p className="text-xs text-oscar-gray truncate max-w-[140px]">
            {asset.name}
          </p>
        </div>
        {isSelected && (
          <div className="w-2 h-2 bg-oscar-gold rounded-full animate-pulse-dot" />
        )}
      </div>
    </button>
  );
}
