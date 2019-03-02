package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type meminfo struct {
	MemTotal          int
	MemFree           int
	MemAvailable      int
	Buffers           int
	Cached            int
	SwapCached        int
	Active            int
	Inactive          int
	ActiveAnon        int
	InactiveAnon      int
	ActiveFile        int
	InactiveFile      int
	Unevictable       int
	Mlocked           int
	SwapTotal         int
	SwapFree          int
	Dirty             int
	Writeback         int
	AnonPages         int
	Mapped            int
	Shmem             int
	KReclaimable      int
	Slab              int
	SReclaimable      int
	SUnreclaim        int
	KernelStack       int
	PageTables        int
	NFS_Unstable      int
	Bounce            int
	WritebackTmp      int
	CommitLimit       int
	Committed_AS      int
	VmallocTotal      int
	VmallocUsed       int
	VmallocChunk      int
	Percpu            int
	HardwareCorrupted int
	AnonHugePages     int
	ShmemHugePages    int
	ShmemPmdMapped    int
	HugePages_Total   int
	HugePages_Free    int
	HugePages_Rsvd    int
	HugePages_Surp    int
	Hugepagesize      int
	Hugetlb           int
	DirectMap4k       int
	DirectMap2M       int
	DirectMap1G       int
}

func readMeminfo() (meminfo, error) {
	var m meminfo

	f_meminfo, err := os.Open("/proc/meminfo")
	if err != nil {
		return m, err
	}
	defer f_meminfo.Close()

	scan := bufio.NewScanner(f_meminfo)
	for scan.Scan() {
		if _, err := m.setLine(scan.Text()); err != nil {
			return m, err
		}
	}

	return m, scan.Err()

}

type meminfoError string

func (m meminfoError) Error() string {
	return fmt.Sprintf("Unkown meminfo field %q", m)
}
func (m *meminfo) setLine(l string) (*meminfo, error) {
	var field string
	var value int
	n, err := fmt.Sscanf(l, "%s %d", &field, &value)
	if n != 2 || err != nil {
		return m, err
	}
	return m.set(strings.TrimRight(field, ":"), value)
}
func (m *meminfo) set(field string, value int) (*meminfo, error) {
	switch field {
	case "MemTotal":
		m.MemTotal = value
	case "MemFree":
		m.MemFree = value
	case "MemAvailable":
		m.MemAvailable = value
	case "Buffers":
		m.Buffers = value
	case "Cached":
		m.Cached = value
	case "SwapCached":
		m.SwapCached = value
	case "Active":
		m.Active = value
	case "Inactive":
		m.Inactive = value
	case "Active(anon)":
		m.ActiveAnon = value
	case "Inactive(anon)":
		m.InactiveAnon = value
	case "Active(file)":
		m.ActiveFile = value
	case "Inactive(file)":
		m.InactiveFile = value
	case "Unevictable":
		m.Unevictable = value
	case "Mlocked":
		m.Mlocked = value
	case "SwapTotal":
		m.SwapTotal = value
	case "SwapFree":
		m.SwapFree = value
	case "Dirty":
		m.Dirty = value
	case "Writeback":
		m.Writeback = value
	case "AnonPages":
		m.AnonPages = value
	case "Mapped":
		m.Mapped = value
	case "Shmem":
		m.Shmem = value
	case "KReclaimable":
		m.KReclaimable = value
	case "Slab":
		m.Slab = value
	case "SReclaimable":
		m.SReclaimable = value
	case "SUnreclaim":
		m.SUnreclaim = value
	case "KernelStack":
		m.KernelStack = value
	case "PageTables":
		m.PageTables = value
	case "NFS_Unstable":
		m.NFS_Unstable = value
	case "Bounce":
		m.Bounce = value
	case "WritebackTmp":
		m.WritebackTmp = value
	case "CommitLimit":
		m.CommitLimit = value
	case "Committed_AS":
		m.Committed_AS = value
	case "VmallocTotal":
		m.VmallocTotal = value
	case "VmallocUsed":
		m.VmallocUsed = value
	case "VmallocChunk":
		m.VmallocChunk = value
	case "Percpu":
		m.Percpu = value
	case "HardwareCorrupted":
		m.HardwareCorrupted = value
	case "AnonHugePages":
		m.AnonHugePages = value
	case "ShmemHugePages":
		m.ShmemHugePages = value
	case "ShmemPmdMapped":
		m.ShmemPmdMapped = value
	case "HugePages_Total":
		m.HugePages_Total = value
	case "HugePages_Free":
		m.HugePages_Free = value
	case "HugePages_Rsvd":
		m.HugePages_Rsvd = value
	case "HugePages_Surp":
		m.HugePages_Surp = value
	case "Hugepagesize":
		m.Hugepagesize = value
	case "Hugetlb":
		m.Hugetlb = value
	case "DirectMap4k":
		m.DirectMap4k = value
	case "DirectMap2M":
		m.DirectMap2M = value
	case "DirectMap1G":
		m.DirectMap1G = value
	default:
		return m, meminfoError(field)
	}
	return m, nil
}
