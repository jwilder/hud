package host

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jwilder/hud/metrics"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

type HostCollector struct {
	metrics.Collector
	interval int
}

func NewHostCollector(prefix string, interval int) *HostCollector {

	collector := &HostCollector{}
	collector.Prefix = prefix
	collector.interval = interval
	return collector
}

func (h *HostCollector) CollectForever() {

	var wg sync.WaitGroup
	for {
		wg.Add(5)
		go func() {
			defer wg.Done()
			err := h.collectCpuUtil()
			if err != nil {
				log.Errorf("ERROR: %s", err)
			}
		}()

		go func() {
			defer wg.Done()
			err := h.collectLoadAvg()
			if err != nil {
				log.Errorf("ERROR: %s", err)
			}
		}()

		go func() {
			defer wg.Done()
			err := h.collectMemory()
			if err != nil {
				log.Errorf("ERROR: %s", err)
			}
		}()

		go func() {
			defer wg.Done()
			err := h.collectNet()
			if err != nil {
				log.Errorf("ERROR: %s", err)
			}
		}()

		go func() {
			defer wg.Done()
			err := h.collectDisk()
			if err != nil {
				log.Errorf("ERROR: %s", err)
			}
		}()

		wg.Wait()
	}
}

func (h *HostCollector) collectLoadAvg() error {
	load, err := load.LoadAvg()
	if err != nil {
		return err
	}
	h.RecordGaugeFloat64("system.load.load1", load.Load1)
	h.RecordGaugeFloat64("system.load.load5", load.Load5)
	h.RecordGaugeFloat64("system.load.load15", load.Load15)
	return nil
}

func (h *HostCollector) collectCpuUtil() error {
	startTimes, err := cpu.CPUTimes(true)
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	stopTimes, err := cpu.CPUTimes(true)
	if err != nil {
		return err
	}

	numCpusInt, _ := cpu.CPUCounts(true)
	numCpus := float64(numCpusInt)

	for i := 0; i < len(startTimes); i++ {
		start := startTimes[i]
		stop := stopTimes[i]

		idle := float64(stop.Idle-start.Idle) / numCpus
		iowait := float64(stop.Iowait-start.Iowait) / numCpus
		idleAll := float64(idle + iowait)

		guest := float64(stop.Guest-start.Guest) / numCpus
		guestNice := float64(stop.GuestNice-start.GuestNice) / numCpus
		nice := float64(stop.Nice-start.Nice) / numCpus
		user := float64(stop.User-start.User) / numCpus
		userAll := float64(user - guest)
		niceAll := float64(nice - guestNice)
		steal := float64(stop.Steal-start.Steal) / numCpus

		system := float64(stop.System-start.System) / numCpus
		irq := float64(stop.Irq-start.Irq) / numCpus
		softIrq := float64(stop.Softirq-start.Softirq) / numCpus
		systemAll := float64(system + irq + softIrq)
		//virtAll := float64(guest + guestNice)
		stolen := float64(stop.Stolen-start.Stolen) / numCpus

		busy := float64(user + system + nice + iowait + irq + softIrq + steal + guest + guestNice + stolen)

		total := busy + idle

		period := float64(total)

		cpu := start.CPU
		if start.CPU == "cpu-total" {
			cpu = "all"
		}

		h.RecordGaugeFloat64(fmt.Sprintf("system.cpu.util.total.%s", cpu),
			busy/period*100)
		h.RecordGaugeFloat64(fmt.Sprintf("system.cpu.util.user.%s", cpu),
			userAll/period*100)
		h.RecordGaugeFloat64(fmt.Sprintf("system.cpu.util.system.%s", cpu),
			systemAll/period*100)
		h.RecordGaugeFloat64(fmt.Sprintf("system.cpu.util.iowait.%s", cpu),
			iowait/period*100)
		h.RecordGaugeFloat64(fmt.Sprintf("system.cpu.util.idle.%s", cpu),
			idleAll/period*100)
		h.RecordGaugeFloat64(fmt.Sprintf("system.cpu.util.nice.%s", cpu),
			niceAll/period*100)
		h.RecordGaugeFloat64(fmt.Sprintf("system.cpu.util.irq.%s", cpu),
			irq/period*100)
		h.RecordGaugeFloat64(fmt.Sprintf("system.cpu.util.softirq.%s", cpu),
			softIrq/period*100)
		h.RecordGaugeFloat64(fmt.Sprintf("system.cpu.util.steal.%s", cpu),
			stolen/period*100)
	}
	return nil
}

