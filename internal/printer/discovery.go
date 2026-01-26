package printer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
)

// DiscoveredPrinter represents a printer found on the network.
type DiscoveredPrinter struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Host           string              `json:"host"`
	Port           int                 `json:"port"`
	Type           model.ConnectionType `json:"type"`
	Model          string              `json:"model,omitempty"`
	Manufacturer   string              `json:"manufacturer,omitempty"`
	Version        string              `json:"version,omitempty"`
	AlreadyAdded   bool                `json:"already_added"`
}

// Discovery handles automatic printer discovery on the local network.
type Discovery struct {
	timeout time.Duration
}

// NewDiscovery creates a new printer discovery instance.
func NewDiscovery() *Discovery {
	return &Discovery{
		timeout: 3 * time.Second,
	}
}

// ScanNetwork discovers printers on the local network.
// It combines multiple discovery methods: mDNS, port scanning, and SSDP.
func (d *Discovery) ScanNetwork(ctx context.Context) ([]DiscoveredPrinter, error) {
	var (
		mu       sync.Mutex
		printers []DiscoveredPrinter
		wg       sync.WaitGroup
	)

	// Get local network range
	localIP, network, err := getLocalNetwork()
	if err != nil {
		return nil, fmt.Errorf("failed to get local network: %w", err)
	}
	slog.Info("scanning network", "local_ip", localIP, "network", network)

	// Generate IP range to scan
	ips := generateIPRange(network)
	slog.Info("scanning IPs", "count", len(ips))

	// Scan each IP for known printer ports
	semaphore := make(chan struct{}, 50) // Limit concurrent connections

	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			found := d.probeHost(ctx, ip)
			if len(found) > 0 {
				mu.Lock()
				printers = append(printers, found...)
				mu.Unlock()
			}
		}(ip)
	}

	wg.Wait()

	slog.Info("discovery complete", "found", len(printers))
	return printers, nil
}

// probeHost checks a single host for known printer services.
func (d *Discovery) probeHost(ctx context.Context, host string) []DiscoveredPrinter {
	var found []DiscoveredPrinter

	// Check common printer ports
	ports := []struct {
		port    int
		check   func(ctx context.Context, host string, port int) *DiscoveredPrinter
	}{
		{80, d.checkOctoPrint},      // OctoPrint default
		{5000, d.checkOctoPrint},    // OctoPrint alt port
		{7125, d.checkMoonraker},    // Moonraker default
		{80, d.checkMoonraker},      // Moonraker behind proxy
		{8883, d.checkBambu},        // Bambu Lab MQTT
		{3000, d.checkChiTu},        // ChiTu/Elegoo resin printers
		{6000, d.checkChiTu},        // ChiTu alt port
	}

	for _, p := range ports {
		// Quick TCP check first
		if !d.isPortOpen(host, p.port) {
			continue
		}

		// Detailed service check
		if printer := p.check(ctx, host, p.port); printer != nil {
			// Avoid duplicates
			isDupe := false
			for _, existing := range found {
				if existing.Host == printer.Host && existing.Type == printer.Type {
					isDupe = true
					break
				}
			}
			if !isDupe {
				found = append(found, *printer)
			}
		}
	}

	return found
}

