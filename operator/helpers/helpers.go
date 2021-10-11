package helpers

import (
	"fmt"
	"strings"
)

type DNSRecord struct {
	Found   bool
	Type    string
	Records []string
	Err     error
}

// DNSActions are all crud actions for DNS Records
type DNSActions interface {
	GetARecord(recordName string, dnsZone string) DNSRecord
	CreateARecord(recordName string, dnsZone string, records []string) error
	UpdateARecord(recordName string, dnsZone string, records []string) error
	DeleteARecord(recordName string, dnsZone string) error
}

func GetDnsZone(recordName string) (string, error) {
	if len(recordName) == 0 {
		return "", fmt.Errorf("DNS record name not found")
	}

	if !strings.Contains(recordName, ".") {
		// The node does not contain dots, so it is not a FQDN name
		return "", fmt.Errorf("not FQDN found in record name: %s", recordName)
	}

	// Get the zone, the last part
	fqdnParts := strings.Split(recordName, ".")

	//if len(fqdnParts) == 0 {
	//	dnsRecord.Err = fmt.Errorf("not FQDN found in record name: %s", recordName)
	//	return dnsRecord
	//}

	dnsZone := fqdnParts[len(fqdnParts)-1]

	return dnsZone, nil
}
