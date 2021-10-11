package powerdns

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jaberchez/k8s-operator-dns-ops/helpers"
)

type PowerDNS struct {
	Server string
	Key    string
}

type httpRequest struct {
	client http.Client
	req    *http.Request
}

func (p *PowerDNS) GetARecord(recordName string, dnsZone string) helpers.DNSRecord {
	var dnsRecord helpers.DNSRecord

	dnsRecord.Found = false

	// Concatenate the dot
	recordName = recordName + "."

	r, err := p.getHttpRequest(dnsZone, "GET", nil)

	if err != nil {
		dnsRecord.Err = err
		return dnsRecord
	}

	resp, err := r.client.Do(r.req)

	if err != nil {
		dnsRecord.Err = fmt.Errorf("Error %s", err)
		return dnsRecord
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		dnsRecord.Err = fmt.Errorf("PowerDNS API call error: %s", resp.Status)
		return dnsRecord
	}

	var target map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&target)

	if err != nil {
		dnsRecord.Err = errors.New(err.Error())
		return dnsRecord
	}

	if _, ok := target["rrsets"]; !ok {
		dnsRecord.Err = errors.New("rrsets field nof found in response")
		return dnsRecord
	}

	// rrsets is an array of interface{}
	rrsets := target["rrsets"].([]interface{})

	for i := range rrsets {
		m := rrsets[i].(map[string]interface{})

		// Get record name
		name := m["name"].(string)

		if name == recordName {
			dnsRecord.Found = true

			recordType := m["type"].(string)

			if recordType == "A" {
				dnsRecord.Type = recordType

				records := m["records"].([]interface{})

				// Add all records for A type
				for _, r := range records {
					t := r.(map[string]interface{})
					content := t["content"].(string)
					dnsRecord.Records = append(dnsRecord.Records, content)
				}

				break
			}
		}
	}

	return dnsRecord
}

func (p *PowerDNS) DeleteARecord(recordName string, dnsZone string) error {
	d := fmt.Sprintf(`{"rrsets": [{"changetype": "DELETE", "type": "A", "name": "%s."}]}`, recordName)

	data := []byte(d)
	r, err := p.getHttpRequest(dnsZone, "PATCH", data)

	if err != nil {
		return err
	}

	resp, err := r.client.Do(r.req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	statusCode := strconv.Itoa(resp.StatusCode)

	switch statusCode[0:1] {
	case "2":
		break
	default:
		return fmt.Errorf("PowerDNS API call error: %s", resp.Status)
	}

	return nil
}

func (p *PowerDNS) CreateARecord(recordName string, dnsZone string, records []string) error {
	return p.updateCreateARecord(recordName, dnsZone, records)
}

func (p *PowerDNS) UpdateARecord(recordName string, dnsZone string, records []string) error {
	return p.updateCreateARecord(recordName, dnsZone, records)
}

func (p *PowerDNS) updateCreateARecord(recordName string, dnsZone string, records []string) error {
	var b bytes.Buffer

	// Create the data string in JSON format
	b.WriteString(fmt.Sprintf(`{"rrsets": [{"changetype": "REPLACE","type": "A","name": "%s.","ttl": 86400,"records": [`, recordName))

	for _, record := range records {
		b.WriteString(fmt.Sprintf(`{"content": "%s", "disabled": false},`, record))
	}

	// Delete last comma
	s := strings.TrimRight(b.String(), ",")

	// Add last part of the json data
	s = s + "]}]}"

	r, err := p.getHttpRequest(dnsZone, "PATCH", []byte(s))

	if err != nil {
		return err
	}

	resp, err := r.client.Do(r.req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	statusCode := strconv.Itoa(resp.StatusCode)

	switch statusCode[0:1] {
	case "2":
		break
	default:
		return fmt.Errorf("PowerDNS API call error: %s", resp.Status)
	}

	return nil
}

func (p *PowerDNS) getHttpRequest(dnsZone string, requestType string, jsonStr []byte) (httpRequest, error) {
	httpReq := httpRequest{}
	req := &http.Request{}
	err := *new(error)

	url := fmt.Sprintf("%s/api/v1/servers/localhost/zones/%s.", p.Server, dnsZone)

	client := http.Client{Timeout: time.Second * 10}
	httpReq.client = client

	if jsonStr == nil {
		req, err = http.NewRequest(requestType, url, nil)
	} else {
		req, err = http.NewRequest(requestType, url, bytes.NewBuffer(jsonStr))
	}

	if err != nil {
		return httpReq, fmt.Errorf("Error %s", err)
	}

	// Set Header for authentication against the PowerDNS API
	req.Header.Add("X-API-Key", p.Key)
	httpReq.req = req

	return httpReq, nil
}