// isPortOpen does a quick TCP port check.
func (d *Discovery) isPortOpen(host string, port int) bool {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// checkOctoPrint probes for OctoPrint API.
func (d *Discovery) checkOctoPrint(ctx context.Context, host string, port int) *DiscoveredPrinter {
	url := fmt.Sprintf("http://%s:%d/api/version", host, port)
	
	client := &http.Client{Timeout: d.timeout}
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	// OctoPrint returns 200 even without API key for version endpoint
	if resp.StatusCode != 200 {
		return nil
	}

	// Must parse the response and verify it's actually OctoPrint
	var version struct {
		API    string `json:"api"`
		Server string `json:"server"`
		Text   string `json:"text"`
	}
	
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])
	
	// Verify this is OctoPrint - must have "OctoPrint" in response
	if !strings.Contains(bodyStr, "OctoPrint") && !strings.Contains(bodyStr, "octoprint") {
		return nil
	}
	
	// Parse JSON for version info
	if err := json.Unmarshal(body[:n], &version); err != nil {
		// Still return if we found OctoPrint string
		version.Server = "unknown"
	}
	
	name := fmt.Sprintf("OctoPrint @ %s", host)
	
	return &DiscoveredPrinter{
		ID:           uuid.New().String(),
		Name:         name,
		Host:         host,
		Port:         port,
		Type:         model.ConnectionTypeOctoPrint,
		Version:      version.Server,
		Manufacturer: "OctoPrint",
	}
}

// checkMoonraker probes for Moonraker API.
func (d *Discovery) checkMoonraker(ctx context.Context, host string, port int) *DiscoveredPrinter {
	url := fmt.Sprintf("http://%s:%d/server/info", host, port)
	
	client := &http.Client{Timeout: d.timeout}
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	// Must verify it's actually Moonraker
	body := make([]byte, 2048)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])
	
	// Moonraker response should contain "moonraker" or "klippy"
	isMoonraker := strings.Contains(strings.ToLower(bodyStr), "moonraker") ||
		strings.Contains(strings.ToLower(bodyStr), "klippy") ||
		strings.Contains(strings.ToLower(bodyStr), "klipper")
	
	if !isMoonraker {
		return nil
	}
	
	name := fmt.Sprintf("Klipper @ %s", host)
	
	return &DiscoveredPrinter{
		ID:           uuid.New().String(),
		Name:         name,
		Host:         host,
		Port:         port,
		Type:         model.ConnectionTypeMoonraker,
		Manufacturer: "Klipper/Moonraker",
	}
}

// checkChiTu probes for ChiTu-based resin printers (Elegoo, Anycubic, Phrozen, etc.).
// These printers run a web server typically on port 3000 or 6000.
func (d *Discovery) checkChiTu(ctx context.Context, host string, port int) *DiscoveredPrinter {
	// ChiTu printers have endpoints like /getSysInfo or /getSystemInfo
	endpoints := []string{
		fmt.Sprintf("http://%s:%d/getSysInfo", host, port),
		fmt.Sprintf("http://%s:%d/getSystemInfo", host, port),
		fmt.Sprintf("http://%s:%d/system/info", host, port),
	}

	client := &http.Client{Timeout: d.timeout}

	for _, url := range endpoints {
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}

		body := make([]byte, 2048)
		n, _ := resp.Body.Read(body)
		bodyStr := string(body[:n])
		bodyLower := strings.ToLower(bodyStr)

		// Check for ChiTu/resin printer indicators
		isChiTu := strings.Contains(bodyLower, "chitu") ||
			strings.Contains(bodyLower, "elegoo") ||
			strings.Contains(bodyLower, "anycubic") ||
			strings.Contains(bodyLower, "phrozen") ||
			strings.Contains(bodyLower, "creality") ||
			strings.Contains(bodyLower, "mars") ||
			strings.Contains(bodyLower, "saturn") ||
			strings.Contains(bodyLower, "resin") ||
			strings.Contains(bodyLower, "lcd") ||
			strings.Contains(bodyLower, "msla")

		if isChiTu {
			name := "Resin Printer"
			manufacturer := "Unknown"

			// Try to identify manufacturer
			if strings.Contains(bodyLower, "elegoo") {
				manufacturer = "Elegoo"
				name = fmt.Sprintf("Elegoo @ %s", host)
			} else if strings.Contains(bodyLower, "anycubic") {
				manufacturer = "Anycubic"
				name = fmt.Sprintf("Anycubic @ %s", host)
			} else if strings.Contains(bodyLower, "phrozen") {
				manufacturer = "Phrozen"
				name = fmt.Sprintf("Phrozen @ %s", host)
			} else if strings.Contains(bodyLower, "creality") {
				manufacturer = "Creality"
				name = fmt.Sprintf("Creality @ %s", host)
			} else {
				name = fmt.Sprintf("Resin Printer @ %s", host)
			}

			slog.Info("found ChiTu resin printer", "host", host, "port", port, "manufacturer", manufacturer)

			return &DiscoveredPrinter{
				ID:           uuid.New().String(),
				Name:         name,
				Host:         host,
				Port:         port,
				Type:         model.ConnectionTypeChiTu,
				Manufacturer: manufacturer,
			}
		}
	}

	return nil
}

