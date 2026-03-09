// ============================================
// Bot Oscar - Asset Selector (Dropdown elegante)
// Reemplaza el sidebar con un dropdown compacto
// con categorías desplegables y búsqueda
// ============================================
import { useState, useRef, useEffect } from 'react';
import {
  ChevronDown,
  Search,
  TrendingUp,
  Shield,
  Crosshair,
  Cpu,
  Rocket,
  X,
} from 'lucide-react';
import type { Asset } from '../../types';

interface AssetSelectorProps {
  assets: Asset[];
  selectedAsset: Asset | null;
  onSelectAsset: (asset: Asset) => void;
}

// Subcategorías para las acciones de defensa
const DEFENSE_SUBCATEGORIES: Record<string, { label: string; icon: React.ReactNode; symbols: string[] }> = {
  contratistas: {
    label: 'Contratistas Principales',
    icon: <Shield className="w-3.5 h-3.5" />,
    symbols: ['LMT', 'RTX', 'NOC', 'GD', 'BA', 'LHX', 'HII'],
  },
  drones: {
    label: 'Drones & Autónomos',
    icon: <Crosshair className="w-3.5 h-3.5" />,
    symbols: ['KTOS', 'AVAV'],
  },
  tech: {
    label: 'Tech & Ciber Militar',
    icon: <Cpu className="w-3.5 h-3.5" />,
    symbols: ['PLTR', 'LDOS', 'SAIC', 'MRCY'],
  },
  aero: {
    label: 'Aeroespacial & Componentes',
    icon: <Rocket className="w-3.5 h-3.5" />,
    symbols: ['TXT', 'HEI', 'RKLB'],
  },
};

export default function AssetSelector({ assets, selectedAsset, onSelectAsset }: AssetSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState('');
  const dropdownRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  // Separar activos por tipo
  const commodities = assets.filter(a => a.type === 'commodity');
  const stocks = assets.filter(a => a.type === 'accion');

  // Filtrar por búsqueda
  const filterAssets = (list: Asset[]) => {
    if (!search) return list;
    const q = search.toLowerCase();
    return list.filter(
      a => a.symbol.toLowerCase().includes(q) || a.name.toLowerCase().includes(q)
    );
  };

  const filteredCommodities = filterAssets(commodities);
  const filteredStocks = filterAssets(stocks);

  // Agrupar stocks en subcategorías
  const getSubcategoryStocks = (symbols: string[]) =>
    filteredStocks.filter(s => symbols.includes(s.symbol));

  // Cerrar dropdown al hacer clic fuera
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setIsOpen(false);
        setSearch('');
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Focus en búsqueda al abrir
  useEffect(() => {
    if (isOpen && searchRef.current) {
      searchRef.current.focus();
    }
  }, [isOpen]);

  // Manejar selección
  const handleSelect = (asset: Asset) => {
    onSelectAsset(asset);
    setIsOpen(false);
    setSearch('');
  };

  // Nombre corto para mostrar
  const displaySymbol = (symbol: string) => symbol.replace('=F', '');

  return (
    <div className="relative" ref={dropdownRef}>
      {/* Botón del selector */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={`flex items-center gap-3 px-4 py-2 rounded-xl border transition-all duration-200 min-w-[260px] ${
          isOpen
            ? 'bg-oscar-gold/10 border-oscar-gold/50 shadow-lg shadow-oscar-gold/5'
            : 'bg-oscar-dark/60 border-gray-700/50 hover:border-oscar-gold/30 hover:bg-oscar-dark/80'
        }`}
      >
        {/* Indicador tipo */}
        <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${
          selectedAsset?.type === 'commodity'
            ? 'bg-amber-500/15 text-amber-400'
            : 'bg-purple-500/15 text-purple-400'
        }`}>
          {selectedAsset?.type === 'commodity'
            ? <TrendingUp className="w-4 h-4" />
            : <Shield className="w-4 h-4" />
          }
        </div>

        {/* Info del activo seleccionado */}
        <div className="flex-1 text-left">
          <div className="flex items-center gap-2">
            <span className="text-white font-bold text-sm">
              {selectedAsset ? displaySymbol(selectedAsset.symbol) : 'Seleccionar'}
            </span>
            <span className={`text-[9px] px-1.5 py-0.5 rounded-full font-medium uppercase tracking-wider ${
              selectedAsset?.type === 'commodity'
                ? 'bg-amber-500/15 text-amber-400 border border-amber-500/20'
                : 'bg-purple-500/15 text-purple-400 border border-purple-500/20'
            }`}>
              {selectedAsset?.type === 'commodity' ? 'Commodity' : 'Defensa'}
            </span>
          </div>
          <p className="text-[11px] text-oscar-gray truncate max-w-[180px]">
            {selectedAsset?.name ?? 'Elige un activo para analizar'}
          </p>
        </div>

        {/* Flecha */}
        <ChevronDown className={`w-4 h-4 text-oscar-gray transition-transform duration-200 ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div className="absolute top-full left-0 mt-2 w-[340px] bg-oscar-card border border-gray-700/60 rounded-xl shadow-2xl shadow-black/40 z-50 overflow-hidden animate-fade-in">
          {/* Buscador */}
          <div className="p-3 border-b border-gray-800/50">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-oscar-gray" />
              <input
                ref={searchRef}
                type="text"
                placeholder="Buscar activo..."
                value={search}
                onChange={e => setSearch(e.target.value)}
                className="w-full pl-9 pr-8 py-2 bg-oscar-dark/60 border border-gray-700/50 rounded-lg text-sm text-white placeholder:text-oscar-gray/50 focus:outline-none focus:border-oscar-gold/40 transition-colors"
              />
              {search && (
                <button
                  onClick={() => setSearch('')}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-oscar-gray hover:text-white"
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              )}
            </div>
          </div>

          {/* Lista scrolleable */}
          <div className="max-h-[420px] overflow-y-auto py-2 custom-scrollbar">
            {/* === COMMODITIES === */}
            {filteredCommodities.length > 0 && (
              <div className="mb-2">
                <div className="flex items-center gap-2 px-4 py-1.5">
                  <TrendingUp className="w-3.5 h-3.5 text-amber-400" />
                  <span className="text-[10px] font-bold text-amber-400 uppercase tracking-widest">
                    Commodities
                  </span>
                  <span className="text-[10px] text-oscar-gray">({filteredCommodities.length})</span>
                </div>
                {filteredCommodities.map(asset => (
                  <DropdownItem
                    key={asset.id}
                    asset={asset}
                    isSelected={selectedAsset?.id === asset.id}
                    onClick={() => handleSelect(asset)}
                    accentColor="amber"
                  />
                ))}
              </div>
            )}

            {/* === DEFENSA (con subcategorías) === */}
            {filteredStocks.length > 0 && (
              <div>
                <div className="flex items-center gap-2 px-4 py-1.5 mt-1">
                  <Shield className="w-3.5 h-3.5 text-purple-400" />
                  <span className="text-[10px] font-bold text-purple-400 uppercase tracking-widest">
                    Defensa & Armamento
                  </span>
                  <span className="text-[10px] text-oscar-gray">({filteredStocks.length})</span>
                </div>

                {search ? (
                  // Si hay búsqueda, mostrar flat sin subcategorías
                  filteredStocks.map(asset => (
                    <DropdownItem
                      key={asset.id}
                      asset={asset}
                      isSelected={selectedAsset?.id === asset.id}
                      onClick={() => handleSelect(asset)}
                      accentColor="purple"
                    />
                  ))
                ) : (
                  // Sin búsqueda, mostrar con subcategorías
                  Object.entries(DEFENSE_SUBCATEGORIES).map(([key, cat]) => {
                    const catStocks = getSubcategoryStocks(cat.symbols);
                    if (catStocks.length === 0) return null;
                    return (
                      <div key={key} className="mb-1">
                        <div className="flex items-center gap-1.5 px-5 py-1 mt-1">
                          <span className="text-purple-400/60">{cat.icon}</span>
                          <span className="text-[10px] text-purple-300/60 font-medium">
                            {cat.label}
                          </span>
                        </div>
                        {catStocks.map(asset => (
                          <DropdownItem
                            key={asset.id}
                            asset={asset}
                            isSelected={selectedAsset?.id === asset.id}
                            onClick={() => handleSelect(asset)}
                            accentColor="purple"
                          />
                        ))}
                      </div>
                    );
                  })
                )}
              </div>
            )}

            {/* Sin resultados */}
            {filteredCommodities.length === 0 && filteredStocks.length === 0 && (
              <div className="text-center py-6 text-oscar-gray">
                <Search className="w-6 h-6 mx-auto mb-2 opacity-30" />
                <p className="text-xs">No se encontró "{search}"</p>
              </div>
            )}
          </div>

          {/* Footer del dropdown */}
          <div className="border-t border-gray-800/50 px-4 py-2 flex items-center justify-between">
            <span className="text-[10px] text-oscar-gray">
              {assets.length} activos disponibles
            </span>
            <span className="text-[10px] text-oscar-gray/50">
              ESC para cerrar
            </span>
          </div>
        </div>
      )}
    </div>
  );
}

