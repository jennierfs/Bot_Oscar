// ============================================
// Bot Oscar - Servicio API
// Todas las llamadas HTTP al backend Go
// ============================================
import axios from 'axios';
import type { Asset, Price, Signal, Operation, Portfolio, BotStatus, IndicatorValues, AISignal } from '../types';

// Cliente HTTP configurado con la base URL de la API
const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
});

// ---- Activos ----
export const getAssets = () =>
  api.get<Asset[]>('/activos').then(res => res.data);

export const getPrices = (assetId: number, limit = 200) =>
  api.get<Price[]>(`/activos/${assetId}/precios`, { params: { limit } }).then(res => res.data);

// ---- Señales ----
export const getSignals = (limit = 20) =>
  api.get<Signal[]>('/senales', { params: { limit } }).then(res => res.data);

// ---- Operaciones ----
export const getOperations = (estado?: string) =>
  api.get<Operation[]>('/operaciones', { params: { estado } }).then(res => res.data);

// ---- Portafolio ----
export const getPortfolio = () =>
  api.get<Portfolio>('/portafolio').then(res => res.data);

// ---- Configuración ----
export const getConfig = () =>
  api.get<Record<string, string>>('/configuracion').then(res => res.data);

export const updateConfig = (clave: string, valor: string) =>
  api.post('/configuracion', { clave, valor });

// ---- Bot ----
export const getBotStatus = () =>
  api.get<BotStatus>('/bot/estado').then(res => res.data);

export const startBot = () =>
  api.post('/bot/iniciar');

export const stopBot = () =>
  api.post('/bot/detener');

// ---- Indicadores ----
export const getIndicators = (symbol: string) =>
  api.get<IndicatorValues>(`/indicadores/${symbol}`).then(res => res.data);

// ---- Señales IA (DeepSeek) ----
// Timeout más largo porque DeepSeek puede tardar hasta 30s en responder
export const generateAISignal = (symbol: string) =>
  api.post<AISignal>(`/ia/senal/${symbol}`, null, { timeout: 60000 }).then(res => res.data);