func (h *HostCollector) collectMemory() error {
	mem, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	h.RecordGauge("system.mem.total", int64(mem.Total))
	h.RecordGauge("system.mem.free", int64(mem.Free))
	h.RecordGauge("system.mem.buffers", int64(mem.Buffers))
	h.RecordGauge("system.mem.cached", int64(mem.Cached))
	h.RecordGauge("system.mem.active", int64(mem.Active))
	h.RecordGauge("system.mem.inactive", int64(mem.Inactive))
	h.RecordGauge("system.mem.available", int64(mem.Available))
	h.RecordGauge("system.mem.shared", int64(mem.Shared))
	return nil
}

func (h *HostCollector) collectNet() error {
	netStart, err := net.NetIOCounters(true)
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	netStop, err := net.NetIOCounters(true)
	if err != nil {
		return err
	}

	ns, err := net.NetInterfaces()
	if err != nil {
		return err
	}

	ifByName := map[string]net.NetInterfaceStat{}
	for _, n := range ns {
		ifByName[n.Name] = n
	}

	cnt := math.Min(float64(len(netStart)), float64(len(netStop)))
	for i := 0; i < int(cnt); i++ {

		start := netStart[i]
		stop := netStop[i]

		iface := ifByName[start.Name]

		intv64 := int64(h.interval)
		bytesSent := int64(stop.BytesSent-start.BytesSent) / intv64
		bytesRecv := int64(stop.BytesRecv-start.BytesRecv) / intv64
		bytesTotal := (bytesRecv + bytesSent)

		h.RecordCount(fmt.Sprintf("system.net.bytes.sent.if.%s", start.Name), bytesSent)
		h.RecordCount(fmt.Sprintf("system.net.bytes.recv.if.%s", start.Name), bytesRecv)
		h.RecordCount(fmt.Sprintf("system.net.bytes.total.if.%s", start.Name), bytesTotal)

		packetsSent := int64(stop.PacketsSent-start.PacketsSent) / intv64
		packetsRecv := int64(stop.PacketsRecv-start.PacketsRecv) / intv64
		packetsTotal := packetsSent + packetsRecv

		h.RecordCount(fmt.Sprintf("system.net.packets.sent.if.%s", start.Name), packetsSent)
		h.RecordCount(fmt.Sprintf("system.net.packets.recv.if.%s", start.Name), packetsRecv)
		h.RecordCount(fmt.Sprintf("system.net.packets.total.if.%s", start.Name), packetsTotal)

		errIn := int64(stop.Errin-start.Errin) / intv64
		errOut := int64(stop.Errout-start.Errout) / intv64
		errTotal := errIn + errOut

		h.RecordCount(fmt.Sprintf("system.net.errors.in.if.%s", start.Name), errIn)
		h.RecordCount(fmt.Sprintf("system.net.errors.out.if.%s", start.Name), errOut)
		h.RecordCount(fmt.Sprintf("system.net.errors.total.if.%s", start.Name), errTotal)

		droppedIn := int64(stop.Dropin-start.Dropin) / intv64
		droppedOut := int64(stop.Dropout-start.Dropin) / intv64
		droppedTotal := droppedIn + droppedOut

		h.RecordCount(fmt.Sprintf("system.net.dropped.in.if.%s", start.Name), droppedIn)
		h.RecordCount(fmt.Sprintf("system.net.dropped.out.if.%s", start.Name), droppedOut)
		h.RecordCount(fmt.Sprintf("system.net.dropped.total.if.%s", start.Name), droppedTotal)

		for _, addr := range iface.Addrs {
			version := "ip4"
			if strings.Contains(addr.Addr, ":") {
				version = "ip6"
			}
			ip := strings.Replace(addr.Addr, ".", "_", -1)
			subnet := strings.LastIndex(ip, "/")
			if subnet != -1 {
				ip = ip[0:subnet]
			}
			h.RecordCount(fmt.Sprintf("system.net.bytes.sent.%s.%s", version, ip), bytesSent)
			h.RecordCount(fmt.Sprintf("system.net.bytes.recv.%s.%s", version, ip), bytesRecv)
			h.RecordCount(fmt.Sprintf("system.net.bytes.total.%s.%s", version, ip), bytesTotal)

			h.RecordCount(fmt.Sprintf("system.net.packets.sent.%s.%s", version, ip), packetsSent)
			h.RecordCount(fmt.Sprintf("system.net.packets.recv.%s.%s", version, ip), packetsRecv)
			h.RecordCount(fmt.Sprintf("system.net.packets.total.%s.%s", version, ip), packetsTotal)

			h.RecordCount(fmt.Sprintf("system.net.errors.in.%s.%s", version, ip), errIn)
			h.RecordCount(fmt.Sprintf("system.net.errors.out.%s.%s", version, ip), errOut)
			h.RecordCount(fmt.Sprintf("system.net.errors.total.%s.%s", version, ip), errTotal)

			h.RecordCount(fmt.Sprintf("system.net.dropped.in.%s.%s", version, ip), droppedIn)
			h.RecordCount(fmt.Sprintf("system.net.dropped.out.%s.%s", version, ip), droppedOut)
			h.RecordCount(fmt.Sprintf("system.net.dropped.total.%s.%s", version, ip), droppedTotal)
		}
	}

	return nil
}

