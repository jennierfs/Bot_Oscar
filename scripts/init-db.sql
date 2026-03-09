-- ============================================
-- Bot Oscar - Inicialización de Base de Datos
-- Tablas, índices y datos iniciales
-- ============================================

-- Tabla de activos monitoreados (commodities y acciones de defensa)
CREATE TABLE IF NOT EXISTS activos (
    id SERIAL PRIMARY KEY,
    simbolo VARCHAR(20) NOT NULL UNIQUE,
    nombre VARCHAR(100) NOT NULL,
    tipo VARCHAR(20) NOT NULL,          -- 'commodity', 'accion'
    activo BOOLEAN DEFAULT true,
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de precios históricos (OHLCV)
CREATE TABLE IF NOT EXISTS precios (
    id SERIAL PRIMARY KEY,
    activo_id INTEGER REFERENCES activos(id) ON DELETE CASCADE,
    apertura DECIMAL(15,4),
    maximo DECIMAL(15,4),
    minimo DECIMAL(15,4),
    cierre DECIMAL(15,4),
    volumen BIGINT,
    fecha TIMESTAMP NOT NULL,
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(activo_id, fecha)
);

-- Tabla de señales generadas por el motor de análisis
CREATE TABLE IF NOT EXISTS senales (
    id SERIAL PRIMARY KEY,
    activo_id INTEGER REFERENCES activos(id) ON DELETE CASCADE,
    tipo VARCHAR(10) NOT NULL,          -- 'COMPRA', 'VENTA', 'MANTENER'
    fuerza INTEGER NOT NULL,            -- Puntuación 0-100
    precio_entrada DECIMAL(15,4),
    stop_loss DECIMAL(15,4),
    take_profit DECIMAL(15,4),
    razon TEXT,                          -- Razón legible de la señal
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de operaciones (trades ejecutados)
CREATE TABLE IF NOT EXISTS operaciones (
    id SERIAL PRIMARY KEY,
    activo_id INTEGER REFERENCES activos(id) ON DELETE CASCADE,
    tipo VARCHAR(10) NOT NULL,          -- 'COMPRA', 'VENTA'
    precio_entrada DECIMAL(15,4) NOT NULL,
    precio_salida DECIMAL(15,4),
    cantidad DECIMAL(15,6) NOT NULL,
    stop_loss DECIMAL(15,4),
    take_profit DECIMAL(15,4),
    estado VARCHAR(20) DEFAULT 'ABIERTA', -- 'ABIERTA', 'CERRADA', 'CANCELADA'
    ganancia_perdida DECIMAL(15,4),
    razon_entrada TEXT,
    razon_salida TEXT,
    abierta_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    cerrada_en TIMESTAMP
);

-- Tabla de configuración del bot
CREATE TABLE IF NOT EXISTS configuracion (
    id SERIAL PRIMARY KEY,
    clave VARCHAR(50) NOT NULL UNIQUE,
    valor TEXT NOT NULL,
    descripcion TEXT,
    actualizado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- Índices para rendimiento de consultas
-- ============================================
CREATE INDEX IF NOT EXISTS idx_precios_activo_fecha ON precios(activo_id, fecha DESC);
CREATE INDEX IF NOT EXISTS idx_senales_activo_fecha ON senales(activo_id, creado_en DESC);
CREATE INDEX IF NOT EXISTS idx_operaciones_estado ON operaciones(estado);
CREATE INDEX IF NOT EXISTS idx_operaciones_activo ON operaciones(activo_id);

-- ============================================
-- Datos iniciales: Activos a monitorear
-- ============================================
INSERT INTO activos (simbolo, nombre, tipo) VALUES
    -- Commodities (Materias Primas)
    ('GC=F', 'Oro (Gold)', 'commodity'),
    ('SI=F', 'Plata (Silver)', 'commodity'),
    ('CL=F', 'Petróleo Crudo (Crude Oil)', 'commodity'),
    ('NG=F', 'Gas Natural', 'commodity'),
    -- Contratistas Principales de Defensa
    ('LMT', 'Lockheed Martin', 'accion'),
    ('RTX', 'Raytheon Technologies', 'accion'),
    ('NOC', 'Northrop Grumman', 'accion'),
    ('GD', 'General Dynamics', 'accion'),
    ('BA', 'Boeing', 'accion'),
    ('LHX', 'L3Harris Technologies', 'accion'),
    ('HII', 'Huntington Ingalls Industries', 'accion'),
    -- Drones y Sistemas Autónomos
    ('KTOS', 'Kratos Defense & Security', 'accion'),
    ('AVAV', 'AeroVironment (Switchblade)', 'accion'),
    -- Tech y Ciber Militar
    ('PLTR', 'Palantir Technologies', 'accion'),
    ('LDOS', 'Leidos Holdings', 'accion'),
    ('SAIC', 'Science Applications Intl.', 'accion'),
    ('MRCY', 'Mercury Systems', 'accion'),
    -- Aeroespacial y Componentes
    ('TXT', 'Textron (Bell Helicopters)', 'accion'),
    ('HEI', 'HEICO Corporation', 'accion'),
    ('RKLB', 'Rocket Lab USA', 'accion')
ON CONFLICT (simbolo) DO NOTHING;

-- ============================================
-- Configuración inicial del bot
-- ============================================
INSERT INTO configuracion (clave, valor, descripcion) VALUES
    ('riesgo_por_operacion', '2', 'Porcentaje máximo de riesgo por operación'),
    ('ratio_riesgo_beneficio', '2', 'Ratio mínimo riesgo/beneficio (1:X)'),
    ('max_operaciones_abiertas', '5', 'Número máximo de operaciones simultáneas'),
    ('capital_inicial', '10000', 'Capital inicial en USD'),
    ('modo', 'paper', 'Modo de operación: paper (simulación) o real'),
    ('intervalo_analisis', '60', 'Segundos entre cada ciclo de análisis'),
    ('min_fuerza_senal', '65', 'Fuerza mínima de señal para ejecutar operación (0-100)')
ON CONFLICT (clave) DO NOTHING;
