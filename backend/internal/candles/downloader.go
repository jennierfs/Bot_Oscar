// ============================================
// Bot Oscar - Descargador de Velas Históricas
// Descarga y almacena velas OHLCV desde Twelve Data
// para los 20 activos en 6 timeframes distintos
//
// Timeframes soportados:
//   - 5min, 15min, 30min (intraday)
//   - 1h, 4h (swing)
//   - 1day (diario)
//
// Organización en BD:
//
//	Tabla "velas" con UNIQUE(activo_id, timeframe, fecha)
//	→ Nunca se mezclan velas de distintos activos ni timeframes
//
// Paginación:
//
//	Twelve Data devuelve máximo 5000 velas por llamada.
//	El descargador pagina hacia atrás (más reciente → más antiguo)
//	hasta llegar a 2010 o quedarse sin datos.
//
// Rate limit:
//
//	Usa el mismo TwelveDataProvider con su rate limiter
//	integrado (8 créditos/min, 800/día).
//
// ============================================
package candles

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"bot-oscar/internal/db"
	"bot-oscar/internal/market"
	"bot-oscar/internal/models"
)

// Timeframes disponibles para descarga (de mayor a menor prioridad)
// Empezamos por 1day que es el más útil y tiene más historia
var Timeframes = []string{"1day", "4h", "1h", "30min", "15min", "5min"}

// CandleDownloader descarga y almacena velas históricas desde Twelve Data
type CandleDownloader struct {
	provider *market.TwelveDataProvider
	db       *db.Database

	mu       sync.Mutex
	running  bool
	progress DownloadProgress
}

// DownloadProgress estado actual de la descarga
type DownloadProgress struct {
	Running      bool     `json:"running"`
	CurrentAsset string   `json:"currentAsset"`
	CurrentTF    string   `json:"currentTimeframe"`
	TotalAssets  int      `json:"totalAssets"`
	DoneAssets   int      `json:"doneAssets"`
	TotalCandles int64    `json:"totalCandles"`
	Errors       []string `json:"errors"`
	StartedAt    string   `json:"startedAt,omitempty"`
	FinishedAt   string   `json:"finishedAt,omitempty"`
}

// NewCandleDownloader crea un nuevo descargador de velas
func NewCandleDownloader(provider *market.TwelveDataProvider, database *db.Database) *CandleDownloader {
	return &CandleDownloader{
		provider: provider,
		db:       database,
	}
}

// IsRunning indica si hay una descarga en progreso
func (d *CandleDownloader) IsRunning() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.running
}

// GetProgress devuelve el estado actual de la descarga
func (d *CandleDownloader) GetProgress() DownloadProgress {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.progress
}

// Start inicia la descarga de velas en segundo plano (goroutine)
func (d *CandleDownloader) Start() error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("ya hay una descarga en progreso")
	}
	d.running = true
	d.progress = DownloadProgress{
		Running:   true,
		StartedAt: time.Now().Format(time.RFC3339),
	}
	d.mu.Unlock()

	go d.run()
	return nil
}

// run ejecuta la descarga completa de todos los activos × timeframes
func (d *CandleDownloader) run() {
	ctx := context.Background()
	defer func() {
		d.mu.Lock()
		d.running = false
		d.progress.Running = false
		d.progress.FinishedAt = time.Now().Format(time.RFC3339)
		d.mu.Unlock()
		log.Printf("✅ Descarga de velas completada — Total: %d velas, Errores: %d",
			d.progress.TotalCandles, len(d.progress.Errors))
	}()

	// Asegurar que la tabla velas existe
	if err := d.db.EnsureCandlesTable(ctx); err != nil {
		d.addError("Error creando tabla velas: " + err.Error())
		return
	}

	// Obtener todos los activos activos
	assets, err := d.db.GetActiveAssets(ctx)
	if err != nil {
		d.addError("Error obteniendo activos: " + err.Error())
		return
	}

	d.mu.Lock()
	d.progress.TotalAssets = len(assets)
	d.mu.Unlock()

	log.Printf("📥 Iniciando descarga de velas: %d activos × %d timeframes = %d combinaciones",
		len(assets), len(Timeframes), len(assets)*len(Timeframes))

	// Iterar cada activo y cada timeframe
	for _, asset := range assets {
		for _, tf := range Timeframes {
			d.mu.Lock()
			d.progress.CurrentAsset = asset.Symbol
			d.progress.CurrentTF = tf
			d.mu.Unlock()

			saved, err := d.downloadAssetTimeframe(ctx, asset, tf)
			if err != nil {
				errMsg := fmt.Sprintf("[%s/%s] %v", asset.Symbol, tf, err)
				d.addError(errMsg)
				log.Printf("⚠️ %s", errMsg)
				continue
			}

			d.mu.Lock()
			d.progress.TotalCandles += saved
			d.mu.Unlock()

			if saved > 0 {
				log.Printf("✅ [%s/%s] %d velas nuevas guardadas", asset.Symbol, tf, saved)
			} else {
				log.Printf("📦 [%s/%s] Sin velas nuevas (ya actualizadas)", asset.Symbol, tf)
			}
		}

		d.mu.Lock()
		d.progress.DoneAssets++
		d.mu.Unlock()
		log.Printf("📊 Progreso: %d/%d activos completados", d.progress.DoneAssets, d.progress.TotalAssets)
	}
}

