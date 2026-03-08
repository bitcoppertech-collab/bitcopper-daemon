package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const (
	RAILWAY_API   = "https://bitcu-server-production.up.railway.app"
	LOCAL_PORT    = "8765"
	VERSION       = "1.0.0"
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
	OK     bool   `json:"ok"`
	Block  int    `json:"block"`
	Hash   string `json:"hash"`
	Reward string `json:"reward"`
}

type StatusResponse struct {
	Version    string  `json:"version"`
	Wallet     string  `json:"wallet"`
	DeviceID   string  `json:"device_id"`
	OS         string  `json:"os"`
	Temp       float64 `json:"temp"`
	CPU        float64 `json:"cpu"`
	Mining     bool    `json:"mining"`
	Blocks     int     `json:"blocks_this_session"`
	TotalBITCU float64 `json:"bitcu_this_session"`
}

var (
	wallet        string
	deviceID      string
	sessionBlocks int
	sessionBITCU  float64
	lastTemp      float64
	lastCPU       float64
	isMining      bool
)

func main() {
	fmt.Printf("BITCOPPER DAEMON v%s\n", VERSION)
	fmt.Printf("Proof of Heat Protocol - In cuprum veritas.\n\n")
	wallet = loadOrCreateWallet()
	deviceID = getDeviceID()
	fmt.Printf("Wallet:    %s\n", wallet)
	fmt.Printf("Device ID: %s\n\n", deviceID[:40]+"...")
	go startLocalServer()
	isMining = true
	go miningLoop()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	fmt.Println("\nDaemon detenido.")
}

func miningLoop() {
	for {
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
		resp, err := http.Post(RAILWAY_API+"/api/mine", "application/json", bytes.NewReader(body))
		if err != nil {
			log.Printf("Error minando: %v", err)
		} else {
			var mr MineResponse
			json.NewDecoder(resp.Body).Decode(&mr)
			resp.Body.Close()
			if mr.OK {
				sessionBlocks++
				reward := 0.0
				fmt.Sscanf(mr.Reward, "%f", &reward)
				sessionBITCU += reward
				fmt.Printf("Bloque #%d | %.10f BITCU | Temp: %.1fC | CPU: %.1f%%\n", mr.Block, reward, temp, cpu)
			}
		}
		time.Sleep(MINE_INTERVAL)
	}
}

func startLocalServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"service": "bitcopper-daemon", "version": VERSION, "status": "running"})
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(StatusResponse{Version: VERSION, Wallet: wallet, DeviceID: deviceID, OS: runtime.GOOS + "/" + runtime.GOARCH, Temp: lastTemp, CPU: lastCPU, Mining: isMining, Blocks: sessionBlocks, TotalBITCU: sessionBITCU})
	})
	mux.HandleFunc("/wallet", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"wallet": wallet, "device_id": deviceID})
	})
	fmt.Printf("API local en http://localhost:%s\n\n", LOCAL_PORT)
	log.Fatal(http.ListenAndServe(":"+LOCAL_PORT, mux))
}

func loadOrCreateWallet() string {
	cfgDir := os.ExpandEnv("$HOME/.bitcopper")
	cfgFile := cfgDir + "/wallet"
	os.MkdirAll(cfgDir, 0700)
	data, err := os.ReadFile(cfgFile)
	if err == nil && len(data) > 10 {
		return string(data)
	}
	seed := fmt.Sprintf("%d-%d-%s", time.Now().UnixNano(), rand.Int63(), runtime.GOOS)
	hash := sha256.Sum256([]byte(seed))
	w := fmt.Sprintf("BTCU-%x", hash)
	os.WriteFile(cfgFile, []byte(w), 0600)
	fmt.Println("Nueva wallet generada y sellada.")
	return w
}

func getDeviceID() string {
	cfgDir := os.ExpandEnv("$HOME/.bitcopper")
	cfgFile := cfgDir + "/device_id"
	data, err := os.ReadFile(cfgFile)
	if err == nil && len(data) > 10 {
		return string(data)
	}
	hostname, _ := os.Hostname()
	seed := fmt.Sprintf("%s|%s|%s|%d", hostname, runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	hash := sha256.Sum256([]byte(seed))
	d := fmt.Sprintf("CU-%x", hash)
	os.WriteFile(cfgFile, []byte(d), 0600)
	return d
}