// checkBambu probes for Bambu Lab printers.
// Bambu printers use MQTT on port 8883 and have FTPS on port 990.
func (d *Discovery) checkBambu(ctx context.Context, host string, port int) *DiscoveredPrinter {
	// Check if both Bambu-specific ports are open
	// Bambu uses 8883 (MQTTS) and 990 (FTPS)
	hasMQTT := d.isPortOpen(host, 8883)
	hasFTPS := d.isPortOpen(host, 990)
	
	// Must have at least MQTT port
	if !hasMQTT {
		return nil
	}
	
	// If only MQTT, could be many things - need more verification
	// If both MQTT and FTPS, very likely Bambu
	if !hasFTPS {
		// Try to verify via FTP banner on port 21 (some Bambu have this)
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:21", host), 2*time.Second)
		if err == nil {
			defer conn.Close()
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			banner := make([]byte, 512)
			n, _ := conn.Read(banner)
			bannerStr := strings.ToLower(string(banner[:n]))
			if !strings.Contains(bannerStr, "bambu") && !strings.Contains(bannerStr, "bbl") {
				return nil
			}
		} else {
			// No FTP either - not conclusive enough
			return nil
		}
	}
	
	slog.Info("found potential Bambu printer", "host", host, "hasMQTT", hasMQTT, "hasFTPS", hasFTPS)
	
	return &DiscoveredPrinter{
		ID:           uuid.New().String(),
		Name:         fmt.Sprintf("Bambu Printer @ %s", host),
		Host:         host,
		Port:         8883,
		Type:         model.ConnectionTypeBambuLAN,
		Manufacturer: "Bambu Lab",
	}
}

// ScanSSDPBambu performs SSDP discovery specifically for Bambu printers.
// Bambu printers advertise via SSDP with specific device types.
func (d *Discovery) ScanSSDPBambu(ctx context.Context) ([]DiscoveredPrinter, error) {
	var printers []DiscoveredPrinter
	var mu sync.Mutex

	// Multiple SSDP search targets to try
	searchTargets := []string{
		"urn:bambulab-com:device:3dprinter:1",
		"ssdp:all",
		"upnp:rootdevice",
	}

	for _, st := range searchTargets {
		found, err := d.sendSSDPSearch(ctx, st)
		if err != nil {
			slog.Debug("SSDP search failed", "target", st, "error", err)
			continue
		}
		mu.Lock()
		printers = append(printers, found...)
		mu.Unlock()
	}

	// Deduplicate by host
	seen := make(map[string]bool)
	var unique []DiscoveredPrinter
	for _, p := range printers {
		if !seen[p.Host] {
			seen[p.Host] = true
			unique = append(unique, p)
		}
	}

	return unique, nil
}

