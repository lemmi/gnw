package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/lemmi/closer"
	"github.com/prometheus/procfs"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// VERSION gnw version string
const VERSION = "gnw-0.0.1"

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
		Uptime                       int64    `xml:"uptime"`
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
		Interfaces []Interface
	} `xml:"interface_data"`
	BatmanAdvInterfaces  string `xml:"batman_adv_interfaces"`
	BatmanAdvOriginators string `xml:"batman_adv_originators"`
	BatmanAdvGatewayMode string `xml:"batman_adv_gateway_mode"`
	BatmanAdvGatewayList string `xml:"batman_adv_gateway_list"`
	BabelNeighbours      struct {
		Neighbours []BabelNeighbour `xml:"neighbour"`
	} `xml:"babel_neighbours"`
	ClientCount int `xml:"client_count"`
	Clients     struct {
		Num []ClientNum
	} `xml:"clients"`
}

// BabelNeighbour is used for xml encoding
type BabelNeighbour struct {
	MacAddr           string `xml:",chardata"`
	OutgoingInterface string `xml:"outgoing_interface"`
	LinkCost          string `xml:"link_cost"`
}

// Interface is used for xml encoding
type Interface struct {
	XMLName   xml.Name
	Name      string `xml:"name"`
	Mtu       int    `xml:"mtu"`
	MacAddr   string `xml:"mac_addr"`
	TrafficRx uint64 `xml:"traffic_rx"`
	TrafficTx uint64 `xml:"traffic_tx"`
}

// ClientNum is used for xml encoding
type ClientNum struct {
	XMLName xml.Name
	N       int `xml:",chardata"`
}

func getBabelInfo() (string, []BabelNeighbour) {
	conn, err := net.Dial("tcp6", "[::1]:33123")
	if err != nil {
		return "", nil
	}
	defer closer.WithStackTrace(conn)

	go fmt.Fprintln(conn, "dump")

	scanner := bufio.NewScanner(conn)

	var version string
	var neighs []BabelNeighbour
	// skip the startup "ok"
	for scanner.Scan() {
		if scanner.Text() == "ok" {
			break
		}
	}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 1 && fields[0] == "ok" {
			break
		}
		if len(fields) > 1 && fields[0] == "version" {
			version = strings.Join(fields[1:], " ")
		}
		if fields[0] == "add" && fields[1] == "neighbour" {
			neighs = append(neighs, BabelNeighbour{
				MacAddr:           fields[4],
				OutgoingInterface: fields[6],
				LinkCost:          fields[len(fields)-1],
			})
		}
	}
	if scanner.Err() != nil {
		return version, nil
	}

	fmt.Fprintln(conn, "quit")

	return version, neighs
}

