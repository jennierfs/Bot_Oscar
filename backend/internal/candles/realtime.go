// ============================================
// Bot Oscar - Actualizador de Velas en Tiempo Real
// Mantiene las velas actualizadas automáticamente
// mientras el mercado de EE.UU. está abierto.
//
// Funcionamiento:
//   - Ciclo continuo que recorre todos los activos × timeframes
//   - Round-robin ponderado: 5min se actualiza 2x más que 1h
//   - ~30 segundos entre llamadas API (2 créditos/min)
//   - Respeta el límite de 800 créditos/día de Twelve Data
//   - Solo activo en horario de mercado (9:15-16:15 ET)
//   - Auto-pausa cuando se agotan los créditos del día
//
// Presupuesto de créditos (plan gratis 800/día):
//   - 700 para actualización en tiempo real
//   - 100 reservados para operaciones manuales (IA, etc.)
//
// Con 20 activos y timeframes [5min×2, 1h×1]:
//   - 60 tareas por ciclo × 30s = ~30 min por ciclo completo
//   - ~13 ciclos por día = ~780 créditos
//   - Cada activo actualiza 5min cada ~10 min, 1h cada ~30 min
//
// ============================================
package candles

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"bot-oscar/internal/db"
	"bot-oscar/internal/market"
	"bot-oscar/internal/models"
)

// ============================================
// Configuración del actualizador en tiempo real
// ============================================

// RealtimeConfig configuración del actualizador
type RealtimeConfig struct {
	IntervalSeconds  int            `json:"intervalSeconds"`  // Segundos entre llamadas API
	MaxCreditsPerDay int64          `json:"maxCreditsPerDay"` // Créditos máximos para RT
	MarketHoursOnly  bool           `json:"marketHoursOnly"`  // Solo en horario de mercado
	Timeframes       []string       `json:"timeframes"`       // Timeframes a actualizar
	Weights          map[string]int `json:"weights"`          // Peso por timeframe (más = más frecuente)
}

// DefaultRealtimeConfig configuración por defecto optimizada para plan gratis
// 800 créditos/día → 700 para RT + 100 para operaciones manuales
var DefaultRealtimeConfig = RealtimeConfig{
	IntervalSeconds:  30,
	MaxCreditsPerDay: 700,
	MarketHoursOnly:  true,
	Timeframes:       []string{"5min", "1h"},
	Weights: map[string]int{
		"5min": 2, // Se actualiza 2x más que otros
		"1h":   1,
	},
}

// ============================================
// Estado del actualizador
// ============================================

// RealtimeStatus estado actual visible via API
type RealtimeStatus struct {
	Running            bool           `json:"running"`
	MarketOpen         bool           `json:"marketOpen"`
	LastUpdate         string         `json:"lastUpdate,omitempty"`
	CurrentAsset       string         `json:"currentAsset,omitempty"`
	CurrentTimeframe   string         `json:"currentTimeframe,omitempty"`
	UpdatesThisCycle   int            `json:"updatesThisCycle"`
	UpdatesToday       int64          `json:"updatesToday"`
	CandlesSavedToday  int64          `json:"candlesSavedToday"`
	CreditsUsedToday   int64          `json:"creditsUsedToday"`
	CycleNumber        int            `json:"cycleNumber"`
	TotalTasksPerCycle int            `json:"totalTasksPerCycle"`
	CycleTimeMinutes   float64        `json:"cycleTimeMinutes"`
	NextUpdate         string         `json:"nextUpdate,omitempty"`
	Errors             []string       `json:"errors"`
	StartedAt          string         `json:"startedAt,omitempty"`
	Config             RealtimeConfig `json:"config"`
}

// ============================================
// Actualizador en tiempo real
// ============================================

// updateTask una tarea individual de actualización
type updateTask struct {
	Asset     models.Asset
	Timeframe string
}