// downloadAssetTimeframe descarga todas las velas de un activo en un timeframe
// Si ya hay datos en BD, solo descarga las nuevas (incremental)
// Si no hay datos, pagina hacia atrás hasta 2010 o fin de datos
func (d *CandleDownloader) downloadAssetTimeframe(ctx context.Context, asset models.Asset, timeframe string) (int64, error) {
	var totalSaved int64

	// Verificar si ya tenemos datos para este activo/timeframe
	latest, err := d.db.GetLatestCandleDate(ctx, asset.ID, timeframe)
	if err == nil && latest != nil {
		// Ya hay datos → solo descargar las nuevas (desde la última fecha)
		log.Printf("📊 [%s/%s] Actualizando desde %s", asset.Symbol, timeframe, latest.Format("2006-01-02 15:04"))

		candles, err := d.provider.FetchHistoricalCandles(ctx, asset.Symbol, timeframe, 5000, "")
		if err != nil {
			return 0, err
		}

		if len(candles) > 0 {
			saved, err := d.db.SaveCandlesBatch(ctx, asset.ID, timeframe, candles)
			if err != nil {
				return 0, err
			}
			return saved, nil
		}
		return 0, nil
	}

	// No hay datos previos → descarga completa paginando hacia atrás
	endDate := "" // Empezar desde la fecha más reciente

	for {
		candles, err := d.provider.FetchHistoricalCandles(ctx, asset.Symbol, timeframe, 5000, endDate)
		if err != nil {
			if totalSaved > 0 {
				// Ya guardamos algo, no es error fatal
				log.Printf("⚠️ [%s/%s] Paginación detenida: %v (guardadas hasta ahora: %d)",
					asset.Symbol, timeframe, err, totalSaved)
				return totalSaved, nil
			}
			return 0, err
		}

		if len(candles) == 0 {
			break
		}

		// Guardar en BD inmediatamente (no acumular en memoria)
		saved, err := d.db.SaveCandlesBatch(ctx, asset.ID, timeframe, candles)
		if err != nil {
			return totalSaved, fmt.Errorf("error guardando velas: %w", err)
		}
		totalSaved += saved

		// La vela más antigua de este lote
		oldest := candles[0].Date

		// Si llegamos antes de 2010, terminamos
		if oldest.Year() < 2010 {
			break
		}

		// Si recibimos menos de ~5000, ya no hay más datos
		if len(candles) < 4900 {
			break
		}

		// Preparar siguiente página: antes de la vela más antigua
		endDate = oldest.Add(-time.Second).Format("2006-01-02 15:04:05")

		log.Printf("📥 [%s/%s] Página: %d velas (total: %d), siguiente antes de %s",
			asset.Symbol, timeframe, len(candles), totalSaved, oldest.Format("2006-01-02"))
	}

	return totalSaved, nil
}

// addError agrega un error al progreso (máximo 50 errores almacenados)
func (d *CandleDownloader) addError(err string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.progress.Errors = append(d.progress.Errors, err)
	if len(d.progress.Errors) > 50 {
		d.progress.Errors = d.progress.Errors[len(d.progress.Errors)-50:]
	}
}
