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
	*a = make(Alfred)
	wrap := struct {
		Wrap map[string]Data `json:"64"`
	}{
		Wrap: *a,
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
	m := map[string]Data(*a)
	return json.Unmarshal(b, &m)
}