// RealtimeUpdater mantiene las velas actualizadas en tiempo real
type RealtimeUpdater struct {
	provider *market.TwelveDataProvider
	db       *db.Database

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	status  RealtimeStatus

	// Contadores atómicos (thread-safe sin mutex)
	updatesToday      atomic.Int64
	candlesSavedToday atomic.Int64

	config RealtimeConfig
}

// NewRealtimeUpdater crea un nuevo actualizador en tiempo real
func NewRealtimeUpdater(provider *market.TwelveDataProvider, database *db.Database) *RealtimeUpdater {
	return &RealtimeUpdater{
		provider: provider,
		db:       database,
		config:   DefaultRealtimeConfig,
		status: RealtimeStatus{
			Errors: make([]string, 0),
		},
	}
}

// ============================================
// Control: Iniciar / Detener / Estado
// ============================================

// IsRunning indica si el actualizador está corriendo
func (u *RealtimeUpdater) IsRunning() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.running
}

// GetStatus devuelve el estado actual del actualizador
func (u *RealtimeUpdater) GetStatus() RealtimeStatus {
	u.mu.Lock()
	defer u.mu.Unlock()
	s := u.status
	s.UpdatesToday = u.updatesToday.Load()
	s.CandlesSavedToday = u.candlesSavedToday.Load()
	s.CreditsUsedToday = u.provider.CreditosUsadosHoy()
	s.MarketOpen = isMarketOpen()
	s.Config = u.config
	return s
}

// Start inicia el actualizador en segundo plano
func (u *RealtimeUpdater) Start() error {
	u.mu.Lock()
	if u.running {
		u.mu.Unlock()
		return fmt.Errorf("actualizador en tiempo real ya está activo")
	}
	u.running = true
	u.stopCh = make(chan struct{})
	u.status = RealtimeStatus{
		Running:   true,
		StartedAt: time.Now().Format(time.RFC3339),
		Errors:    make([]string, 0),
	}
	u.mu.Unlock()

	go u.run()
	log.Println("🔴 LIVE — Actualizador de velas en tiempo real INICIADO")
	return nil
}

// Stop detiene el actualizador
func (u *RealtimeUpdater) Stop() {
	u.mu.Lock()
	if !u.running {
		u.mu.Unlock()
		return
	}
	close(u.stopCh)
	u.running = false
	u.status.Running = false
	u.mu.Unlock()
	log.Println("⏹️  Actualizador de velas en tiempo real DETENIDO")
}

// ============================================
// Bucle principal de actualización
// ============================================

