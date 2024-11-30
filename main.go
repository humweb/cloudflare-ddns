package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const configPathEnv = "DDNS_CONFIG_PATH"
const defaultTTL = 300

type Config struct {
	ApiKey     string `json:"api_token"`
	ZoneId     string `json:"zone_id"`
	Subdomains []struct {
		Name    string `json:"name"`
		Proxied bool   `json:"proxied"`
	} `json:"subdomains"`
	PurgeUnknownRecords bool `json:"purgeUnknownRecords"`
	Ttl                 int  `json:"ttl"`
}
type Client struct {
	client *http.Client
	apiUrl string
	ip4Url string
}
type CfDDns struct {
	config     *Config
	configPath string
	api        *Client
}

func NewCfDDns() *CfDDns {
	configPath = os.Getenv(configPathEnv)
	if configPath == "" {
		configPath = "."
	}

	return &CfDDns{
		configPath: configPath,
		api: &Client{
			client: http.DefaultClient,
			apiUrl: "https://api.cloudflare.com/client/v4/",
			ip4Url: "https://1.1.1.1/cdn-cgi/trace",
		},
	}
}

var (
	stopSignalChan = make(chan os.Signal, 1)
	config         *Config
	configPath     string
)

type GracefulExit struct {
	stop chan bool
	wg   sync.WaitGroup
}

func NewGracefulExit() *GracefulExit {
	exit := &GracefulExit{
		stop: make(chan bool),
	}
	signal.Notify(stopSignalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopSignalChan
		fmt.Println("🛑 Stopping main thread...")
		close(exit.stop)
	}()
	return exit
}

func (d *CfDDns) OverrideClient(client *Client) {
	d.api = client
}

func (d *CfDDns) LoadConfig() *CfDDns {

	configData, err := os.ReadFile(d.configPath + "/config.json")
	if err != nil {
		fmt.Println("😡 Error reading config.json")
		time.Sleep(10 * time.Second)
		os.Exit(1)
	}
	if err := json.Unmarshal(configData, &d.config); err != nil {
		fmt.Println("😡 Error parsing config.json")
		time.Sleep(10 * time.Second)
		os.Exit(1)
	}

	if d.config.Ttl < 30 {
		d.config.Ttl = defaultTTL
		fmt.Sprintf("⚙️ TTL is too low or missing - defaulting to %d (auto)", defaultTTL)
	}
	return d
}

func (d *CfDDns) FetchIP() string {

	resp, err := http.Get(d.api.ip4Url)

	defer resp.Body.Close()
	if err != nil {
		fmt.Printf("🧩 Error fetching IP from %s: %v\n", d.api.ip4Url, err)
		return ""
	}

	body, _ := io.ReadAll(resp.Body)
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ip=") {
			return strings.TrimPrefix(line, "ip=")
		}
	}
	return ""
}

func (d *CfDDns) apiRequest(endpoint, method string, data any) ([]byte, error) {
	var reqBody []byte
	var err error
	if data != nil {
		reqBody, err = json.Marshal(&data)
		if err != nil {
			return nil, err
		}
	}

	apiURL := d.api.apiUrl + endpoint
	req, err := http.NewRequest(method, apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	if d.config.ApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+d.config.ApiKey)
	} else {
		return nil, errors.New("missing API key")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := d.api.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return io.ReadAll(resp.Body)
	}

	body, _ := io.ReadAll(resp.Body)
	return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
}

type ZoneResult struct {
	Result struct {
		Name string `json:"name"`
	} `json:"result"`
}

func (d *CfDDns) GetExistingRecords() *DnsRecords {
	// Get existing DNS records
	dnsRecordsData, err := d.apiRequest("/zones/"+d.config.ZoneId+"/dns_records?per_page=100&type=A", "GET", nil)
	if err != nil {
		fmt.Println("😡 Error fetching DNS records:", err)
		return nil
	}

	var dnsRecordsResponse DnsRecords
	if err := json.Unmarshal(dnsRecordsData, &dnsRecordsResponse); err != nil {
		fmt.Println("😡 Error parsing DNS records response:", err)
		return nil
	}

	return &dnsRecordsResponse
}
func (d *CfDDns) Run(ip string) bool {

	zone := d.GetZone()
	if zone == nil {
		return false
	}

	dnsRecordsResponse := d.GetExistingRecords()
	if dnsRecordsResponse == nil {
		return false
	}

	for _, subdomain := range d.config.Subdomains {

		name := strings.TrimSpace(strings.ToLower(subdomain.Name))

		fqdn := zone.Result.Name
		if name != "" && name != "@" {
			fqdn = name + "." + zone.Result.Name
		}

		record := DnsPayload{
			Type:    "A",
			Name:    fqdn,
			Content: ip,
			Proxied: subdomain.Proxied,
			Ttl:     d.config.Ttl,
		}

		var identifier string
		var modified bool
		var duplicateIds []string

		if dnsRecordsResponse.Result != nil {
			for _, r := range dnsRecordsResponse.Result {
				if r.Name == fqdn {
					if identifier != "" {
						if r.Content == ip {
							duplicateIds = append(duplicateIds, identifier)
							identifier = r.Id
						} else {
							duplicateIds = append(duplicateIds, r.Id)
						}
					} else {
						identifier = r.Id
						if r.Content != record.Content || r.Proxied != record.Proxied {
							modified = true
						}
					}
				}
			}
		}

		// Add or update records
		if identifier != "" {
			if modified {
				fmt.Println("📡 Updating record", record)
				_, err := d.apiRequest("zones/"+d.config.ZoneId+"/dns_records/"+identifier, "PUT", record)
				if err != nil {
					fmt.Println("😡 Error updating record:", err)
				}
			}
		} else {
			fmt.Println("➕ Adding new record", record)
			_, err := d.apiRequest("zones/"+d.config.ZoneId+"/dns_records", "POST", record)
			if err != nil {
				fmt.Println("😡 Error adding new record:", err)
			}
		}

		// Delete stale records if enabled
		if d.config.PurgeUnknownRecords {
			for _, duplicateId := range duplicateIds {
				fmt.Println("🗑️ Deleting stale record", duplicateId)
				_, err := d.apiRequest("zones/"+d.config.ZoneId+"/dns_records/"+duplicateId, "DELETE", nil)
				if err != nil {
					fmt.Println("😡 Error deleting stale record:", err)
				}
			}
		}
	}

	return true
}

func (d *CfDDns) GetZone() *ZoneResult {
	// Get the zone information
	responseData, err := d.apiRequest("/zones/"+d.config.ZoneId, "GET", d.config.Subdomains)
	if err != nil {
		fmt.Println("😡 Error fetching zone information:", err)
		time.Sleep(5 * time.Second)
		return nil
	}

	var response ZoneResult
	if err := json.Unmarshal(responseData, &response); err != nil || response.Result.Name == "" {
		fmt.Println("😡 Invalid response or zone name missing.")
		time.Sleep(5 * time.Second)
		return nil
	}
	return &response
}

func main() {

	ddns := NewCfDDns().LoadConfig()
	gracefulExit := NewGracefulExit()

	if len(os.Args) > 1 && os.Args[1] == "--repeat" {
		ticker := time.NewTicker(time.Duration(ddns.config.Ttl) * time.Second)
		for {
			select {
			case <-ticker.C:
				ddns.Run(ddns.FetchIP())
			case <-gracefulExit.stop:
				ticker.Stop()
				fmt.Println("Stopped Cloudflare DDNS updater.")
				return
			}
		}
	} else {
		ddns.Run(ddns.FetchIP())
	}
}
