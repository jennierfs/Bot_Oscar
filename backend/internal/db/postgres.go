// ============================================
// Bot Oscar - Capa de Base de Datos (PostgreSQL)
// Maneja todas las consultas y operaciones CRUD
// ============================================
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"bot-oscar/internal/config"
	"bot-oscar/internal/models"
)

// Database encapsula el pool de conexiones a PostgreSQL
type Database struct {
	Pool *pgxpool.Pool
}

// Connect crea un pool de conexiones a PostgreSQL
func Connect(cfg *config.Config) (*Database, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	poolCfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("error parseando configuración de BD: %w", err)
	}

	// Configurar pool de conexiones
	poolCfg.MaxConns = 10
	poolCfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("error creando pool de conexiones: %w", err)
	}

	// Verificar conexión
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("error verificando conexión a BD: %w", err)
	}

	return &Database{Pool: pool}, nil
}

// Close cierra el pool de conexiones
func (d *Database) Close() {
	d.Pool.Close()
}

// ============================================
// Operaciones con Activos
// ============================================

// GetActiveAssets obtiene todos los activos activos
func (d *Database) GetActiveAssets(ctx context.Context) ([]models.Asset, error) {
	query := `SELECT id, simbolo, nombre, tipo, activo, creado_en 
              FROM activos WHERE activo = true ORDER BY tipo, nombre`

	rows, err := d.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error consultando activos: %w", err)
	}
	defer rows.Close()

	assets := make([]models.Asset, 0)
	for rows.Next() {
		var a models.Asset
		err := rows.Scan(&a.ID, &a.Symbol, &a.Name, &a.Type, &a.Active, &a.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("error escaneando activo: %w", err)
		}
		assets = append(assets, a)
	}

	return assets, nil
}

// ============================================
// Operaciones con Precios
// ============================================

// GetPrices obtiene los últimos N precios de un activo (deduplicado por fecha)
func (d *Database) GetPrices(ctx context.Context, assetID int, limit int) ([]models.Price, error) {
	// Usar DISTINCT ON para obtener solo un registro por fecha (el más reciente)
	// Esto evita duplicados cuando hay datos viejos (demo) + datos reales (Yahoo)
	query := `SELECT DISTINCT ON (fecha::date) id, activo_id, apertura, maximo, minimo, cierre, volumen, fecha
              FROM precios WHERE activo_id = $1 
              AND apertura > 0 AND maximo > 0 AND minimo > 0 AND cierre > 0
              ORDER BY fecha::date DESC, id DESC LIMIT $2`

	rows, err := d.Pool.Query(ctx, query, assetID, limit)
	if err != nil {
		return nil, fmt.Errorf("error consultando precios: %w", err)
	}
	defer rows.Close()

	prices := make([]models.Price, 0)
	for rows.Next() {
		var p models.Price
		err := rows.Scan(&p.ID, &p.AssetID, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume, &p.Date)
		if err != nil {
			return nil, fmt.Errorf("error escaneando precio: %w", err)
		}
		prices = append(prices, p)
	}

	// Invertir para que estén en orden cronológico (más antiguo primero)
	for i, j := 0, len(prices)-1; i < j; i, j = i+1, j-1 {
		prices[i], prices[j] = prices[j], prices[i]
	}

	return prices, nil
}

// SavePrices guarda precios en la BD (upsert por activo_id + fecha)
func (d *Database) SavePrices(ctx context.Context, assetID int, prices []models.Price) error {
	query := `INSERT INTO precios (activo_id, apertura, maximo, minimo, cierre, volumen, fecha)
              VALUES ($1, $2, $3, $4, $5, $6, $7)
              ON CONFLICT (activo_id, fecha) DO UPDATE SET
                  apertura = EXCLUDED.apertura,
                  maximo = EXCLUDED.maximo,
                  minimo = EXCLUDED.minimo,
                  cierre = EXCLUDED.cierre,
                  volumen = EXCLUDED.volumen`

	for _, p := range prices {
		_, err := d.Pool.Exec(ctx, query, assetID, p.Open, p.High, p.Low, p.Close, p.Volume, p.Date)
		if err != nil {
			return fmt.Errorf("error guardando precio: %w", err)
		}
	}

	return nil
}

// ============================================
// Operaciones con Señales
// ============================================

// GetLatestSignals obtiene las últimas N señales con info del activo
func (d *Database) GetLatestSignals(ctx context.Context, limit int) ([]models.Signal, error) {
	query := `SELECT s.id, s.activo_id, a.simbolo, a.nombre, s.tipo, s.fuerza,
                     s.precio_entrada, s.stop_loss, s.take_profit, s.razon, s.creado_en
              FROM senales s
              JOIN activos a ON a.id = s.activo_id
              ORDER BY s.creado_en DESC LIMIT $1`

	rows, err := d.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("error consultando señales: %w", err)
	}
	defer rows.Close()

	signals := make([]models.Signal, 0)
	for rows.Next() {
		var s models.Signal
		err := rows.Scan(
			&s.ID, &s.AssetID, &s.Symbol, &s.AssetName, &s.Type, &s.Strength,
			&s.EntryPrice, &s.StopLoss, &s.TakeProfit, &s.Reason, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error escaneando señal: %w", err)
		}
		signals = append(signals, s)
	}

	return signals, nil
}