func crawl(c Config) (d Data, err error) {
	stat, err := procfs.NewStat()
	if err != nil {
		return
	}

	{
		var mem meminfo
		mem, err = readMeminfo()
		if err != nil {
			return
		}

		var load loadavg
		load, err = readLoadavg()
		if err != nil {
			return
		}

		var sysinfo unix.Sysinfo_t
		if err = unix.Sysinfo(&sysinfo); err != nil {
			return
		}

		d.SystemData.Status = "online"
		d.SystemData.Idletime = stat.CPUTotal.Idle
		d.SystemData.Loadavg = load.load15
		d.SystemData.LocalTime = time.Now().Unix()
		d.SystemData.MemoryBuffering = mem.Buffers
		d.SystemData.MemoryCaching = mem.Cached
		d.SystemData.MemoryFree = mem.MemFree
		d.SystemData.MemoryTotal = mem.MemTotal
		d.SystemData.Processes = fmt.Sprintf("%d/%d", load.runnable, load.procs)
		d.SystemData.Uptime = sysinfo.Uptime
	}

	{
		var utsname unix.Utsname
		if err = unix.Uname(&utsname); err != nil {
			return
		}
		d.SystemData.KernelVersion = string(bytes.Trim(utsname.Release[:], "\x00"))
	}

	ifs, err := net.Interfaces()
	if err != nil {
		return
	}

	for _, i := range ifs {
		// skip lo
		if i.Name == "lo" {
			continue
		}

		link, err := netlink.LinkByIndex(i.Index)
		if err != nil {
			return d, err
		}

		if link.Attrs().Flags&net.FlagUp == 0 {
			continue
		}

		d.InterfaceData.Interfaces = append(d.InterfaceData.Interfaces, Interface{
			XMLName: xml.Name{
				Local: i.Name,
			},
			Name:      i.Name,
			Mtu:       i.MTU,
			MacAddr:   i.HardwareAddr.String(),
			TrafficRx: link.Attrs().Statistics.RxBytes,
			TrafficTx: link.Attrs().Statistics.TxBytes,
		})

		// only run neighbour discovery on layer2 devices
		if len(bytes.Trim(i.HardwareAddr, "\x00")) == 0 {
			continue
		}

		neighs, err := netlink.NeighList(i.Index, netlink.FAMILY_ALL)
		if err != nil {
			return d, err
		}

		var neighProbe []net.IP
		for _, neigh := range neighs {
			if neigh.State&netlink.NUD_REACHABLE == 0 {
				if neigh.IP.IsLinkLocalUnicast() {
					neighProbe = append(neighProbe, neigh.IP)
				}
			}
		}

		nc, err := newNDP()
		if err != nil {
			return d, err
		}
		defer nc.Close()
		_, err = nc.solicit(2*time.Second, &i, neighProbe...)
		if err != nil {
			return d, err
		}

		neighAddrs := map[string]struct{}{}
		neighs, err = netlink.NeighList(i.Index, netlink.FAMILY_ALL)
		if err != nil {
			return d, err
		}
		for _, neigh := range neighs {
			if neigh.State&netlink.NUD_REACHABLE > 0 {
				neighAddrs[neigh.HardwareAddr.String()] = struct{}{}
			}
		}

		count := len(neighAddrs)
		d.ClientCount += count
		d.Clients.Num = append(d.Clients.Num, ClientNum{
			XMLName: xml.Name{
				Local: i.Name,
			},
			N: count,
		})
	}

	d.SystemData.BabelVersion, d.BabelNeighbours.Neighbours = getBabelInfo()

	d.SystemData.Hostname = c.Hostname
	d.SystemData.Description = c.Description
	d.SystemData.Geo.Lat = c.Lat
	d.SystemData.Geo.Lng = c.Lng
	d.SystemData.PositionComment = c.PositionComment
	d.SystemData.Hood = c.Hood
	d.SystemData.Contact = c.Contact
	d.SystemData.Distname = c.Distname
	d.SystemData.Distversion = c.Distversion
	d.SystemData.FirmwareVersion = "Generic"
	d.SystemData.NodewatcherVersion = VERSION

	// unused
	d.SystemData.Chipset = ""
	d.SystemData.CPU = []string(nil)
	d.SystemData.Model = ""
	d.SystemData.Hoodid = ""
	d.SystemData.BatmanAdvancedVersion = ""
	d.SystemData.FirmwareRevision = ""
	d.SystemData.OpenwrtCoreRevision = ""
	d.SystemData.OpenwrtFeedsPackagesRevision = ""
	d.SystemData.VpnActive = 0

	return d, err
}

func wrapInJSON(d Data, xpayload []byte) []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, `{%q: {%q: %q}}`,
		"64",
		d.InterfaceData.Interfaces[0].MacAddr,
		"<?xml version='1.0' standalone='yes'?>"+string(xpayload),
	)

	return buf.Bytes()
}

func sendReport(c Config, payload []byte) error {
	req, err := http.NewRequest("POST", "https://monitoring.freifunk-franken.de/api/alfred", bytes.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	if c.Debug {
		c.Log.Println()
		c.Log.Println("POST:")
		c.Log.Println()
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			return err
		}
		c.Log.Println(string(dump))
	}

	if !c.Dry {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer closer.WithStackTrace(resp.Body)

		if c.Debug {
			c.Log.Println()
			c.Log.Println("HTTP Response:")
			c.Log.Println()
			if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
				c.Log.Println(err)
				return err
			}
		}
	}

	return nil
}

func prepareReport(c Config) ([]byte, error) {
	d, err := crawl(c)
	if err != nil {
		return nil, err
	}

	if c.Debug {
		c.Log.Println("XML Output:")
		e := xml.NewEncoder(os.Stdout)
		e.Indent("", "\t")
		if err := e.Encode(d); err != nil {
			return nil, err
		}
		c.Log.Println()
	}

	xpayload, err := xml.Marshal(d)
	if err != nil {
		return nil, err
	}

	if c.Debug {
		c.Log.Println()
		c.Log.Println("XML Payload:")
		c.Log.Println()
		c.Log.Println(string(xpayload))
	}

	return wrapInJSON(d, xpayload), nil
}

func main() {
	c, err := getConfig()
	if err != nil {
		c.Log.Println(err)
		os.Exit(1)
	}

	c.Log.Println("Starting Nodewatcher")

	maxRetries := uint(6)
	for {
		c.Log.Println("Sending Report")
		for retries := uint(1); retries <= maxRetries; retries++ {
			payload, err := prepareReport(c)
			if err != nil {
				c.Log.Println("Failed to gather node information")
				c.Log.Println(err)
				os.Exit(1)
			}
			err = sendReport(c, payload)
			if err == nil {
				c.Log.Println("Successfully sent report")
				break
			}

			if retries == maxRetries {
				c.Log.Println("Failed to send report, giving up")
				break
			}

			delay := time.Second << (retries - 1)
			c.Log.Printf("Failed to send Report, retrying in %s", delay)
			time.Sleep(delay)
		}
		runtime.GC()
		time.Sleep(5 * time.Minute)
	}
}