func (u *RealtimeUpdater) run() {
	// Capturar nuestro stopCh para detectar si Start() creó uno nuevo
	u.mu.Lock()
	myStopCh := u.stopCh
	u.mu.Unlock()

	defer func() {
		u.mu.Lock()
		// Solo resetear si no se ha llamado Start() de nuevo
		// (si Start() creó un nuevo stopCh, no tocamos el estado)
		if u.stopCh == myStopCh {
			u.running = false
			u.status.Running = false
		}
		u.mu.Unlock()
		log.Println("⏹️  LIVE — Bucle de actualización finalizado")
	}()

	ctx := context.Background()

	// Obtener todos los activos activos
	assets, err := u.db.GetActiveAssets(ctx)
	if err != nil {
		u.addError("Error obteniendo activos: " + err.Error())
		return
	}

	// Construir cola de tareas ponderada
	tasks := u.buildTaskQueue(assets)
	if len(tasks) == 0 {
		u.addError("No hay tareas para ejecutar (sin activos o timeframes)")
		return
	}

	cycleTime := float64(len(tasks)*u.config.IntervalSeconds) / 60.0

	u.mu.Lock()
	u.status.TotalTasksPerCycle = len(tasks)
	u.status.CycleTimeMinutes = cycleTime
	u.mu.Unlock()

	log.Printf("🔴 LIVE — Monitoreando %d activos × %v = %d tareas/ciclo (%.0f min/ciclo)",
		len(assets), u.config.Timeframes, len(tasks), cycleTime)

	taskIndex := 0
	cycleNum := 0
	interval := time.Duration(u.config.IntervalSeconds) * time.Second
	lastResetDay := time.Now().Truncate(24 * time.Hour)

	for {
		// Verificar señal de stop
		select {
		case <-u.stopCh:
			return
		default:
		}

		// ── Reset diario de contadores ──
		today := time.Now().Truncate(24 * time.Hour)
		if today.After(lastResetDay) {
			u.updatesToday.Store(0)
			u.candlesSavedToday.Store(0)
			lastResetDay = today
			cycleNum = 0
			log.Println("🔴 LIVE — 🌅 Nuevo día, contadores reseteados")
		}

		// ── Verificar presupuesto de créditos ──
		if u.provider.CreditosUsadosHoy() >= u.config.MaxCreditsPerDay {
			u.mu.Lock()
			u.status.NextUpdate = "⏸️ Límite de créditos alcanzado, esperando nuevo día"
			u.mu.Unlock()

			if !u.waitWithStop(5 * time.Minute) {
				return
			}
			continue
		}

		// ── Verificar horario de mercado ──
		if u.config.MarketHoursOnly && !isMarketOpen() {
			u.mu.Lock()
			u.status.MarketOpen = false
			u.status.NextUpdate = "Mercado cerrado — " + proximaApertura()
			u.mu.Unlock()

			if !u.waitWithStop(30 * time.Second) {
				return
			}
			continue
		}

		// ── Obtener siguiente tarea ──
		task := tasks[taskIndex]
		taskIndex++
		if taskIndex >= len(tasks) {
			taskIndex = 0
			cycleNum++
			log.Printf("🔴 LIVE — ✅ Ciclo #%d completado | %d updates | +%d velas nuevas hoy | %d créditos usados",
				cycleNum, u.updatesToday.Load(), u.candlesSavedToday.Load(), u.provider.CreditosUsadosHoy())
		}

		// Actualizar estado
		u.mu.Lock()
		u.status.CurrentAsset = task.Asset.Symbol
		u.status.CurrentTimeframe = task.Timeframe
		u.status.MarketOpen = true
		u.status.CycleNumber = cycleNum
		u.status.UpdatesThisCycle = taskIndex
		u.mu.Unlock()

		// ── Ejecutar: obtener y guardar velas ──
		saved := u.fetchAndSave(ctx, task.Asset, task.Timeframe)
		u.updatesToday.Add(1)
		if saved > 0 {
			u.candlesSavedToday.Add(saved)
		}

		u.mu.Lock()
		u.status.LastUpdate = time.Now().Format(time.RFC3339)
		u.mu.Unlock()

		// ── Esperar intervalo antes de siguiente llamada ──
		if !u.waitWithStop(interval) {
			return
		}
	}
}

// ============================================
// Obtener y guardar velas
// ============================================

// fetchAndSave obtiene las últimas velas de un activo/timeframe y las guarda en BD
func (u *RealtimeUpdater) fetchAndSave(ctx context.Context, asset models.Asset, timeframe string) int64 {
	// Pedir solo las 2 últimas velas (1 crédito, mínima data)
	// ON CONFLICT DO NOTHING en BD maneja los duplicados
	candles, err := u.provider.FetchHistoricalCandles(ctx, asset.Symbol, timeframe, 2, "")
	if err != nil {
		u.addError(fmt.Sprintf("[%s/%s] %v", asset.Symbol, timeframe, err))
		return 0
	}

	if len(candles) == 0 {
		return 0
	}

	saved, err := u.db.SaveCandlesBatch(ctx, asset.ID, timeframe, candles)
	if err != nil {
		u.addError(fmt.Sprintf("[%s/%s] Error guardando: %v", asset.Symbol, timeframe, err))
		return 0
	}

	if saved > 0 {
		log.Printf("🔴 LIVE [%s/%s] +%d vela(s) nueva(s)", asset.Symbol, timeframe, saved)
	}

	return saved
}

// ============================================
// Constructor de cola de tareas ponderada
// ============================================

