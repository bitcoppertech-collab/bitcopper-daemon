package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"io"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const (
	RAILWAY_API   = "https://bitcu-server-production.up.railway.app"
	RAILWAY_HOST  = "bitcu-server-production.up.railway.app"
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
			// Resolver DNS via 8.8.8.8
			host, port, _ := net.SplitHostPort(addr)
			if port == "" { port = "443" }
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

func doHandshake(wallet, deviceID string) {
	type HandshakeReq struct {
		Wallet   string  `json:"wallet"`
		DeviceID string  `json:"device_id"`
		Platform string  `json:"platform"`
		Temp     float64 `json:"temp"`
		Version  string  `json:"version"`
	}
	req := HandshakeReq{
		Wallet:   wallet,
		DeviceID: deviceID,
		Platform: "android_tv",
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

func main() {
	fmt.Printf("BITCOPPER DAEMON v%s\n", VERSION)
	fmt.Printf("Proof of Heat Protocol - In cuprum veritas.\n\n")
	httpClient = newHTTPClient()
	wallet = loadOrCreateWallet()
	deviceID = getDeviceID()
	fmt.Printf("Wallet:    %s\n", wallet)
	devPreview := deviceID
	if len(devPreview) > 40 {
		devPreview = devPreview[:40]
	}
	fmt.Printf("Device ID: %s\n\n", devPreview+"...")
	doHandshake(wallet, deviceID)
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
		fmt.Printf("Enviando bloque...\n")
		resp, err := httpClient.Post(RAILWAY_API+"/api/mine", "application/json", bytes.NewReader(body))
		if err != nil {
			log.Printf("Error minando: %v", err)
		} else {
				body2, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			log.Printf("Respuesta raw: %s", string(body2))
			var mr MineResponse
			json.Unmarshal(body2, &mr)
			log.Printf("Respuesta: ok=%v block=%d reward=%s", mr.OK, mr.Block, mr.Reward)
			if mr.OK {
				sessionBlocks++
				reward := 0.0
				fmt.Sscanf(mr.Reward, "%f", &reward)
				sessionBITCU += reward
				fmt.Printf("Bloque #%d | %.10f BITCU\n", mr.Block, reward)
			}
		}
		time.Sleep(MINE_INTERVAL)
	}
}

func getCfgDir() string {
	if runtime.GOOS == "android" {
		return "/sdcard/.bitcopper"
	}
	return os.ExpandEnv("$HOME/.bitcopper")
}

func loadOrCreateWallet() string {
	cfgDir := getCfgDir()
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
	cfgDir := getCfgDir()
	cfgFile := cfgDir + "/device_id"
	os.MkdirAll(cfgDir, 0700)
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
