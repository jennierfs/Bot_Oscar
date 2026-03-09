// ============================================
// Bot Oscar - Panel de Señales
// Muestra las señales de trading generadas por el motor
// ============================================
import { Zap, ArrowUpCircle, ArrowDownCircle, MinusCircle } from 'lucide-react';
import type { Signal } from '../../types';

interface SignalPanelProps {
  signals: Signal[];
}

export default function SignalPanel({ signals }: SignalPanelProps) {
  return (
    <div className="glass-card p-5 animate-fade-in">
      {/* Encabezado */}
      <div className="flex items-center gap-2 mb-4">
        <Zap className="w-5 h-5 text-oscar-gold" />
        <h3 className="text-lg font-bold">Señales Recientes</h3>
      </div>

      {/* Lista de señales */}
      {signals.length === 0 ? (
        <div className="text-center py-8 text-oscar-gray">
          <p className="text-sm">No hay señales generadas aún</p>
          <p className="text-xs mt-1">Inicia el bot para comenzar el análisis</p>
        </div>
      ) : (
        <div className="space-y-3 max-h-[400px] overflow-y-auto">
          {signals.slice(0, 10).map(signal => (
            <SignalItem key={signal.id} signal={signal} />
          ))}
        </div>
      )}
    </div>
  );
}

// ---- Componente de cada señal ----
function SignalItem({ signal }: { signal: Signal }) {
  // Ícono según tipo de señal
  const Icon =
    signal.type === 'COMPRA'
      ? ArrowUpCircle
      : signal.type === 'VENTA'
      ? ArrowDownCircle
      : MinusCircle;

  // Color según tipo
  const typeColor =
    signal.type === 'COMPRA'
      ? 'text-oscar-green'
      : signal.type === 'VENTA'
      ? 'text-oscar-red'
      : 'text-oscar-gold';

  // Badge class
  const badgeClass =
    signal.type === 'COMPRA'
      ? 'badge-compra'
      : signal.type === 'VENTA'
      ? 'badge-venta'
      : 'badge-mantener';

  // Color de la barra de fuerza
  const strengthColor =
    signal.strength >= 65
      ? 'bg-oscar-green'
      : signal.strength <= 35
      ? 'bg-oscar-red'
      : 'bg-oscar-gold';

  // Formatear hora
  const time = new Date(signal.createdAt).toLocaleString('es-ES', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });

  return (
    <div className="bg-oscar-dark/40 border border-gray-800/50 rounded-lg p-3 hover:border-oscar-gold/20 transition-all">
      {/* Fila superior: símbolo, tipo, fuerza */}
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <Icon className={`w-4 h-4 ${typeColor}`} />
          <span className="text-sm font-bold text-white">{signal.symbol.replace('=F', '')}</span>
          <span className={badgeClass}>{signal.type}</span>
        </div>
        <span className="text-xs text-oscar-gray">{time}</span>
      </div>

      {/* Barra de fuerza */}
      <div className="mb-2">
        <div className="flex items-center justify-between text-xs mb-1">
          <span className="text-oscar-gray">Fuerza</span>
          <span className={`font-bold ${typeColor}`}>{signal.strength}/100</span>
        </div>
        <div className="w-full h-1.5 bg-gray-800 rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all duration-500 ${strengthColor}`}
            style={{ width: `${signal.strength}%` }}
          />
        </div>
      </div>

      {/* Precios: Entrada, SL, TP */}
      <div className="grid grid-cols-3 gap-2 text-xs">
        <div>
          <span className="text-oscar-gray block">Entrada</span>
          <span className="text-white font-mono">${signal.entryPrice.toFixed(2)}</span>
        </div>
        <div>
          <span className="text-oscar-gray block">Stop Loss</span>
          <span className="text-oscar-red font-mono">${signal.stopLoss.toFixed(2)}</span>
        </div>
        <div>
          <span className="text-oscar-gray block">Take Profit</span>
          <span className="text-oscar-green font-mono">${signal.takeProfit.toFixed(2)}</span>
        </div>
      </div>

      {/* Razón */}
      <p className="text-xs text-oscar-gray mt-2 leading-relaxed">
        {signal.reason}
      </p>
    </div>
  );
}