// buildTaskQueue construye una cola de tareas con prioridad ponderada
// Los timeframes con mayor peso se repiten más veces en el ciclo
// Ejemplo con 5min×2, 1h×1 y 3 activos [A,B,C]:
//
//	A/5min, A/1h, A/5min, B/5min, B/1h, B/5min, C/5min, C/1h, C/5min
//
// Esto garantiza que 5min se actualiza 2x más frecuente que 1h
func (u *RealtimeUpdater) buildTaskQueue(assets []models.Asset) []updateTask {
	// Determinar el peso máximo
	maxWeight := 0
	for _, w := range u.config.Weights {
		if w > maxWeight {
			maxWeight = w
		}
	}

	var tasks []updateTask

	// Por cada activo, intercalar timeframes con sus pesos
	// Ronda 0: todos los TF con peso >= 1
	// Ronda 1: solo TF con peso >= 2
	// etc.
	for _, asset := range assets {
		for round := 0; round < maxWeight; round++ {
			for _, tf := range u.config.Timeframes {
				w := u.config.Weights[tf]
				if w == 0 {
					w = 1
				}
				if w > round {
					tasks = append(tasks, updateTask{
						Asset:     asset,
						Timeframe: tf,
					})
				}
			}
		}
	}

	return tasks
}

// ============================================
// Helpers: Horario de mercado
// ============================================

// isMarketOpen verifica si el mercado de EE.UU. está abierto
// NYSE/NASDAQ: 9:30 AM - 4:00 PM Eastern Time, lunes a viernes
// Extendemos ±15 min para capturar velas de apertura/cierre
func isMarketOpen() bool {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return true // Fallback: asumir abierto si no hay timezone
	}

	now := time.Now().In(loc)
	weekday := now.Weekday()

	// Fin de semana → cerrado
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	// Horario: 9:15 - 16:15 ET (±15 min de margen)
	minuteOfDay := now.Hour()*60 + now.Minute()
	marketOpen := 9*60 + 15   // 9:15 (15 min antes de apertura)
	marketClose := 16*60 + 15 // 16:15 (15 min después de cierre)

	return minuteOfDay >= marketOpen && minuteOfDay <= marketClose
}

// proximaApertura devuelve texto descriptivo de la próxima apertura
func proximaApertura() string {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return "próximo día hábil 9:30 ET"
	}

	now := time.Now().In(loc)
	next := now

	// Si ya pasó la apertura de hoy, ir al siguiente día
	openTime := time.Date(next.Year(), next.Month(), next.Day(), 9, 30, 0, 0, loc)
	if now.After(openTime) {
		next = next.AddDate(0, 0, 1)
	}

	// Saltar fines de semana
	for next.Weekday() == time.Saturday || next.Weekday() == time.Sunday {
		next = next.AddDate(0, 0, 1)
	}

	nextOpen := time.Date(next.Year(), next.Month(), next.Day(), 9, 30, 0, 0, loc)
	duration := time.Until(nextOpen)

	if duration.Hours() > 24 {
		return fmt.Sprintf("Abre %s (~%.0fh)", nextOpen.Format("Mon 2 Jan 15:04 ET"), duration.Hours())
	}
	if duration.Hours() > 1 {
		hours := int(duration.Hours())
		mins := int(duration.Minutes()) - hours*60
		return fmt.Sprintf("Abre hoy a las 9:30 ET (en %dh %dmin)", hours, mins)
	}
	return fmt.Sprintf("Abre en %.0f min", duration.Minutes())
}

// waitWithStop espera una duración pero puede cancelarse con stop
// Retorna true si el wait completó normalmente, false si fue cancelado
func (u *RealtimeUpdater) waitWithStop(d time.Duration) bool {
	select {
	case <-u.stopCh:
		return false
	case <-time.After(d):
		return true
	}
}

// addError agrega un error al historial (máximo 30)
func (u *RealtimeUpdater) addError(err string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.status.Errors = append(u.status.Errors, time.Now().Format("15:04:05")+" — "+err)
	if len(u.status.Errors) > 30 {
		u.status.Errors = u.status.Errors[len(u.status.Errors)-30:]
	}
}
