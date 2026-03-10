// ============================================
// Bot Oscar - Componente principal de la aplicación
// Gestiona el estado global y la comunicación con el backend
// ============================================
import { useState, useEffect, useCallback } from 'react';
import type { Asset, Price, Signal, Portfolio, BotStatus, IndicatorValues, AISignal } from './types';
import * as api from './services/api';
import Header from './components/Layout/Header';
import Dashboard from './components/Dashboard/Dashboard';

function App() {
  // ---- Estado de la aplicación ----
  const [assets, setAssets] = useState<Asset[]>([]);
  const [selectedAsset, setSelectedAsset] = useState<Asset | null>(null);
  const [prices, setPrices] = useState<Price[]>([]);
  const [signals, setSignals] = useState<Signal[]>([]);
  const [portfolio, setPortfolio] = useState<Portfolio | null>(null);
  const [botStatus, setBotStatus] = useState<BotStatus | null>(null);
  const [indicators, setIndicators] = useState<IndicatorValues | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // ---- Estado de señal IA (compartido entre Header y AISignalPanel) ----
  const [aiSignal, setAiSignal] = useState<AISignal | null>(null);
  const [aiLoading, setAiLoading] = useState(false);
  const [aiError, setAiError] = useState<string | null>(null);

  // ---- Función para cargar datos generales ----
  const fetchGeneralData = useCallback(async () => {
    try {
      const [assetsData, signalsData, portfolioData, statusData] = await Promise.all([
        api.getAssets().catch(() => []),
        api.getSignals().catch(() => []),
        api.getPortfolio().catch(() => null),
        api.getBotStatus().catch(() => null),
      ]);

      setAssets(assetsData);
      setSignals(signalsData);
      setPortfolio(portfolioData);
      setBotStatus(statusData);
      setError(null);

      // Seleccionar el primer activo si no hay ninguno seleccionado
      if (!selectedAsset && assetsData.length > 0) {
        setSelectedAsset(assetsData[0]);
      }
    } catch (err) {
      setError('Error conectando con el servidor');
      console.error('Error cargando datos:', err);
    } finally {
      setLoading(false);
    }
  }, [selectedAsset]);

  // ---- Función para cargar datos del activo seleccionado ----
  const fetchAssetData = useCallback(async (asset: Asset) => {
    try {
      const [pricesData, indicatorsData] = await Promise.all([
        api.getPrices(asset.id).catch(() => []),
        api.getIndicators(asset.symbol).catch(() => null),
      ]);
      setPrices(pricesData);
      setIndicators(indicatorsData);
    } catch (err) {
      console.error('Error cargando datos del activo:', err);
    }
  }, []);

  // ---- Cargar datos al montar y configurar polling ----
  useEffect(() => {
    fetchGeneralData();
    // Refrescar datos cada 10 segundos
    const interval = setInterval(fetchGeneralData, 10000);
    return () => clearInterval(interval);
  }, [fetchGeneralData]);

  // ---- Cargar datos cuando cambia el activo seleccionado ----
  useEffect(() => {
    if (selectedAsset) {
      fetchAssetData(selectedAsset);
    }
  }, [selectedAsset, fetchAssetData]);

  // ---- Handler para generar señal IA (compartido Header + AISignalPanel) ----
  const handleGenerateAISignal = async () => {
    if (!selectedAsset) return;
    setAiLoading(true);
    setAiError(null);
    setAiSignal(null);
    try {
      const signal = await api.generateAISignal(selectedAsset.symbol);
      setAiSignal(signal);
    } catch (err: unknown) {
      const errorMsg = err instanceof Error ? err.message : 'Error generando señal IA';
      const axiosErr = err as { response?: { data?: { error?: string } } };
      setAiError(axiosErr.response?.data?.error || errorMsg);
    } finally {
      setAiLoading(false);
    }
  };

  // ---- Handlers del bot ----
  const handleStartBot = async () => {
    try {
      await api.startBot();
      const status = await api.getBotStatus();
      setBotStatus(status);
    } catch (err) {
      console.error('Error iniciando bot:', err);
    }
  };

  const handleStopBot = async () => {
    try {
      await api.stopBot();
      const status = await api.getBotStatus();
      setBotStatus(status);
    } catch (err) {
      console.error('Error deteniendo bot:', err);
    }
  };

  // ---- Pantalla de carga ----
  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen bg-oscar-black">
        <div className="text-center animate-fade-in">
          <div className="text-6xl mb-4">🤖</div>
          <h1 className="text-2xl font-bold text-oscar-gold mb-2">Bot Oscar</h1>
          <p className="text-oscar-gray">Conectando con el servidor...</p>
        </div>
      </div>
    );
  }

  // ---- Pantalla de error de conexión ----
  if (error && assets.length === 0) {
    return (
      <div className="flex items-center justify-center h-screen bg-oscar-black">
        <div className="text-center glass-card p-8 animate-fade-in">
          <div className="text-6xl mb-4">⚠️</div>
          <h1 className="text-2xl font-bold text-oscar-gold mb-2">Sin Conexión</h1>
          <p className="text-oscar-gray mb-4">{error}</p>
          <p className="text-sm text-oscar-gray">
            Asegúrate de que el backend esté ejecutándose en el puerto 8080
          </p>
          <button
            onClick={fetchGeneralData}
            className="btn-gold mt-4"
          >
            Reintentar
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen bg-oscar-black overflow-hidden">
      {/* Encabezado con logo, selector de activo y estado del bot */}
      <Header
        botStatus={botStatus}
        onStartBot={handleStartBot}
        onStopBot={handleStopBot}
        assets={assets}
        selectedAsset={selectedAsset}
        onSelectAsset={setSelectedAsset}
        onGenerateSignal={handleGenerateAISignal}
        aiLoading={aiLoading}
      />

      {/* Dashboard con gráficos e indicadores — ancho completo */}
      <Dashboard
        prices={prices}
        indicators={indicators}
        selectedAsset={selectedAsset}
        aiSignal={aiSignal}
        aiLoading={aiLoading}
        aiError={aiError}
        onGenerateSignal={handleGenerateAISignal}
      />
    </div>
  );
}

export default App;
