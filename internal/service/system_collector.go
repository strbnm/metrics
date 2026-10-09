package service

import (
	"log"
	"strconv"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"

	models "github.com/strbnm/metrics/internal/model"
)

func collectSystemMetrics() []models.Metrics {
	metrics := make([]models.Metrics, 0)

	// Память
	vm, err := mem.VirtualMemory()
	if err == nil {
		metrics = append(metrics,
			models.Metrics{ID: "TotalMemory", MType: models.Gauge, Value: ptr(float64(vm.Total))},
			models.Metrics{ID: "FreeMemory", MType: models.Gauge, Value: ptr(float64(vm.Free))},
		)
	} else {
		log.Printf("failed to get virtual memory: %v", err)
	}

	// CPU по ядрам
	percentages, err := cpu.Percent(0, true)
	if err == nil && len(percentages) > 0 {
		for i, p := range percentages {
			id := "CPUUtilization"
			if len(percentages) > 1 {
				id += strconv.Itoa(i)
			}
			metrics = append(metrics, models.Metrics{
				ID: id, MType: models.Gauge, Value: ptr(p),
			})
		}
	} else if err != nil {
		log.Printf("failed to get CPU utilization: %v", err)
	}

	return metrics
}
