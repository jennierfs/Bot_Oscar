// ============================================
// Bot Oscar - Indicador: Volume Profile
//
// El Volume Profile muestra DÓNDE se acumuló el volumen por
// nivel de precio, revelando zonas institucionales de soporte
// y resistencia que NO son visibles con indicadores clásicos.
//
// Es el mejor proxy del "libro de órdenes" usando datos OHLCV:
//   - Donde hubo MUCHO volumen = posiciones institucionales grandes
//     → esas zonas actúan como soporte/resistencia futuros
//   - Donde hubo POCO volumen = zonas de baja liquidez
//     → el precio se mueve rápido a través de ellas
//
// Métricas calculadas:
//
//	POC (Point of Control): precio con mayor volumen acumulado
//	  → el nivel más importante; soporte si el precio está encima,
//	    resistencia si está debajo
//	Value Area (VA): rango que contiene el 70% del volumen
//	  → VAH (Value Area High) y VAL (Value Area Low)
//	  → precio dentro del VA = zona de equilibrio
//	  → precio fuera del VA = movimiento direccional
//	HVN (High Volume Nodes): zonas con volumen muy alto
//	  → soportes/resistencias fuertes
//	LVN (Low Volume Nodes): zonas con volumen muy bajo
//	  → el precio las cruza rápido, gaps de liquidez
//
// Ventana: últimos 50 días, dividido en N niveles de precio
// ============================================
package indicators

import (
	"fmt"
	"math"
	"sort"
)

// VolumeProfileResult resultado completo del Volume Profile
type VolumeProfileResult struct {
	POC       float64       `json:"poc"`       // Point of Control (precio con más volumen)
	POCVolume int64         `json:"pocVolume"` // Volumen en el POC
	VAH       float64       `json:"vah"`       // Value Area High (techo del 70%)
	VAL       float64       `json:"val"`       // Value Area Low (piso del 70%)
	HVNs      []VolumeNode  `json:"hvns"`      // High Volume Nodes (soportes/resistencias fuertes)
	LVNs      []VolumeNode  `json:"lvns"`      // Low Volume Nodes (gaps de liquidez)
	Levels    []VolumeLevel `json:"levels"`    // Todos los niveles (para visualización)
	TotalVol  int64         `json:"totalVolume"`
	SummaryAI string        `json:"-"` // Resumen para DeepSeek
}

// VolumeNode representa una zona de alto o bajo volumen
type VolumeNode struct {
	PriceLow  float64 `json:"priceLow"`  // Precio bajo del rango
	PriceHigh float64 `json:"priceHigh"` // Precio alto del rango
	PriceMid  float64 `json:"priceMid"`  // Punto medio del rango
	Volume    int64   `json:"volume"`    // Volumen acumulado
	Percent   float64 `json:"percent"`   // % del volumen total
	Type      string  `json:"type"`      // "HVN" o "LVN"
}

// VolumeLevel un nivel individual del perfil de volumen
type VolumeLevel struct {
	PriceLow  float64 `json:"priceLow"`
	PriceHigh float64 `json:"priceHigh"`
	PriceMid  float64 `json:"priceMid"`
	Volume    int64   `json:"volume"`
	Percent   float64 `json:"percent"`
	IsPOC     bool    `json:"isPOC"`
	InVA      bool    `json:"inVA"` // Dentro del Value Area
}

