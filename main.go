package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/lemmi/closer"
	alfredxml "github.com/lemmi/gnw/alfredxml"
	"github.com/prometheus/procfs"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"inet.af/netaddr"
)

// VERSION gnw version string
const VERSION = "gnw-0.0.8"

func getBabeldInfo() (string, []alfredxml.BabelNeighbour) {
	const timeout = time.Second * 10
	conn, err := net.DialTimeout("tcp6", "[::1]:33123", timeout)
	if err != nil {
		return "", nil
	}
	defer closer.WithStackTrace(conn)
	conn.SetDeadline(time.Now().Add(timeout))

	go fmt.Fprintln(conn, "dump")

	scanner := bufio.NewScanner(conn)

	var version string
	var neighs []alfredxml.BabelNeighbour
	// skip the startup "ok"
	for scanner.Scan() {
		if scanner.Text() == "ok" {
			break
		}

		fields := strings.Fields(scanner.Text())
		if len(fields) > 1 && fields[0] == "version" {
			version = strings.Join(fields[1:], " ")
		}
	}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 1 && fields[0] == "ok" {
			break
		}
		if fields[0] == "add" && fields[1] == "neighbour" {
			neighs = append(neighs, alfredxml.BabelNeighbour{
				IP:                fields[4],
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

func getBirdInfo() (string, []alfredxml.BabelNeighbour) {
	const timeout = time.Second * 10
	conn, err := net.DialTimeout("unix", "/run/bird/bird.ctl", timeout)
	if err != nil {
		return "", nil
	}
	defer closer.WithStackTrace(conn)
	conn.SetDeadline(time.Now().Add(timeout))

	go func() {
		fmt.Fprintln(conn, "show babel neighbors")
		fmt.Fprintln(conn, "quit")
	}()

	scanner := bufio.NewScanner(conn)

	var version string
	var neighs []alfredxml.BabelNeighbour

	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "8") || strings.HasPrefix(text, "9") {
			break
		}

		fields := strings.Fields(text)
		if len(fields) == 1 && fields[0] == "0000" {
			break
		}
		if len(fields) == 4 &&
			strings.HasPrefix(fields[0], "0") &&
			strings.Contains(fields[1], "BIRD") {
			version = "bird-" + fields[2]
		}
		if len(fields) == 6 && strings.HasPrefix(text, " ") {
			ll, err := netaddr.ParseIP(fields[0])
			if err != nil || !ll.Is6() || !ll.IsLinkLocalUnicast() {
				continue
			}
			neighs = append(neighs, alfredxml.BabelNeighbour{
				IP:                fields[0],
				OutgoingInterface: fields[1],
				LinkCost:          fields[2],
			})
		}
	}

	if scanner.Err() != nil {
		return version, nil
	}

	return version, neighs
}

func netInterfaceFromLink(link netlink.Link) net.Interface {
	attrs := link.Attrs()
	return net.Interface{
		Index:        attrs.Index,
		MTU:          attrs.MTU,
		Name:         attrs.Name,
		HardwareAddr: attrs.HardwareAddr,
		Flags:        attrs.Flags,
	}
}

func crawl(c Config) (d alfredxml.Data, err error) {
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
		d.SystemData.MemoryAvailable = mem.MemAvailable
		d.SystemData.Processes = fmt.Sprintf("%d/%d", load.runnable, load.procs)
		d.SystemData.Uptime = float64(sysinfo.Uptime)
	}

	{
		var utsname unix.Utsname
		if err = unix.Uname(&utsname); err != nil {
			return
		}
		d.SystemData.KernelVersion = string(bytes.Trim(utsname.Release[:], "\x00"))
	}

	links, err := netlink.LinkList()
	if err != nil {
		return
	}

	// sort links by name and make sure "client" interface is sorted first
	sort.Slice(links, func(i, j int) bool {
		iname := links[i].Attrs().Name
		jname := links[j].Attrs().Name
		if iname == c.ClientIfName {
			return true
		}
		if jname == c.ClientIfName {
			return false
		}
		return iname < jname
	})

	// rename the client interface if requested
	if len(links) > 0 && c.RenameClientIf && links[0].Attrs().Name == c.ClientIfName {
		links[0].Attrs().Name = "br-client"
	}

	for _, link := range links {
		// skip lo
		attrs := link.Attrs()
		if attrs.Name == "lo" {
			continue
		}

		if attrs.Flags&net.FlagUp == 0 {
			continue
		}

		d.InterfaceData.Interfaces = append(d.InterfaceData.Interfaces, alfredxml.Interface{
			XMLName: xml.Name{
				Local: attrs.Name,
			},
			Name:      attrs.Name,
			Mtu:       attrs.MTU,
			MacAddr:   attrs.HardwareAddr.String(),
			TrafficRx: attrs.Statistics.RxBytes,
			TrafficTx: attrs.Statistics.TxBytes,
		})

		// only run neighbour discovery on layer2 devices
		if len(bytes.Trim(attrs.HardwareAddr, "\x00")) == 0 {
			continue
		}

		neighs, err := netlink.NeighList(attrs.Index, netlink.FAMILY_ALL)
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
		_, err = nc.solicit(2*time.Second, netInterfaceFromLink(link), neighProbe...)
		if err != nil {
			return d, err
		}

		neighAddrs := map[string]struct{}{}
		neighs, err = netlink.NeighList(attrs.Index, netlink.FAMILY_ALL)
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
		d.Clients.Num = append(d.Clients.Num, alfredxml.ClientNum{
			XMLName: xml.Name{
				Local: attrs.Name,
			},
			N: count,
		})
	}

	babeldversion, babeldneighbours := getBabeldInfo()
	birdversion, birdneighbours := getBirdInfo()
	d.BabelNeighbours.Neighbours = append(babeldneighbours, birdneighbours...)

	d.SystemData.BabelVersion = babeldversion
	if babeldversion != "" && birdversion != "" {
		d.SystemData.BabelVersion += ", "
	}
	d.SystemData.BabelVersion += birdversion

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

func sendReport(c Config, payload []byte) error {
	req, err := http.NewRequest("POST", "https://monitoring.freifunk-franken.de/api/alfred2", bytes.NewReader(payload))
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

	payload, err := json.Marshal(alfredxml.Alfred2Slice{d})
	if err != nil {
		return nil, err
	}

	if c.Debug {
		c.Log.Println()
		c.Log.Println("Alfred2 Payload:")
		c.Log.Println()
		c.Log.Println(string(payload))
	}

	return payload, nil
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

			c.Log.Println(err)

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
