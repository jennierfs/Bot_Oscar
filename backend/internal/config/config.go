// ============================================
// Bot Oscar - Configuración
// Lee las variables de entorno y establece valores por defecto
// ============================================
package config

import (
	"os"
	"strconv"
)

// Config contiene toda la configuración del bot
type Config struct {
	// Servidor
	Port string

	// PostgreSQL
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Redis
	RedisHost string
	RedisPort string

	// APIs externas
	AlphaVantageAPIKey string
	DeepSeekAPIKey     string

	// Trading - valores por defecto que se sobreescriben desde la BD
	RiskPerTrade     float64 // Porcentaje de riesgo por operación
	RiskRewardRatio  float64 // Ratio riesgo/beneficio mínimo
	MaxOpenTrades    int     // Máximo de operaciones simultáneas
	AnalysisInterval int     // Segundos entre análisis
	MinSignalScore   int     // Puntuación mínima para operar (0-100)
}

// Load carga la configuración desde variables de entorno
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "bot_oscar"),
		DBPassword:         getEnv("DB_PASSWORD", "bot_oscar_2026"),
		DBName:             getEnv("DB_NAME", "bot_oscar_db"),
		RedisHost:          getEnv("REDIS_HOST", "localhost"),
		RedisPort:          getEnv("REDIS_PORT", "6379"),
		AlphaVantageAPIKey: getEnv("ALPHA_VANTAGE_API_KEY", "demo"),
		DeepSeekAPIKey:     getEnv("DEEPSEEK_API_KEY", ""),
		RiskPerTrade:       getEnvFloat("RISK_PER_TRADE", 2.0),
		RiskRewardRatio:    getEnvFloat("RISK_REWARD_RATIO", 2.0),
		MaxOpenTrades:      getEnvInt("MAX_OPEN_TRADES", 5),
		AnalysisInterval:   getEnvInt("ANALYSIS_INTERVAL", 60),
		MinSignalScore:     getEnvInt("MIN_SIGNAL_SCORE", 65),
	}
}

// getEnv obtiene una variable de entorno o devuelve el valor por defecto
func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

// getEnvInt obtiene una variable de entorno como entero
func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

// getEnvFloat obtiene una variable de entorno como flotante
func getEnvFloat(key string, defaultVal float64) float64 {
	if val, ok := os.LookupEnv(key); ok {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			return floatVal
		}
	}
	return defaultVal
}