// CalculateVolumeProfile calcula el perfil de volumen por niveles de precio
// highs, lows, closes: arrays OHLC
// volumes: array de volumen
// numLevels: número de niveles/bins de precio (recomendado: 24-40)
// lookback: número de velas a analizar (recomendado: 50)
func CalculateVolumeProfile(highs, lows, closes []float64, volumes []int64, numLevels, lookback int) *VolumeProfileResult {
	n := len(closes)
	if n < lookback || len(highs) != n || len(lows) != n || len(volumes) != n {
		return nil
	}

	if numLevels < 10 {
		numLevels = 24
	}

	// Tomar la ventana de lookback
	startIdx := n - lookback
	windowHighs := highs[startIdx:]
	windowLows := lows[startIdx:]
	windowCloses := closes[startIdx:]
	windowVolumes := volumes[startIdx:]

	// 1. Encontrar rango de precios en la ventana
	priceMax := windowHighs[0]
	priceMin := windowLows[0]
	for i := 0; i < lookback; i++ {
		if windowHighs[i] > priceMax {
			priceMax = windowHighs[i]
		}
		if windowLows[i] < priceMin {
			priceMin = windowLows[i]
		}
	}

	priceRange := priceMax - priceMin
	if priceRange <= 0 {
		return nil
	}

	// 2. Crear niveles (bins) de precio
	levelSize := priceRange / float64(numLevels)
	levels := make([]VolumeLevel, numLevels)
	for i := 0; i < numLevels; i++ {
		levels[i].PriceLow = priceMin + float64(i)*levelSize
		levels[i].PriceHigh = priceMin + float64(i+1)*levelSize
		levels[i].PriceMid = (levels[i].PriceLow + levels[i].PriceHigh) / 2.0
	}

	// 3. Distribuir volumen de cada vela en los niveles que toca
	// Cada vela distribuye su volumen proporcionalmente entre los niveles
	// que cubre su rango High-Low
	var totalVolume int64
	for i := 0; i < lookback; i++ {
		candleLow := windowLows[i]
		candleHigh := windowHighs[i]
		candleVol := windowVolumes[i]
		totalVolume += candleVol

		if candleVol == 0 || candleHigh <= candleLow {
			continue
		}

		candleRange := candleHigh - candleLow

		// Distribuir volumen proporcionalmente a cada nivel que la vela toca
		for j := 0; j < numLevels; j++ {
			// Calcular solapamiento entre la vela y este nivel
			overlapLow := math.Max(candleLow, levels[j].PriceLow)
			overlapHigh := math.Min(candleHigh, levels[j].PriceHigh)

			if overlapHigh > overlapLow {
				// Proporción del volumen de la vela que va a este nivel
				overlap := overlapHigh - overlapLow
				proportion := overlap / candleRange

				// Dar más peso si el cierre está en este nivel
				if windowCloses[i] >= levels[j].PriceLow && windowCloses[i] <= levels[j].PriceHigh {
					proportion *= 1.5 // 50% más peso al nivel del cierre
				}

				levels[j].Volume += int64(float64(candleVol) * proportion)
			}
		}
	}

	if totalVolume == 0 {
		return nil
	}

	// 4. Calcular porcentajes y encontrar POC
	pocIdx := 0
	var maxVol int64
	for i := range levels {
		levels[i].Percent = float64(levels[i].Volume) / float64(totalVolume) * 100
		if levels[i].Volume > maxVol {
			maxVol = levels[i].Volume
			pocIdx = i
		}
	}
	levels[pocIdx].IsPOC = true

	// 5. Calcular Value Area (70% del volumen)
	// Empieza desde el POC y expande hacia arriba y abajo
	targetVol := int64(float64(totalVolume) * 0.70)
	vaVolume := levels[pocIdx].Volume
	vaLow := pocIdx
	vaHigh := pocIdx

	for vaVolume < targetVol {
		expandUp := false
		expandDown := false

		// Intentar expandir hacia arriba
		if vaHigh+1 < numLevels {
			expandUp = true
		}
		// Intentar expandir hacia abajo
		if vaLow-1 >= 0 {
			expandDown = true
		}

		if !expandUp && !expandDown {
			break
		}

		if expandUp && expandDown {
			// Expandir hacia el lado con más volumen
			if levels[vaHigh+1].Volume >= levels[vaLow-1].Volume {
				vaHigh++
				vaVolume += levels[vaHigh].Volume
			} else {
				vaLow--
				vaVolume += levels[vaLow].Volume
			}
		} else if expandUp {
			vaHigh++
			vaVolume += levels[vaHigh].Volume
		} else {
			vaLow--
			vaVolume += levels[vaLow].Volume
		}
	}

	// Marcar niveles dentro del Value Area
	for i := vaLow; i <= vaHigh; i++ {
		levels[i].InVA = true
	}

	// 6. Identificar HVN y LVN
	avgVol := totalVolume / int64(numLevels)
	hvnThreshold := float64(avgVol) * 1.5 // >150% del promedio = HVN
	lvnThreshold := float64(avgVol) * 0.4 // <40% del promedio = LVN

	hvns := make([]VolumeNode, 0)
	lvns := make([]VolumeNode, 0)

	for _, lvl := range levels {
		if float64(lvl.Volume) >= hvnThreshold {
			hvns = append(hvns, VolumeNode{
				PriceLow:  lvl.PriceLow,
				PriceHigh: lvl.PriceHigh,
				PriceMid:  lvl.PriceMid,
				Volume:    lvl.Volume,
				Percent:   lvl.Percent,
				Type:      "HVN",
			})
		} else if float64(lvl.Volume) <= lvnThreshold && lvl.Volume > 0 {
			lvns = append(lvns, VolumeNode{
				PriceLow:  lvl.PriceLow,
				PriceHigh: lvl.PriceHigh,
				PriceMid:  lvl.PriceMid,
				Volume:    lvl.Volume,
				Percent:   lvl.Percent,
				Type:      "LVN",
			})
		}
	}

	// Ordenar HVNs por volumen descendente
	sort.Slice(hvns, func(i, j int) bool {
		return hvns[i].Volume > hvns[j].Volume
	})

	// Limitar a los top 5 HVNs y LVNs
	if len(hvns) > 5 {
		hvns = hvns[:5]
	}
	if len(lvns) > 5 {
		lvns = lvns[:5]
	}

	result := &VolumeProfileResult{
		POC:       levels[pocIdx].PriceMid,
		POCVolume: levels[pocIdx].Volume,
		VAH:       levels[vaHigh].PriceHigh,
		VAL:       levels[vaLow].PriceLow,
		HVNs:      hvns,
		LVNs:      lvns,
		Levels:    levels,
		TotalVol:  totalVolume,
	}

	// 7. Generar resumen para DeepSeek
	result.SummaryAI = buildVolumeProfileSummary(result, closes[n-1])

	return result
}