// ---- Item individual del dropdown ----
function DropdownItem({
  asset,
  isSelected,
  onClick,
  accentColor,
}: {
  asset: Asset;
  isSelected: boolean;
  onClick: () => void;
  accentColor: 'amber' | 'purple';
}) {
  const displaySymbol = asset.symbol.replace('=F', '');
  const selectedBg = accentColor === 'amber'
    ? 'bg-amber-500/10 border-l-2 border-l-amber-400'
    : 'bg-purple-500/10 border-l-2 border-l-purple-400';
  const hoverBg = 'hover:bg-gray-800/40';

  return (
    <button
      onClick={onClick}
      className={`w-full text-left px-4 py-2 flex items-center gap-3 transition-all duration-150 ${
        isSelected ? selectedBg : `${hoverBg} border-l-2 border-l-transparent`
      }`}
    >
      {/* Símbolo */}
      <div className={`w-10 h-7 rounded flex items-center justify-center text-xs font-bold font-mono ${
        isSelected
          ? accentColor === 'amber' ? 'bg-amber-500/20 text-amber-300' : 'bg-purple-500/20 text-purple-300'
          : 'bg-gray-800/60 text-oscar-gray'
      }`}>
        {displaySymbol.substring(0, 4)}
      </div>

      {/* Nombre */}
      <div className="flex-1 min-w-0">
        <p className={`text-sm font-medium truncate ${isSelected ? 'text-white' : 'text-oscar-gray'}`}>
          {asset.name}
        </p>
        <p className="text-[10px] text-oscar-gray/60 font-mono">{asset.symbol}</p>
      </div>

      {/* Indicador seleccionado */}
      {isSelected && (
        <div className={`w-1.5 h-1.5 rounded-full ${
          accentColor === 'amber' ? 'bg-amber-400' : 'bg-purple-400'
        } animate-pulse-dot`} />
      )}
    </button>
  );
}