// sendSSDPSearch sends an SSDP M-SEARCH and collects Bambu responses.
func (d *Discovery) sendSSDPSearch(ctx context.Context, searchTarget string) ([]DiscoveredPrinter, error) {
	var printers []DiscoveredPrinter

	// SSDP M-SEARCH request
	ssdpAddr := "239.255.255.250:1900"
	searchMsg := fmt.Sprintf("M-SEARCH * HTTP/1.1\r\n"+
		"HOST: 239.255.255.250:1900\r\n"+
		"MAN: \"ssdp:discover\"\r\n"+
		"MX: 3\r\n"+
		"ST: %s\r\n"+
		"\r\n", searchTarget)

	// Create UDP socket
	addr, err := net.ResolveUDPAddr("udp4", ssdpAddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Send discovery request multiple times for reliability
	for i := 0; i < 3; i++ {
		conn.WriteToUDP([]byte(searchMsg), addr)
		time.Sleep(100 * time.Millisecond)
	}

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read responses
	buffer := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			break // Timeout or error
		}

		response := string(buffer[:n])
		responseLower := strings.ToLower(response)
		
		// Check for Bambu-specific identifiers
		isBambu := strings.Contains(responseLower, "bambu") ||
			strings.Contains(responseLower, "bbl") ||
			strings.Contains(responseLower, "x1c") ||
			strings.Contains(responseLower, "p1p") ||
			strings.Contains(responseLower, "p1s") ||
			strings.Contains(responseLower, "a1 mini") ||
			strings.Contains(responseLower, "a1mini")

		if isBambu {
			printer := DiscoveredPrinter{
				ID:           uuid.New().String(),
				Name:         fmt.Sprintf("Bambu Printer @ %s", remoteAddr.IP.String()),
				Host:         remoteAddr.IP.String(),
				Port:         8883,
				Type:         model.ConnectionTypeBambuLAN,
				Manufacturer: "Bambu Lab",
			}

			// Parse headers for more info
			lines := strings.Split(response, "\r\n")
			for _, line := range lines {
				lineLower := strings.ToLower(line)
				if strings.HasPrefix(lineLower, "usn:") {
					usn := strings.TrimSpace(line[4:])
					printer.Model = usn
					// Try to extract model name
					if strings.Contains(usn, "X1") {
						printer.Name = fmt.Sprintf("Bambu X1 @ %s", remoteAddr.IP.String())
					} else if strings.Contains(usn, "P1P") {
						printer.Name = fmt.Sprintf("Bambu P1P @ %s", remoteAddr.IP.String())
					} else if strings.Contains(usn, "P1S") {
						printer.Name = fmt.Sprintf("Bambu P1S @ %s", remoteAddr.IP.String())
					} else if strings.Contains(usn, "A1") {
						printer.Name = fmt.Sprintf("Bambu A1 @ %s", remoteAddr.IP.String())
					}
				}
				if strings.HasPrefix(lineLower, "server:") {
					printer.Version = strings.TrimSpace(line[7:])
				}
				if strings.HasPrefix(lineLower, "location:") {
					// Location header might have useful info
					slog.Debug("Bambu location", "location", line)
				}
			}

			slog.Info("found Bambu printer", "host", printer.Host, "name", printer.Name)
			printers = append(printers, printer)
		}
	}

	return printers, nil
}

// getLocalNetwork returns the local IP and network CIDR.
func getLocalNetwork() (string, *net.IPNet, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", nil, err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), ipnet, nil
			}
		}
	}

	return "", nil, fmt.Errorf("no suitable network interface found")
}

// generateIPRange generates all IPs in a /24 network.
func generateIPRange(network *net.IPNet) []string {
	var ips []string
	
	// For simplicity, assume /24 network
	ip := network.IP.Mask(network.Mask)
	
	for i := 1; i < 255; i++ {
		newIP := make(net.IP, len(ip))
		copy(newIP, ip)
		newIP[3] = byte(i)
		ips = append(ips, newIP.String())
	}

	return ips
}

