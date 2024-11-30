package main

import (
	"encoding/json"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
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
	suite.Equal("www", records.Result[1].Name)
}
func (suite *DDnsSuite) TestGetZone() {
	records := suite.Api.GetZone()
	suite.Equal("example.com", records.Result.Name)
}

func (suite *DDnsSuite) SetupTest() {
	_ = os.Setenv("DDNS_CONFIG_PATH", "./stubs")
	suite.mux = http.NewServeMux()
	suite.server = httptest.NewServer(suite.mux)
	suite.Api = NewCfDDns()
	suite.Api.OverrideClient(&Client{
		client: http.DefaultClient,
		apiUrl: suite.server.URL,
		ip4Url: suite.server.URL + "/cdn-cgi/trace",
	})
	suite.Api.LoadConfig()

	suite.mux.HandleFunc("/cdn-cgi/trace", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("h=1.1.1.1\n" +
			"ip=192.168.1.1\n" +
			"ts=1732864022.564\n" +
			"visit_scheme=https\n"))
	})

	suite.mux.HandleFunc("/zones/", func(w http.ResponseWriter, r *http.Request) {
		re := regexp.MustCompile("/zones/([0-9]+)/dns_records")
		reZones := regexp.MustCompile("/zones/([0-9]+)")
		matches := re.FindStringSubmatch(r.URL.Path)
		if matches != nil {

			dnsResults := &DnsRecords{
				Result: []DnsResult{
					{
						Comment:   "Domain verification record",
						Name:      "example.com",
						Proxied:   true,
						Ttl:       3600,
						Content:   "192.168.1.2",
						Type:      "A",
						Id:        "1234",
						Proxiable: true,
					},
					{
						Comment:   "Domain verification record",
						Name:      "www",
						Proxied:   true,
						Ttl:       3600,
						Content:   "192.168.1.2",
						Type:      "A",
						Id:        "1234",
						Proxiable: true,
					},
				},
			}
			resp, _ := json.Marshal(dnsResults)
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			return
		}
		matches = reZones.FindStringSubmatch(r.URL.Path)
		if matches != nil {
			zoneResults := ZoneResult{
				Result: struct {
					Name string `json:"name"`
				}{Name: "example.com"}}

			resp, _ := json.Marshal(zoneResults)
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
		}
	})

}

func TestDDnsSuite(t *testing.T) {
	suite.Run(t, new(DDnsSuite))
}
