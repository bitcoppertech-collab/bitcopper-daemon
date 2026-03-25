package main

import (
	"math/rand"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func getTemperature() float64 {
	switch runtime.GOOS {
	case "darwin":
		return getMacTemp()
	case "linux":
		return getLinuxTemp()
	case "windows":
		return getWindowsTemp()
	default:
		return estimateTemp()
	}
}

func getMacTemp() float64 {
	out, err := exec.Command("sudo", "-n", "powermetrics", "-n", "1", "-i", "100",
		"--samplers", "thermal").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "CPU die temperature") {
				parts := strings.Fields(line)
				for _, p := range parts {
					if t, err := strconv.ParseFloat(p, 64); err == nil && t > 20 && t < 120 {
						return t
					}
				}
			}
		}
	}
	return estimateTemp()
}

func getLinuxTemp() float64 {
	out, err := exec.Command("sensors", "-u").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "temp1_input") || strings.Contains(line, "Core 0") {
				parts := strings.Fields(line)
				for _, p := range parts {
					if t, err := strconv.ParseFloat(p, 64); err == nil && t > 20 && t < 120 {
						return t
					}
				}
			}
		}
	}
	out2, err2 := exec.Command("cat", "/sys/class/thermal/thermal_zone0/temp").Output()
	if err2 == nil {
		if t, err := strconv.ParseFloat(strings.TrimSpace(string(out2)), 64); err == nil {
			return t / 1000.0
		}
	}
	return estimateTemp()
}

func getWindowsTemp() float64 {
	// Método 1: WMI MSAcpi (Windows 10/11 con drivers correctos)
	out, err := exec.Command("powershell", "-NonInteractive", "-Command",
		"(Get-WmiObject MSAcpi_ThermalZoneTemperature -Namespace root/wmi -ErrorAction SilentlyContinue | Select-Object -First 1).CurrentTemperature").Output()
	if err == nil {
		s := strings.TrimSpace(string(out))
		if t, err := strconv.ParseFloat(s, 64); err == nil && t > 2000 {
			return (t / 10.0) - 273.15
		}
	}

	// Método 2: OpenHardwareMonitor via WMI (si está instalado)
	out2, err2 := exec.Command("powershell", "-NonInteractive", "-Command",
		"(Get-WmiObject -Namespace root/OpenHardwareMonitor -Class Sensor -ErrorAction SilentlyContinue | Where-Object {$_.SensorType -eq 'Temperature' -and $_.Name -like '*CPU*'} | Select-Object -First 1).Value").Output()
	if err2 == nil {
		s := strings.TrimSpace(string(out2))
		if t, err := strconv.ParseFloat(s, 64); err == nil && t > 20 && t < 120 {
			return t
		}
	}

	// Método 3: WMIC CPU LoadPercentage para estimar
	out3, err3 := exec.Command("wmic", "cpu", "get", "loadpercentage", "/value").Output()
	if err3 == nil {
		for _, line := range strings.Split(string(out3), "\n") {
			if strings.Contains(line, "LoadPercentage=") {
				parts := strings.Split(line, "=")
				if len(parts) == 2 {
					if load, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
						// Estimar temp desde carga
						return 35.0 + (load * 0.45)
					}
				}
			}
		}
	}

	return estimateTemp()
}

func estimateTemp() float64 {
	cpu := getCPULoad()
	base := 35.0 + (cpu * 0.5)
	rand.Seed(time.Now().UnixNano())
	jitter := (rand.Float64() - 0.5) * 3.0
	return base + jitter
}

func getCPULoad() float64 {
	start := time.Now()
	count := 0
	for time.Since(start) < 100*time.Millisecond {
		count++
	}
	load := float64(count) / 1000000.0
	if load > 100 {
		load = 100
	}
	return load
}
