/*
Package alfredxml provided types to marshal and unmarshal alfred monitoring data.

Example to decode a stream of monitoring data from an io.Reader:

	...

	dec := json.NewDecoder(r)
	for {
		var a alfredxml.Alfred

		if err := dec.Decode(&a); err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		for mac, data := range a {
			...
		}
	}

	...
*/
package alfredxml

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
)

// Data is used xml encoding
type Data struct {
	XMLName    xml.Name `xml:"data"`
	SystemData struct {
		Status      string `xml:"status"`
		Hostname    string `xml:"hostname"`
		Description string `xml:"description"`
		Geo         struct {
			Lat float64 `xml:"lat"`
			Lng float64 `xml:"lng"`
		} `xml:"geo"`
		PositionComment              string   `xml:"position_comment"`
		Contact                      string   `xml:"contact"`
		Hood                         string   `xml:"hood"`
		Hoodid                       string   `xml:"hoodid"`
		Distname                     string   `xml:"distname"`
		Distversion                  string   `xml:"distversion"`
		Chipset                      string   `xml:"chipset"`
		CPU                          []string `xml:"cpu"`
		Model                        string   `xml:"model"`
		MemoryTotal                  int      `xml:"memory_total"`
		MemoryAvailable              int      `xml:"memory_available"`
		MemoryFree                   int      `xml:"memory_free"`
		MemoryBuffering              int      `xml:"memory_buffering"`
		MemoryCaching                int      `xml:"memory_caching"`
		Loadavg                      float64  `xml:"loadavg"`
		Processes                    string   `xml:"processes"`
		Uptime                       float64  `xml:"uptime"`
		Idletime                     float64  `xml:"idletime"`
		LocalTime                    int64    `xml:"local_time"`
		BabelVersion                 string   `xml:"babel_version"`
		BatmanAdvancedVersion        string   `xml:"batman_advanced_version"`
		KernelVersion                string   `xml:"kernel_version"`
		NodewatcherVersion           string   `xml:"nodewatcher_version"`
		FirmwareVersion              string   `xml:"firmware_version"`
		FirmwareRevision             string   `xml:"firmware_revision"`
		OpenwrtCoreRevision          string   `xml:"openwrt_core_revision"`
		OpenwrtFeedsPackagesRevision string   `xml:"openwrt_feeds_packages_revision"`
		VpnActive                    int      `xml:"vpn_active"`
	} `xml:"system_data"`
	InterfaceData struct {
		Interfaces []Interface `xml:",any"`
	} `xml:"interface_data"`
	BatmanAdvInterfaces struct {
		Interfaces []BatmanAdvInterface `xml:",any"`
	} `xml:"batman_adv_interfaces"`
	BatmanAdvOriginators struct {
		Originators []BatmanAdvOriginator `xml:",any"`
	} `xml:"batman_adv_originators"`
	BatmanAdvGatewayMode string `xml:"batman_adv_gateway_mode"`
	BatmanAdvGatewayList struct {
		Gateways []BatmanAdvGateway `xml:",any"`
	} `xml:"batman_adv_gateway_list"`
	BabelNeighbours struct {
		Neighbours []BabelNeighbour `xml:"neighbour"`
	} `xml:"babel_neighbours"`
	ClientCount int `xml:"client_count"`
	Clients     struct {
		Num []ClientNum `xml:",any"`
	} `xml:"clients"`
}

// BabelNeighbour is used for xml encoding
type BabelNeighbour struct {
	IP                string `xml:"ip"`
	OutgoingInterface string `xml:"outgoing_interface"`
	LinkCost          string `xml:"link_cost"`
}

// Interface is used for xml encoding
type Interface struct {
	XMLName           xml.Name
	Name              string   `xml:"name,omitempty"`
	Mtu               int      `xml:"mtu,omitempty"`
	MacAddr           string   `xml:"mac_addr,omitempty"`
	TrafficRx         uint64   `xml:"traffic_rx,omitempty"`
	TrafficTx         uint64   `xml:"traffic_tx,omitempty"`
	IPv4Addr          []string `xml:"ipv4_addr,omitempty"`
	IPv6Addr          []string `xml:"ipv6_addr,omitempty"`
	IPv6LinkLocalAddr []string `xml:"ipv6_link_local_addr,omitempty"`
	WlanMode          string   `xml:"wlan_mode,omitempty"`
	WlanTxPower       string   `xml:"wlan_tx_power,omitempty"`
	WlanSsid          string   `xml:"wlan_ssid,omitempty"`
	WlanType          string   `xml:"wlan_type,omitempty"`
	WlanChannel       string   `xml:"wlan_channel,omitempty"`
	WlanWidth         string   `xml:"wlan_width,omitempty"`
}