// SaveSignal guarda una señal en la BD
func (d *Database) SaveSignal(ctx context.Context, s *models.Signal) error {
	query := `INSERT INTO senales (activo_id, tipo, fuerza, precio_entrada, stop_loss, take_profit, razon)
              VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	return d.Pool.QueryRow(ctx, query,
		s.AssetID, s.Type, s.Strength, s.EntryPrice, s.StopLoss, s.TakeProfit, s.Reason,
	).Scan(&s.ID)
}

// ============================================
// Operaciones con Trades
// ============================================

// GetOperations obtiene operaciones filtradas por estado (vacío = todas)
func (d *Database) GetOperations(ctx context.Context, status string) ([]models.Operation, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `SELECT o.id, o.activo_id, a.simbolo, a.nombre, o.tipo, o.precio_entrada,
                        o.precio_salida, o.cantidad, o.stop_loss, o.take_profit, o.estado,
                        o.ganancia_perdida, o.razon_entrada, o.razon_salida, o.abierta_en, o.cerrada_en
                 FROM operaciones o
                 JOIN activos a ON a.id = o.activo_id
                 WHERE o.estado = $1
                 ORDER BY o.abierta_en DESC`
		args = []interface{}{status}
	} else {
		query = `SELECT o.id, o.activo_id, a.simbolo, a.nombre, o.tipo, o.precio_entrada,
                        o.precio_salida, o.cantidad, o.stop_loss, o.take_profit, o.estado,
                        o.ganancia_perdida, o.razon_entrada, o.razon_salida, o.abierta_en, o.cerrada_en
                 FROM operaciones o
                 JOIN activos a ON a.id = o.activo_id
                 ORDER BY o.abierta_en DESC`
	}

	rows, err := d.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error consultando operaciones: %w", err)
	}
	defer rows.Close()

	operations := make([]models.Operation, 0)
	for rows.Next() {
		var op models.Operation
		err := rows.Scan(
			&op.ID, &op.AssetID, &op.Symbol, &op.AssetName, &op.Type, &op.EntryPrice,
			&op.ExitPrice, &op.Quantity, &op.StopLoss, &op.TakeProfit, &op.Status,
			&op.PnL, &op.EntryReason, &op.ExitReason, &op.OpenedAt, &op.ClosedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error escaneando operación: %w", err)
		}
		operations = append(operations, op)
	}

	return operations, nil
}

// SaveOperation guarda una nueva operación
func (d *Database) SaveOperation(ctx context.Context, op *models.Operation) error {
	query := `INSERT INTO operaciones (activo_id, tipo, precio_entrada, cantidad, stop_loss, take_profit, estado, razon_entrada)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`

	return d.Pool.QueryRow(ctx, query,
		op.AssetID, op.Type, op.EntryPrice, op.Quantity, op.StopLoss, op.TakeProfit, op.Status, op.EntryReason,
	).Scan(&op.ID)
}

// ============================================
// Configuración del Bot
// ============================================

// GetAllConfig obtiene toda la configuración como mapa clave-valor
func (d *Database) GetAllConfig(ctx context.Context) (map[string]string, error) {
	query := `SELECT clave, valor FROM configuracion`

	rows, err := d.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error consultando configuración: %w", err)
	}
	defer rows.Close()

	cfg := make(map[string]string)
	for rows.Next() {
		var key, val string
		if err := rows.Scan(&key, &val); err != nil {
			return nil, fmt.Errorf("error escaneando configuración: %w", err)
		}
		cfg[key] = val
	}

	return cfg, nil
}

// UpdateConfig actualiza un valor de configuración
func (d *Database) UpdateConfig(ctx context.Context, key, value string) error {
	query := `UPDATE configuracion SET valor = $1, actualizado_en = CURRENT_TIMESTAMP WHERE clave = $2`
	_, err := d.Pool.Exec(ctx, query, value, key)
	if err != nil {
		return fmt.Errorf("error actualizando configuración: %w", err)
	}
	return nil
}

// ============================================
// Portafolio
// ============================================

// GetPortfolioSummary calcula el resumen del portafolio
func (d *Database) GetPortfolioSummary(ctx context.Context) (*models.Portfolio, error) {
	// Obtener capital inicial desde configuración
	var capitalStr string
	err := d.Pool.QueryRow(ctx, "SELECT valor FROM configuracion WHERE clave = 'capital_inicial'").Scan(&capitalStr)
	initialCapital := 10000.0
	if err == nil {
		fmt.Sscanf(capitalStr, "%f", &initialCapital)
	}

	// Sumar ganancias/pérdidas de operaciones cerradas
	var totalPnL float64
	err = d.Pool.QueryRow(ctx,
		"SELECT COALESCE(SUM(ganancia_perdida), 0) FROM operaciones WHERE estado = 'CERRADA'",
	).Scan(&totalPnL)
	if err != nil {
		totalPnL = 0
	}

	// Contar operaciones abiertas y totales
	var openCount, totalCount int
	d.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM operaciones WHERE estado = 'ABIERTA'").Scan(&openCount)
	d.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM operaciones").Scan(&totalCount)

	// Obtener operaciones abiertas
	openOps, _ := d.GetOperations(ctx, "ABIERTA")

	capital := initialCapital + totalPnL
	returnPct := 0.0
	if initialCapital > 0 {
		returnPct = (totalPnL / initialCapital) * 100
	}

	return &models.Portfolio{
		Capital:             capital,
		InitialCapital:      initialCapital,
		PnL:                 totalPnL,
		ReturnPercent:       returnPct,
		OpenOperationsCount: openCount,
		TotalOperations:     totalCount,
		Operations:          openOps,
	}, nil
}
