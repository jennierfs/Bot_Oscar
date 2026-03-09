// ============================================
// Bot Oscar - Panel de Portafolio
// Muestra el resumen del portafolio y operaciones abiertas
// ============================================
import { Wallet, ArrowUpRight, ArrowDownRight } from 'lucide-react';
import type { Portfolio, Operation } from '../../types';

interface PortfolioPanelProps {
  portfolio: Portfolio | null;
}

export default function PortfolioPanel({ portfolio }: PortfolioPanelProps) {
  // Formatear moneda USD
  const formatUSD = (n: number) =>
    new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(n);

  return (
    <div className="glass-card p-5 animate-fade-in">
      {/* Encabezado */}
      <div className="flex items-center gap-2 mb-4">
        <Wallet className="w-5 h-5 text-oscar-gold" />
        <h3 className="text-lg font-bold">Portafolio</h3>
      </div>

      {/* Resumen */}
      {portfolio ? (
        <>
          <div className="grid grid-cols-2 gap-3 mb-4">
            <MiniStat
              label="Capital"
              value={formatUSD(portfolio.capital)}
              color="text-oscar-gold"
            />
            <MiniStat
              label="P&L Total"
              value={formatUSD(portfolio.gananciaPerdida)}
              color={
                portfolio.gananciaPerdida >= 0 ? 'text-oscar-green' : 'text-oscar-red'
              }
            />
            <MiniStat
              label="Retorno"
              value={`${portfolio.porcentajeRetorno.toFixed(2)}%`}
              color={
                portfolio.porcentajeRetorno >= 0 ? 'text-oscar-green' : 'text-oscar-red'
              }
            />
            <MiniStat
              label="Total Ops"
              value={String(portfolio.totalOperaciones)}
              color="text-white"
            />
          </div>

          {/* Operaciones abiertas */}
          <div className="mt-4">
            <h4 className="text-sm font-semibold text-oscar-gray mb-3">
              Operaciones Abiertas ({portfolio.operacionesAbiertas})
            </h4>
            {portfolio.operaciones.length === 0 ? (
              <div className="text-center py-4 text-oscar-gray">
                <p className="text-xs">Sin operaciones abiertas</p>
              </div>
            ) : (
              <div className="space-y-2 max-h-[280px] overflow-y-auto">
                {portfolio.operaciones.map(op => (
                  <OperationRow key={op.id} operation={op} />
                ))}
              </div>
            )}
          </div>
        </>
      ) : (
        <div className="text-center py-8 text-oscar-gray">
          <p className="text-sm">Cargando portafolio...</p>
        </div>
      )}
    </div>
  );
}

// ---- Mini estadística ----
function MiniStat({
  label,
  value,
  color,
}: {
  label: string;
  value: string;
  color: string;
}) {
  return (
    <div className="bg-oscar-dark/40 rounded-lg p-3">
      <p className="text-xs text-oscar-gray mb-1">{label}</p>
      <p className={`text-sm font-bold font-mono ${color}`}>{value}</p>
    </div>
  );
}

// ---- Fila de operación ----
function OperationRow({ operation }: { operation: Operation }) {
  const isLong = operation.type === 'COMPRA';
  const Icon = isLong ? ArrowUpRight : ArrowDownRight;

  const formatUSD = (n: number) =>
    new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(n);

  return (
    <div className="bg-oscar-dark/30 border border-gray-800/50 rounded-lg p-3 flex items-center justify-between hover:border-oscar-gold/20 transition-all">
      <div className="flex items-center gap-3">
        <div
          className={`p-1.5 rounded-lg ${
            isLong ? 'bg-oscar-green/15' : 'bg-oscar-red/15'
          }`}
        >
          <Icon
            className={`w-4 h-4 ${
              isLong ? 'text-oscar-green' : 'text-oscar-red'
            }`}
          />
        </div>
        <div>
          <p className="text-sm font-medium text-white">
            {operation.symbol.replace('=F', '')}
          </p>
          <p className="text-xs text-oscar-gray">
            {operation.type} • {operation.quantity.toFixed(4)} uds
          </p>
        </div>
      </div>
      <div className="text-right">
        <p className="text-sm font-mono text-white">
          {formatUSD(operation.entryPrice)}
        </p>
        <p className="text-xs text-oscar-gray">
          SL: {formatUSD(operation.stopLoss)}
        </p>
      </div>
    </div>
  );
}
