// ╔══════════════════════════════════════════════════════════════════════╗
// ║        BITCOPPER® DAEMON — Proof of Heat Protocol                    ║
// ║        Versión 2.0.0 — Protocolo Nyquist Integrado                  ║
// ║                                                                      ║
// ║  © 2024-2026 Bitcopper Tech SpA — Todos los derechos reservados     ║
// ║  RUT: 78.355.848-3 | Chuquicamata, Calama, Región de Antofagasta   ║
// ║  Bitcopper Technologies LLC — Wyoming, United States                 ║
// ║  ID: 2026-001909118                                                  ║
// ║                                                                      ║
// ║  BITCOPPER® es marca registrada de Bitcopper Tech SpA.              ║
// ║  BITCU® y CUPR® son marcas registradas de Bitcopper Tech SpA.       ║
// ║                                                                      ║
// ║  Software propietario. Prohibida su reproducción, distribución       ║
// ║  o modificación sin autorización expresa de Bitcopper Tech SpA.     ║
// ║                                                                      ║
// ║  "In cuprum veritas" — Una placa. Una wallet. Una identidad.        ║
// ╚══════════════════════════════════════════════════════════════════════╝

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	RAILWAY_API   = "https://bitcu-server-production.up.railway.app"
	RAILWAY_HOST  = "bitcu-server-production.up.railway.app"
	VERSION       = "2.0.0"
	MINE_INTERVAL = 5 * time.Second
)

type MineRequest struct {
	Wallet    string  `json:"wallet"`
	DeviceID  string  `json:"device_id"`
	Temp      float64 `json:"temp"`
	CPU       float64 `json:"cpu"`
	Rho       string  `json:"rho"`
	RAM       float64 `json:"ram"`
	Timestamp int64   `json:"timestamp"`
}

type MineResponse struct {
	OK             bool   `json:"ok"`
	Block          int    `json:"block"`
	Hash           string `json:"hash"`
	Reward         string `json:"reward"`
	Error          string `json:"error"`
	Reason         string `json:"reason"`
	WalletSoberana string `json:"wallet_soberana"`
}

var (
	wallet        string
	deviceID      string
	sessionBlocks int
	sessionBITCU  float64
	lastTemp      float64
	lastCPU       float64
	isMining      bool
	httpClient    *http.Client
)

func newHTTPClient() *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := &net.Dialer{Timeout: 15 * time.Second}
			host, port, _ := net.SplitHostPort(addr)
			if port == "" {
				port = "443"
			}
			addrs, err := (&net.Resolver{
				PreferGo: true,
				Dial: func(ctx2 context.Context, network2, address string) (net.Conn, error) {
					return net.Dial("udp", "8.8.8.8:53")
				},
			}).LookupHost(ctx, host)
			if err != nil || len(addrs) == 0 {
				return d.DialContext(ctx, network, addr)
			}
			return d.DialContext(ctx, network, addrs[0]+":"+port)
		},
		TLSClientConfig: &tls.Config{
			ServerName:         RAILWAY_HOST,
			InsecureSkipVerify: true,
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	}
	return &http.Client{Transport: transport, Timeout: 20 * time.Second}
}

// ── IDENTIDAD SELLADA ──────────────────────────────────────────────────────

func getCfgDir() string {
	if runtime.GOOS == "android" {
		return "/sdcard/.bitcopper"
	}
	return os.ExpandEnv("$HOME/.bitcopper")
}

// loadWallet carga la wallet sellada. Si no existe, la genera UNA SOLA VEZ y la sella.
// En arranques posteriores, si el archivo no existe → error fatal (identidad corrompida).
func loadWallet() string {
	cfgDir := getCfgDir()
	cfgFile := cfgDir + "/wallet"
	os.MkdirAll(cfgDir, 0700)

	data, err := os.ReadFile(cfgFile)
	if err == nil && len(data) > 10 {
		return strings.TrimSpace(string(data))
	}

	// Primera vez: generar y sellar
	fmt.Println("[NYQUIST] Primera ejecución — generando identidad soberana...")
	seed := fmt.Sprintf("%d-%d-%s", time.Now().UnixNano(), rand.Int63(), runtime.GOOS)
	hash := sha256.Sum256([]byte(seed))
	w := fmt.Sprintf("BTCU-%x", hash)
	if err := os.WriteFile(cfgFile, []byte(w), 0400); err != nil {
		log.Fatalf("[NYQUIST] ERROR FATAL: No se pudo sellar la wallet: %v", err)
	}
	fmt.Println("[NYQUIST] ✓ Wallet sellada permanentemente.")
	return w
}