// ScanBambuUDP uses Bambu's native UDP discovery protocol on port 2021.
func (d *Discovery) ScanBambuUDP(ctx context.Context) ([]DiscoveredPrinter, error) {
	var printers []DiscoveredPrinter

	// Bambu printers respond to broadcast on port 2021
	broadcastAddr := "255.255.255.255:2021"

	// Discovery message (Bambu uses a simple JSON-like format)
	discoveryMsg := []byte(`M99999`)

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: 0})
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP socket: %w", err)
	}
	defer conn.Close()

	// Enable broadcast
	addr, err := net.ResolveUDPAddr("udp4", broadcastAddr)
	if err != nil {
		return nil, err
	}

	// Send discovery packet multiple times
	for i := 0; i < 3; i++ {
		conn.WriteToUDP(discoveryMsg, addr)
		time.Sleep(200 * time.Millisecond)
	}

	// Set timeout
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read responses
	buffer := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			break
		}

		response := string(buffer[:n])
		slog.Debug("Bambu UDP response", "from", remoteAddr.IP.String(), "response", response)

		// Bambu responds with printer info
		// Example response might contain device name, model, etc.
		if len(response) > 0 {
			printer := DiscoveredPrinter{
				ID:           uuid.New().String(),
				Name:         fmt.Sprintf("Bambu Printer @ %s", remoteAddr.IP.String()),
				Host:         remoteAddr.IP.String(),
				Port:         8883,
				Type:         model.ConnectionTypeBambuLAN,
				Manufacturer: "Bambu Lab",
			}

			// Try to parse model from response
			responseLower := strings.ToLower(response)
			if strings.Contains(responseLower, "x1") || strings.Contains(responseLower, "x1c") {
				printer.Name = fmt.Sprintf("Bambu X1 @ %s", remoteAddr.IP.String())
				printer.Model = "X1 Carbon"
			} else if strings.Contains(responseLower, "p1s") {
				printer.Name = fmt.Sprintf("Bambu P1S @ %s", remoteAddr.IP.String())
				printer.Model = "P1S"
			} else if strings.Contains(responseLower, "p1p") {
				printer.Name = fmt.Sprintf("Bambu P1P @ %s", remoteAddr.IP.String())
				printer.Model = "P1P"
			} else if strings.Contains(responseLower, "a1 mini") || strings.Contains(responseLower, "a1mini") {
				printer.Name = fmt.Sprintf("Bambu A1 Mini @ %s", remoteAddr.IP.String())
				printer.Model = "A1 Mini"
			} else if strings.Contains(responseLower, "a1") {
				printer.Name = fmt.Sprintf("Bambu A1 @ %s", remoteAddr.IP.String())
				printer.Model = "A1"
			}

			slog.Info("found Bambu via UDP", "host", printer.Host, "model", printer.Model)
			printers = append(printers, printer)
		}
	}

	return printers, nil
}

// QuickScan does a fast scan of only the most common ports.
func (d *Discovery) QuickScan(ctx context.Context) ([]DiscoveredPrinter, error) {
	var (
		mu       sync.Mutex
		printers []DiscoveredPrinter
		wg       sync.WaitGroup
	)

	// Run Bambu SSDP discovery
	wg.Add(1)
	go func() {
		defer wg.Done()
		bambuPrinters, err := d.ScanSSDPBambu(ctx)
		if err == nil && len(bambuPrinters) > 0 {
			mu.Lock()
			printers = append(printers, bambuPrinters...)
			mu.Unlock()
		}
	}()

	// Run Bambu UDP discovery
	wg.Add(1)
	go func() {
		defer wg.Done()
		bambuPrinters, err := d.ScanBambuUDP(ctx)
		if err == nil && len(bambuPrinters) > 0 {
			mu.Lock()
			printers = append(printers, bambuPrinters...)
			mu.Unlock()
		}
	}()

	// Run network scan
	networkPrinters, err := d.ScanNetwork(ctx)
	if err == nil {
		mu.Lock()
		printers = append(printers, networkPrinters...)
		mu.Unlock()
	}

	wg.Wait()

	// Deduplicate by host
	seen := make(map[string]bool)
	var unique []DiscoveredPrinter
	for _, p := range printers {
		key := fmt.Sprintf("%s:%s", p.Host, p.Type)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, p)
		}
	}

	return unique, nil
}

