/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        // Paleta principal de Bot Oscar
        oscar: {
          black: '#0D0D0D',     // Fondo principal
          dark: '#1A1A2E',      // Fondo de tarjetas
          card: '#16213E',      // Fondo de elementos secundarios
          gold: '#F0B90B',      // Color de acento principal (dorado)
          'gold-dark': '#C99A08', // Dorado oscuro para hover
          'gold-light': '#FFD54F', // Dorado claro para textos
          green: '#00C853',     // Ganancias / Señal de compra
          red: '#FF1744',       // Pérdidas / Señal de venta
          gray: '#8892B0',      // Texto secundario
        },
      },
      boxShadow: {
        // Sombras personalizadas con efecto glow
        'glow-gold': '0 0 20px rgba(240, 185, 11, 0.15)',
        'glow-green': '0 0 15px rgba(0, 200, 83, 0.2)',
        'glow-red': '0 0 15px rgba(255, 23, 68, 0.2)',
      },
      backdropBlur: {
        xs: '2px',
      },
    },
  },
  plugins: [],
};
