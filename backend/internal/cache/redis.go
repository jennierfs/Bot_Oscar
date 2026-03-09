// ============================================
// Bot Oscar - Capa de Caché (Redis)
// Almacena precios y datos temporales para acceso rápido
// ============================================
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"bot-oscar/internal/config"
	"bot-oscar/internal/models"
)

// Cache encapsula el cliente de Redis
type Cache struct {
	Client *redis.Client
}

// Connect crea una conexión a Redis
func Connect(cfg *config.Config) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		DB:       0,
		PoolSize: 10,
	})

	// Verificar conexión
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("error conectando a Redis: %w", err)
	}

	return &Cache{Client: client}, nil
}

// Close cierra la conexión a Redis
func (c *Cache) Close() error {
	return c.Client.Close()
}

// ============================================
// Caché de Precios
// ============================================

// SetPrices guarda los precios de un activo en caché
func (c *Cache) SetPrices(ctx context.Context, symbol string, prices []models.Price, ttl time.Duration) error {
	key := fmt.Sprintf("prices:%s", symbol)

	data, err := json.Marshal(prices)
	if err != nil {
		return fmt.Errorf("error serializando precios: %w", err)
	}

	return c.Client.Set(ctx, key, data, ttl).Err()
}

// GetPrices obtiene los precios de un activo desde caché
// Retorna nil si no hay datos en caché
func (c *Cache) GetPrices(ctx context.Context, symbol string) ([]models.Price, error) {
	key := fmt.Sprintf("prices:%s", symbol)

	data, err := c.Client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No hay datos en caché, no es error
		}
		return nil, fmt.Errorf("error obteniendo precios de caché: %w", err)
	}

	var prices []models.Price
	if err := json.Unmarshal(data, &prices); err != nil {
		return nil, fmt.Errorf("error deserializando precios: %w", err)
	}

	return prices, nil
}

// ============================================
// Caché Genérico (JSON)
// ============================================

// SetJSON guarda cualquier valor serializable en caché
func (c *Cache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("error serializando datos: %w", err)
	}

	return c.Client.Set(ctx, key, data, ttl).Err()
}

// GetJSON obtiene un valor de caché y lo deserializa
func (c *Cache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := c.Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// Delete elimina una clave de caché
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.Client.Del(ctx, key).Err()
}
