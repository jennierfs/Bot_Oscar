// ============================================
// Bot Oscar - StatCard (Tarjeta de estadística)
// Componente reutilizable con efecto glassmorphism
// ============================================
import type { ReactNode } from 'react';

interface StatCardProps {
  title: string;
  value: string;
  subtitle?: string;
  icon: ReactNode;
  trend?: 'up' | 'down' | 'neutral';
  accentColor?: 'gold' | 'green' | 'red';
}

export default function StatCard({
  title,
  value,
  subtitle,
  icon,
  trend,
  accentColor = 'gold',
}: StatCardProps) {
  // Colores de acento según el tipo
  const accentStyles = {
    gold: 'text-oscar-gold border-oscar-gold/20',
    green: 'text-oscar-green border-oscar-green/20',
    red: 'text-oscar-red border-oscar-red/20',
  };

  const iconBgStyles = {
    gold: 'bg-oscar-gold/10',
    green: 'bg-oscar-green/10',
    red: 'bg-oscar-red/10',
  };

  return (
    <div className="glass-card p-5 animate-fade-in">
      <div className="flex items-start justify-between mb-3">
        <div className={`p-2.5 rounded-xl ${iconBgStyles[accentColor]}`}>
          {icon}
        </div>
        {trend && (
          <span
            className={`text-xs font-bold px-2 py-1 rounded-full ${
              trend === 'up'
                ? 'bg-oscar-green/15 text-oscar-green'
                : trend === 'down'
                ? 'bg-oscar-red/15 text-oscar-red'
                : 'bg-oscar-gold/15 text-oscar-gold'
            }`}
          >
            {trend === 'up' ? '▲' : trend === 'down' ? '▼' : '—'}
          </span>
        )}
      </div>
      <p className="text-oscar-gray text-xs uppercase tracking-wider mb-1">{title}</p>
      <p className={`text-2xl font-bold ${accentStyles[accentColor].split(' ')[0]}`}>
        {value}
      </p>
      {subtitle && (
        <p className="text-xs text-oscar-gray mt-1">{subtitle}</p>
      )}
    </div>
  );
}