// loadDeviceID carga el device_id sellado. Misma lógica que wallet.
func loadDeviceID() string {
	cfgDir := getCfgDir()
	cfgFile := cfgDir + "/device_id"
	os.MkdirAll(cfgDir, 0700)

	data, err := os.ReadFile(cfgFile)
	if err == nil && len(data) > 10 {
		return strings.TrimSpace(string(data))
	}

	// Primera vez: generar y sellar
	hostname, _ := os.Hostname()
	seed := fmt.Sprintf("%s|%s|%s|%d", hostname, runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	hash := sha256.Sum256([]byte(seed))
	d := fmt.Sprintf("CU-%x", hash)
	if err := os.WriteFile(cfgFile, []byte(d), 0400); err != nil {
		log.Fatalf("[NYQUIST] ERROR FATAL: No se pudo sellar el device_id: %v", err)
	}
	fmt.Println("[NYQUIST] ✓ Device ID sellado permanentemente.")
	return d
}

// verifyIdentity verifica consistencia local antes de minar
func verifyIdentity() {
	cfgDir := getCfgDir()

	wData, err := os.ReadFile(cfgDir + "/wallet")
	if err != nil {
		log.Fatalf("[NYQUIST] ERROR FATAL: Wallet no encontrada. Identidad corrompida.")
	}
	dData, err := os.ReadFile(cfgDir + "/device_id")
	if err != nil {
		log.Fatalf("[NYQUIST] ERROR FATAL: Device ID no encontrado. Identidad corrompida.")
	}

	if strings.TrimSpace(string(wData)) != wallet {
		log.Fatalf("[NYQUIST] ERROR FATAL: Wallet en disco no coincide con wallet en memoria. Posible manipulación.")
	}
	if strings.TrimSpace(string(dData)) != deviceID {
		log.Fatalf("[NYQUIST] ERROR FATAL: Device ID en disco no coincide. Posible manipulación.")
	}
}

// recoverIdentity intenta recuperar la wallet soberana del servidor via device_id
func recoverIdentity() {
	fmt.Println("[NYQUIST] Intentando recuperar identidad soberana del servidor...")
	url := RAILWAY_API + "/api/recover/" + deviceID
	resp, err := httpClient.Get(url)
	if err != nil {
		log.Printf("[NYQUIST] No se pudo contactar servidor para recuperación: %v", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK             bool   `json:"ok"`
		WalletSoberana string `json:"wallet_soberana"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.OK || result.WalletSoberana == "" {
		log.Printf("[NYQUIST] Recuperación fallida: %s", string(body))
		return
	}

	cfgDir := getCfgDir()
	// Temporalmente dar permisos de escritura para recuperación
	os.Chmod(cfgDir+"/wallet", 0600)
	if err := os.WriteFile(cfgDir+"/wallet", []byte(result.WalletSoberana), 0400); err != nil {
		log.Fatalf("[NYQUIST] ERROR FATAL: No se pudo restaurar wallet soberana: %v", err)
	}
	fmt.Printf("[NYQUIST] ✓ Identidad soberana restaurada: %s\n", result.WalletSoberana[:20]+"...")
	fmt.Println("[NYQUIST] Reiniciando daemon con identidad correcta...")
	// Recargar wallet en memoria
	wallet = result.WalletSoberana
}

// ── HANDSHAKE ─────────────────────────────────────────────────────────────

func doHandshake(wallet, deviceID string) {
	type HandshakeReq struct {
		Wallet   string  `json:"wallet"`
		DeviceID string  `json:"device_id"`
		Platform string  `json:"platform"`
		Temp     float64 `json:"temp"`
		Version  string  `json:"version"`
	}
	platform := "desktop"
	switch runtime.GOOS {
	case "android":
		platform = "android_tv"
	case "darwin":
		platform = "mac"
	case "linux":
		platform = "linux"
	case "windows":
		platform = "windows"
	}
	req := HandshakeReq{
		Wallet:   wallet,
		DeviceID: deviceID,
		Platform: platform,
		Temp:     getTemperature(),
		Version:  VERSION,
	}
	body, _ := json.Marshal(req)
	resp, err := httpClient.Post(RAILWAY_API+"/api/handshake", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("Handshake error: %v", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("Handshake OK")
}

// ── MAIN ──────────────────────────────────────────────────────────────────

func main() {
	fmt.Printf("BITCOPPER DAEMON v%s\n", VERSION)
	fmt.Printf("Proof of Heat Protocol - In cuprum veritas.\n\n")

	httpClient = newHTTPClient()

	// Cargar identidad sellada
	wallet = loadWallet()
	deviceID = loadDeviceID()

	// Verificar consistencia local (Nyquist Layer 1)
	verifyIdentity()

	fmt.Printf("Wallet:    %s\n", wallet)
	devPreview := deviceID
	if len(devPreview) > 40 {
		devPreview = devPreview[:40]
	}
	fmt.Printf("Device ID: %s...\n\n", devPreview)

	doHandshake(wallet, deviceID)

	isMining = true
	go gossipLoop()
	go miningLoop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	fmt.Println("\nDaemon detenido.")
}

// ── MINING LOOP ───────────────────────────────────────────────────────────

func miningLoop() {
	consecutiveErrors := 0

	for {
		// Verificar identidad antes de cada ciclo (Nyquist Layer 1)
		verifyIdentity()

		temp := getTemperature()
		cpu := getCPULoad()
		lastTemp = temp
		lastCPU = cpu

		rho := 1.68e-8 * (1 + 0.00393*((temp+273.15)-293.15))
		rhoStr := fmt.Sprintf("%.4e", rho)

		req := MineRequest{
			Wallet:    wallet,
			DeviceID:  deviceID,
			Temp:      temp,
			CPU:       cpu,
			Rho:       rhoStr,
			RAM:       50,
			Timestamp: time.Now().UnixMilli(),
		}
		body, _ := json.Marshal(req)
		fmt.Printf("Enviando bloque...\n")

		resp, err := httpClient.Post(RAILWAY_API+"/api/mine", "application/json", bytes.NewReader(body))
		if err != nil {
			log.Printf("Error de red: %v", err)
			consecutiveErrors++
			if consecutiveErrors >= 10 {
				log.Printf("[NYQUIST] 10 errores consecutivos — esperando 60s antes de reintentar...")
				time.Sleep(60 * time.Second)
				consecutiveErrors = 0
			}
			time.Sleep(MINE_INTERVAL)
			continue
		}

		body2, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		log.Printf("Respuesta raw: %s", string(body2))

		var mr MineResponse
		json.Unmarshal(body2, &mr)
		log.Printf("Respuesta: ok=%v block=%d reward=%s", mr.OK, mr.Block, mr.Reward)

		if mr.OK {
			consecutiveErrors = 0
			sessionBlocks++
			reward := 0.0
			fmt.Sscanf(mr.Reward, "%f", &reward)
			sessionBITCU += reward
			fmt.Printf("Bloque #%d | %.10f BITCU\n", mr.Block, reward)
		} else {
			consecutiveErrors++

			switch mr.Error {
			case "NYQUIST_VIOLATION":
				// El servidor detectó intrusión — recuperar identidad soberana
				fmt.Printf("[NYQUIST] ⚠ Violación detectada por servidor. Wallet soberana: %s\n", mr.WalletSoberana)
				if mr.WalletSoberana != "" && mr.WalletSoberana != wallet {
					log.Fatalf("[NYQUIST] ERROR FATAL: La wallet soberana del servidor no coincide con la local. Identidad comprometida.")
				}
				// Si wallet_soberana == wallet local, el intruso era otro — continúa
				fmt.Println("[NYQUIST] Identidad local verificada. Continuando...")

			case "Wallet inválida":
				// Intentar recuperación automática
				fmt.Println("[NYQUIST] Wallet rechazada — iniciando recuperación automática...")
				recoverIdentity()

			default:
				log.Printf("[ERROR] Servidor rechazó bloque: %s — %s", mr.Error, mr.Reason)
			}
		}

		time.Sleep(MINE_INTERVAL)
	}
}

// ── COPPER-CORE GOSSIP ────────────────────────────────────────────────────
const COPPER_CORE = "https://copper-core-production.up.railway.app"

func announceToSeed() {
        type AnnounceReq struct {
                Wallet  string `json:"wallet"`
                Port    int    `json:"port"`
                Version string `json:"version"`
        }
        req := AnnounceReq{Wallet: wallet, Port: 8765, Version: VERSION}
        body, _ := json.Marshal(req)
        resp, err := httpClient.Post(COPPER_CORE+"/announce", "application/json", bytes.NewReader(body))
        if err != nil {
                log.Printf("[P2P] No se pudo anunciar al seed node: %v", err)
                return
        }
        defer resp.Body.Close()
        fmt.Println("[P2P] ✓ Anunciado al copper-core seed node")
}

func gossipLoop() {
        time.Sleep(10 * time.Second)
        for {
                announceToSeed()
                time.Sleep(60 * time.Second)
        }
}
