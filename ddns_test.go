package main

import (
	"encoding/json"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Define the suite, and absorb the built-in basic suite
// returns the current testing context
type DDnsSuite struct {
	suite.Suite
	Api    *CfDDns
	mux    *http.ServeMux
	server *httptest.Server
}

func (suite *DDnsSuite) TestFetchIp() {
	ip := suite.Api.FetchIP()
	suite.Equal("192.168.1.1", ip)
}

func (suite *DDnsSuite) TestGetExistingRecords() {
	records := suite.Api.GetExistingRecords()
	suite.Equal("example.com", records.Result[0].Name)
	suite.Equal("www.example.com", records.Result[1].Name)
}
func (suite *DDnsSuite) TestGetZone() {
	records := suite.Api.GetZone()
	suite.Equal("example.com", records.Result.Name)
}

func (suite *DDnsSuite) TestUpdates() {

	ip := "192.168.1.1"
	zone := suite.Api.GetZone()
	records := suite.Api.GetExistingRecords()
	for _, subdomain := range suite.Api.Config().Subdomains {
		fqdn := suite.Api.GetFullDomain(subdomain.Name, zone)

		record := DnsPayload{
			Type:    "A",
			Name:    fqdn,
			Content: ip,
			Proxied: subdomain.Proxied,
			Ttl:     suite.Api.Config().Ttl,
		}

		if subdomain.Name == "www" {
			identifier, modified, duplicateIds := suite.Api.FindMatchingRecords(fqdn, ip, record, records.Result)
			suite.True(modified)
			suite.Equal("2", identifier)
			suite.Equal(0, len(duplicateIds))
		}
		if subdomain.Name == "admin" {
			identifier, modified, duplicateIds := suite.Api.FindMatchingRecords(fqdn, ip, record, records.Result)
			suite.False(modified)
			suite.Equal("", identifier)
			suite.Equal(0, len(duplicateIds))
		}
		//suite.Equal("example.com", records.Result.Name)
	}
}

func (suite *DDnsSuite) SetupTest() {
	_ = os.Setenv("DDNS_CONFIG_PATH", "./stubs")
	suite.mux = http.NewServeMux()
	suite.server = httptest.NewServer(suite.mux)
	suite.Api = NewCfDDns().
		OverrideClient(&Client{
			client: http.DefaultClient,
			apiUrl: suite.server.URL,
			ip4Url: suite.server.URL + "/cdn-cgi/trace",
		}).
		LoadConfig()

	// Get IP
	suite.mux.HandleFunc("/cdn-cgi/trace", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("h=1.1.1.1\n" +
			"ip=192.168.1.1\n" +
			"ts=1732864022.564\n" +
			"visit_scheme=https\n"))
	})

	// Zone info and Existing records
	suite.mux.HandleFunc("/zones/123456789101121314151617181920", func(w http.ResponseWriter, r *http.Request) {
		zoneResults := ZoneResult{
			Result: struct {
				Name string `json:"name"`
			}{Name: "example.com"}}

		resp, _ := json.Marshal(zoneResults)
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	})
	suite.mux.HandleFunc("/zones/123456789101121314151617181920/dns_records", func(w http.ResponseWriter, r *http.Request) {
		// Get existing records
		dnsResults := &DnsRecords{
			Result: []DnsResult{
				{
					Name:      "example.com",
					Proxied:   true,
					Ttl:       3600,
					Content:   "192.168.1.2",
					Type:      "A",
					Id:        "1",
					Proxiable: true,
				},
				{
					Name:      "www.example.com",
					Proxied:   true,
					Ttl:       3600,
					Content:   "192.168.1.1",
					Type:      "A",
					Id:        "2",
					Proxiable: true,
				},
				{
					Name:      "old.example.com",
					Proxied:   true,
					Ttl:       3600,
					Content:   "192.168.1.1",
					Type:      "A",
					Id:        "3",
					Proxiable: true,
				},
			},
		}
		resp, _ := json.Marshal(dnsResults)
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	})
}

func TestDDnsSuite(t *testing.T) {
	suite.Run(t, new(DDnsSuite))
}