// buildVolumeProfileSummary genera resumen textual para el prompt de IA
func buildVolumeProfileSummary(vp *VolumeProfileResult, currentPrice float64) string {
	// Posición del precio respecto al Volume Profile
	priceVsPOC := "EN"
	if currentPrice > vp.POC*1.005 {
		priceVsPOC = "ENCIMA del"
	} else if currentPrice < vp.POC*0.995 {
		priceVsPOC = "DEBAJO del"
	}

	priceVsVA := "DENTRO del Value Area (zona de equilibrio)"
	if currentPrice > vp.VAH {
		priceVsVA = "ENCIMA del Value Area → breakout alcista, buscar compras"
	} else if currentPrice < vp.VAL {
		priceVsVA = "DEBAJO del Value Area → breakdown bajista, buscar ventas"
	}

	summary := fmt.Sprintf("Volume Profile (50 días, %d niveles):\n", len(vp.Levels))
	summary += fmt.Sprintf("  POC (Point of Control): $%.2f (nivel con más volumen acumulado)\n", vp.POC)
	summary += fmt.Sprintf("  → Precio %s POC ($%.2f vs $%.2f)\n", priceVsPOC, currentPrice, vp.POC)
	summary += fmt.Sprintf("  Value Area (70%% del volumen): $%.2f - $%.2f\n", vp.VAL, vp.VAH)
	summary += fmt.Sprintf("  → Precio %s\n", priceVsVA)

	if len(vp.HVNs) > 0 {
		summary += fmt.Sprintf("  High Volume Nodes (soportes/resistencias fuertes): %d zonas\n", len(vp.HVNs))
		for i, hvn := range vp.HVNs {
			role := "soporte"
			if hvn.PriceMid > currentPrice {
				role = "resistencia"
			}
			summary += fmt.Sprintf("    %d. $%.2f-$%.2f (%.1f%% del vol total) → actúa como %s\n",
				i+1, hvn.PriceLow, hvn.PriceHigh, hvn.Percent, role)
		}
	}

	if len(vp.LVNs) > 0 {
		summary += fmt.Sprintf("  Low Volume Nodes (gaps de liquidez): %d zonas\n", len(vp.LVNs))
		for i, lvn := range vp.LVNs {
			if i >= 3 { // Solo mostrar top 3 LVNs
				break
			}
			summary += fmt.Sprintf("    %d. $%.2f-$%.2f (%.1f%% vol) → zona de baja liquidez, precio se mueve rápido aquí\n",
				i+1, lvn.PriceLow, lvn.PriceHigh, lvn.Percent)
		}
	}

	// Conclusión contextual
	if currentPrice > vp.VAH {
		summary += "  ✅ CONCLUSIÓN: Precio sobre Value Area → momentum ALCISTA. El POC actúa como soporte si hay retroceso.\n"
	} else if currentPrice < vp.VAL {
		summary += "  ✅ CONCLUSIÓN: Precio bajo Value Area → momentum BAJISTA. El POC actúa como resistencia si hay rebote.\n"
	} else if math.Abs(currentPrice-vp.POC)/vp.POC < 0.01 {
		summary += "  ⚠️ CONCLUSIÓN: Precio en el POC → zona de máxima indecisión. Esperar breakout del VA para dirección.\n"
	} else if currentPrice > vp.POC {
		summary += "  ✅ CONCLUSIÓN: Precio entre POC y VAH → sesgo ALCISTA moderado dentro de zona de equilibrio.\n"
	} else {
		summary += "  ⚠️ CONCLUSIÓN: Precio entre VAL y POC → sesgo BAJISTA moderado dentro de zona de equilibrio.\n"
	}

	return summary
}
