package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

const (
	TorProxyAddress = "127.0.0.1:9150"
	TargetFile      = "targets.yaml"
	OutputDir       = "sonuclar"
	ReportFile      = "scan_report.log"
)

func main() {

	fmt.Println("[BASLATILIYOR] Tor Scraper Aracı...")
	hazirlikYap()

	client, err := torIstemcisiOlustur()
	if err != nil {
		log.Fatalf("[HATA] Tor baglantisi kurulamadi: %v", err)
	}

	ipKontrol(client)

	hedefler, err := dosyaOku(TargetFile)
	if err != nil {
		log.Fatalf("[HATA] Hedef dosyasi okunamadi: %v", err)
	}
	fmt.Printf("[BILGI] Toplam %d adet hedef bulundu.\n", len(hedefler))

	raporDosyasi, _ := os.Create(ReportFile)
	defer raporDosyasi.Close()

	for _, url := range hedefler {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		fmt.Printf("[TARANIYOR] %s ... ", url)

		baslangic := time.Now()
		icerik, err := istekAt(client, url)
		sure := time.Since(baslangic)

		durumMesaji := ""
		if err != nil {
			fmt.Println("-> [BASARISIZ]")
			durumMesaji = fmt.Sprintf("[FAIL] %s | Hata: %v | Sure: %s\n", url, err, sure)
		} else {
			fmt.Println("-> [BASARILI]")
			durumMesaji = fmt.Sprintf("[SUCCESS] %s | Sure: %s\n", url, sure)
			dosyaAdi := temizDosyaAdi(url) + ".html"
			kaydet(dosyaAdi, icerik)
		}

		raporDosyasi.WriteString(durumMesaji)
	}

	fmt.Println("\n[BITTI] Tarama tamamlandi. Rapor: scan_report.log")
}

func hazirlikYap() {
	if _, err := os.Stat(OutputDir); os.IsNotExist(err) {
		os.Mkdir(OutputDir, 0755)
	}
}

func torIstemcisiOlustur() (*http.Client, error) {

	dialer, err := proxy.SOCKS5("tcp", TorProxyAddress, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		Dial:            dialer.Dial,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   60 * time.Second,
	}

	return client, nil
}

func ipKontrol(client *http.Client) {
	fmt.Print("[KONTROL] IP Adresi kontrol ediliyor... ")
	resp, err := client.Get("http://check.torproject.org/api/ip")
	if err != nil {
		fmt.Printf("HATA! (%v)\n", err)
		fmt.Println("UYARI: Tor baglantisi veya internet yok gibi görünüyor.")
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Baglanti OK! %s", string(body))
}

func dosyaOku(dosyaYolu string) ([]string, error) {
	file, err := os.Open(dosyaYolu)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var satirlar []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		satir := strings.TrimSpace(scanner.Text())
		if satir != "" && !strings.HasPrefix(satir, "#") {
			satirlar = append(satirlar, satir)
		}
	}
	return satirlar, scanner.Err()
}

func istekAt(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; rv:91.0) Gecko/20100101 Firefox/91.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP Hatasi: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func kaydet(dosyaAdi string, veri []byte) {
	yol := OutputDir + "/" + dosyaAdi
	err := os.WriteFile(yol, veri, 0644)
	if err != nil {
		log.Printf("[HATA] Dosya yazilamadi: %v", err)
	}
}

func temizDosyaAdi(url string) string {
	temiz := strings.ReplaceAll(url, "http://", "")
	temiz = strings.ReplaceAll(temiz, "https://", "")
	temiz = strings.ReplaceAll(temiz, "/", "_")
	temiz = strings.ReplaceAll(temiz, ":", "")
	return temiz
}