// BatmanAdvInterface is used for xml coding
type BatmanAdvInterface struct {
	XMLName xml.Name
	Name    string `xml:"name,omitempty"`
	Status  string `xml:"status,omitempty"`
}

// BatmanAdvOriginator is used for xml coding
type BatmanAdvOriginator struct {
	XMLName           xml.Name
	Originator        string `xml:"originator,omitempty"`
	LinkQuality       string `xml:"link_quality,omitempty"`
	Nexthop           string `xml:"nexthop,omitempty"`
	LastSeen          string `xml:"last_seen,omitempty"`
	OutgoingInterface string `xml:"outgoing_interface,omitempty"`
}

// BatmanAdvGateway is used for xml coding
type BatmanAdvGateway struct {
	XMLName           xml.Name
	Selected          string `xml:"selected,omitempty"`
	Gateway           string `xml:"gateway,omitempty"`
	LinkQuality       string `xml:"link_quality,omitempty"`
	Nexthop           string `xml:"nexthop,omitempty"`
	OutgoingInterface string `xml:"outgoing_interface,omitempty"`
	GwClass           string `xml:"gw_class,omitempty"`
}

// ClientNum is used for xml encoding
type ClientNum struct {
	XMLName xml.Name
	N       int `xml:",chardata"`
}

// UnmarshalJSON decodes the xml embedded in the json string
func (d *Data) UnmarshalJSON(b []byte) error {
	if bytes.Equal([]byte("null"), b) {
		return nil
	}

	var s string

	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	return xml.Unmarshal([]byte(s), d)
}

// Type to facilitate encoding via standard lib encoders
type alfredData Data

// MarshalText encodes the data into an xml string
func (ad alfredData) MarshalText() ([]byte, error) {
	d := Data(ad)
	var buf bytes.Buffer

	if _, err := buf.WriteString("<?xml version='1.0' standalone='yes'?>"); err != nil {
		return buf.Bytes(), err
	}

	err := xml.NewEncoder(&buf).Encode(d)
	return buf.Bytes(), err
}

/*
Alfred is used to deserialize monitoring data dumped directly from alfred-json

The expected format is a stream of json objects with a single "64" element that
contains a map from mac addresses to thair monitoring data:

	{
		"64": {
			"AA:BB:CC:DD:EE:FF": "<xml...>",
			"00:11:22:33:44:55": "<xml...>",
			...
		}
	}
	{
		"64": {
			...
		}
	}
*/
type Alfred map[string]Data

// UnmarshalJSON for alfred data
func (a *Alfred) UnmarshalJSON(b []byte) error {
	wrap := struct {
		Wrap *map[string]Data `json:"64"`
	}{
		Wrap: (*map[string]Data)(a),
	}

	return json.Unmarshal(b, &wrap)
}

/*
Alfred2 is used to deserialize monitoring data inteded for the alfred2 endpoint

The expected format is a stream of json object from mac addresses to monitoring
data

	{
		"AA:BB:CC:DD:EE:FF": "<xml...>",
		"00:11:22:33:44:55": "<xml...>",
		...
	}
	{
		...
	}
*/
type Alfred2 map[string]Data

// UnmarshalJSON for alfred2 data
func (a *Alfred2) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, (*map[string]Data)(a))
}

// Alfred2Slice collects data from multiple devices
type Alfred2Slice []Data

// MarshalJSON encodes all monitoring data for the alfred2 api endpoint
func (a2s Alfred2Slice) MarshalJSON() ([]byte, error) {
	a2 := make(map[string]alfredData, len(a2s))
	for _, d := range a2s {
		a2[d.InterfaceData.Interfaces[0].MacAddr] = alfredData(d)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(a2)

	return buf.Bytes(), err
}
