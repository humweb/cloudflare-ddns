package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type CfDDns struct {
	config     *Config
	configPath string
	api        *Client
}

func (d *CfDDns) Config() *Config {
	return d.config
}

// NewCfDDns init new ddns service
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

// OverrideClient option to override default http client
func (d *CfDDns) OverrideClient(client *Client) *CfDDns {
	d.api = client
	return d
}

// LoadConfig load ddns config file
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

// FetchIP get public IP address
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

// apiRequest helper to send api requests to cloudflare
func (d *CfDDns) apiRequest(endpoint, method string, data any) ([]byte, error) {
	var reqBody []byte
	var err error
	if data != nil {
		reqBody, err = json.Marshal(&data)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, d.api.apiUrl+endpoint, bytes.NewBuffer(reqBody))
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

// GetExistingRecords fetch existing records from cloudflare
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

// FindMatchingRecords find matching records
func (d *CfDDns) FindMatchingRecords(fqdn, ip string, record DnsPayload, results []DnsResult) (string, bool, []string) {
	var (
		identifier   string
		modified     bool
		duplicateIds []string
	)
	for _, r := range results {
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
				if r.Content != record.Content || r.Ttl != record.Ttl || r.Proxied != record.Proxied {
					modified = true
				}
			}
		}
	}
	return identifier, modified, duplicateIds
}

// addRecord add a new record
func (d *CfDDns) addRecord(record DnsPayload) bool {
	fmt.Println("➕ Adding new record:", record)
	_, err := d.apiRequest("zones/"+d.config.ZoneId+"/dns_records", "POST", record)
	return err == nil
}

// updateRecord update an existing record
func (d *CfDDns) updateRecord(identifier string, record DnsPayload) bool {
	fmt.Println("📡 Updating record:", record)
	_, err := d.apiRequest("zones/"+d.config.ZoneId+"/dns_records/"+identifier, "PUT", record)
	return err == nil
}

// deleteStaleRecords delete stale records
func (d *CfDDns) deleteStaleRecords(duplicateIds []string) {
	for _, duplicateId := range duplicateIds {
		fmt.Println("🗑️ Deleting stale record:", duplicateId)
		_, err := d.apiRequest("zones/"+d.config.ZoneId+"/dns_records/"+duplicateId, "DELETE", nil)
		if err != nil {
			fmt.Println("😡 Error deleting stale record:", err)
		}
	}
}

// GetFullDomain get fully qualified domain name
func (d *CfDDns) GetFullDomain(subdomain string, zone *ZoneResult) string {
	name := strings.TrimSpace(strings.ToLower(subdomain))

	fqdn := zone.Result.Name
	if name != "" && name != "@" {
		fqdn = name + "." + zone.Result.Name
	}
	return fqdn
}

// GetZone get zone info from cloudflare
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

func (d *CfDDns) Run() bool {
	ip := d.FetchIP()
	zone := d.GetZone()
	if zone == nil {
		fmt.Println("❌ Zone not found.")
		return false
	}

	dnsRecordsResponse := d.GetExistingRecords()
	if dnsRecordsResponse == nil {
		fmt.Println("❌ Failed to fetch existing DNS records.")
		return false
	}

	for _, subdomain := range d.config.Subdomains {
		fqdn := d.GetFullDomain(subdomain.Name, zone)

		record := DnsPayload{
			Type:    "A",
			Name:    fqdn,
			Content: ip,
			Proxied: subdomain.Proxied,
			Ttl:     d.config.Ttl,
		}

		identifier, modified, duplicateIds := d.FindMatchingRecords(fqdn, ip, record, dnsRecordsResponse.Result)

		// Handle record addition or update
		if identifier == "" {
			if !d.addRecord(record) {
				fmt.Println("❌ Failed to add new record for", fqdn)
			}
		} else if modified && !d.updateRecord(identifier, record) {
			fmt.Println("❌ Failed to update record for", fqdn)
		}

		// Purge stale records if enabled
		if d.config.PurgeUnknownRecords {
			d.deleteStaleRecords(duplicateIds)
		}
	}

	return true
}