func (h *HostCollector) collectDisk() error {

	partitions, err := disk.DiskPartitions(true)
	if err != nil {
		return err
	}

	partByDevice := map[string]disk.DiskPartitionStat{}
	for _, p := range partitions {
		partByDevice[p.Device] = p
	}

	start, err := disk.DiskIOCounters()
	if err != nil {
		return err
	}
	time.Sleep(1 * time.Second)

	stop, err := disk.DiskIOCounters()
	if err != nil {
		return err
	}

	for disk, stats := range start {
		device := fmt.Sprintf("/dev/%s", disk)
		mp := partByDevice[device].Mountpoint

		read := int64(stop[disk].ReadBytes - stats.ReadBytes)
		write := int64(stop[disk].WriteBytes - stats.WriteBytes)
		total := read + write

		if mp != "" {
			h.RecordCount(fmt.Sprintf("system.disk.bytes.read.mount.%s.", mp), read)
			h.RecordCount(fmt.Sprintf("system.disk.bytes.write.mount.%s", mp), write)
			h.RecordCount(fmt.Sprintf("system.disk.bytes.total.mount.%s", mp), total)
		}

		h.RecordCount(fmt.Sprintf("system.disk.bytes.read.dev.%s.", disk), read)
		h.RecordCount(fmt.Sprintf("system.disk.bytes.write.dev.%s", disk), write)
		h.RecordCount(fmt.Sprintf("system.disk.bytes.total.dev.%s", disk), total)

		read = int64(stop[disk].ReadCount - stats.ReadCount)
		write = int64(stop[disk].WriteCount - stats.WriteCount)
		total = read + write

		if mp != "" {
			h.RecordCount(fmt.Sprintf("system.disk.iops.read.mount.%s", mp), read)
			h.RecordCount(fmt.Sprintf("system.disk.iops.write.mount.%s", mp), write)
			h.RecordCount(fmt.Sprintf("system.disk.iops.total.mount.%s", mp), total)
		}

		h.RecordCount(fmt.Sprintf("system.disk.iops.read.dev.%s", disk), read)
		h.RecordCount(fmt.Sprintf("system.disk.iops.write.dev.%s", disk), write)
		h.RecordCount(fmt.Sprintf("system.disk.iops.total.dev.%s", disk), total)

	}

	return nil
}
