package docker

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	dockerapi "github.com/fsouza/go-dockerclient"
	"github.com/jwilder/hud/metrics"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/docker"
	"github.com/shirou/gopsutil/mem"
)

const interval = 1 * time.Second

type DockerCollector struct {
	metrics.Collector
	sync.Mutex
	client      *dockerapi.Client
	Broadcaster *Broadcaster
	tailer      *Tailer
	interval    int
}

func NewDockerCollector(prefix string, broadcaster *Broadcaster, interval int) *DockerCollector {

	collector := &DockerCollector{
		Broadcaster: broadcaster,
		interval:    interval,
	}
	collector.Prefix = prefix
	broadcaster.AddEventHandler(collector.onDockerEvent)

	tailer := &Tailer{
		Broadcaster: broadcaster,
	}
	collector.tailer = tailer
	go tailer.Tail()
	tailer.AddLogHandler(collector)

	return collector
}

func (d *DockerCollector) AddLogHandler(handler LogHandler) {
	d.tailer.AddLogHandler(handler)
}

func (d *DockerCollector) getDockerClient() (*dockerapi.Client, error) {
	d.Lock()
	defer d.Unlock()
	if d.client == nil {
		var err error
		endpoint := d.Broadcaster.Endpoint
		client, err := NewDockerClient(endpoint)
		if err != nil {
			return nil, fmt.Errorf("unable to connect to docker daemon: %s", err)
		}
		d.client = client
	}
	return d.client, nil
}

func (d *DockerCollector) CollectForever() {

	go d.collectDockerImages()

	var wg sync.WaitGroup
	for {

		client, err := d.getDockerClient()
		if err != nil {
			log.Errorf("Unable to connect to docker daemon: %s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		apiContainers, err := client.ListContainers(dockerapi.ListContainersOptions{
			All:  false,
			Size: false,
		})
		if err != nil {
			log.Errorf("Unable to list containers: %s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		d.RecordGauge("docker.containers", int64(len(apiContainers)))

		wg.Add(2)

		go func() {
			defer wg.Done()
			err := d.collectDockerCpu(apiContainers)
			if err != nil {
				log.Errorf("ERROR: Unable to collect docker cpu stats: %s", err)
			}

		}()

		go func() {
			defer wg.Done()
			err := d.collectDockerMemory(apiContainers)
			if err != nil {
				log.Errorf("ERROR: Unable to collect docker mem stats: %s", err)
			}
		}()
		wg.Wait()
		time.Sleep(time.Duration(d.interval) * time.Second)
	}
}

func (d *DockerCollector) collectDockerCpu(containers []dockerapi.APIContainers) error {
	startTimes := map[string]*cpu.CPUTimesStat{}
	stopTimes := map[string]*cpu.CPUTimesStat{}
	hostStart, err := cpu.CPUTimes(false)
	if err != nil {
		return err
	}

	for _, container := range containers {
		id := container.ID
		start, err := docker.CgroupCPUDocker(id)
		if err != nil {
			return err
		}
		startTimes[id] = start
	}

	time.Sleep(1 * time.Second)
	hostStop, err := cpu.CPUTimes(false)
	if err != nil {
		return err
	}

	for _, container := range containers {
		id := container.ID
		stop, err := docker.CgroupCPUDocker(id)
		if err != nil {
			return err
		}
		stopTimes[id] = stop
	}

	numCpusInt, _ := cpu.CPUCounts(true)
	numCpus := float64(numCpusInt)

	for _, container := range containers {
		id := container.ID

		start := startTimes[id]
		stop := stopTimes[id]

		user := float64(stop.User-start.User) / numCpus
		system := float64(stop.System-start.System) / numCpus

		hostUser := float64(hostStop[0].User-hostStart[0].User) / numCpus
		hostSystem := float64(hostStop[0].System-hostStart[0].System) / numCpus
		hostIowait := float64(hostStop[0].Iowait-hostStart[0].Iowait) / numCpus
		hostIrq := float64(hostStop[0].Irq-hostStart[0].Irq) / numCpus
		hostSoftirq := float64(hostStop[0].Softirq-hostStart[0].Softirq) / numCpus
		hostIdle := float64(hostStop[0].Idle-hostStart[0].Idle) / numCpus
		hostNice := float64(hostStop[0].Nice-hostStart[0].Nice) / numCpus
		hostGuest := float64(hostStop[0].Guest-hostStart[0].Guest) / numCpus
		hostGuestNice := float64(hostStop[0].GuestNice-hostStart[0].GuestNice) / numCpus
		hostStolen := float64(hostStop[0].Stolen-hostStart[0].Stolen) / numCpus

		period := float64(hostUser + hostSystem + hostIowait + hostIrq + hostSoftirq +
			hostIdle + hostNice + hostGuest + hostGuestNice + hostStolen)

		//period = float64(user + system)

		total := float64(0)

		userPerc := float64(0)
		sysPerc := float64(0)

		if user+system > 0 {
			userPerc = user / (period) * numCpus * 100
			sysPerc = system / (period) * numCpus * 100
		}

		if period > 0 {
			total = (user + system) / period * numCpus * 100
		}

		name := d.safeName(container.Names[0][1:])

		d.RecordGaugeFloat64(fmt.Sprintf("docker.cpu.total.%s", name), total)
		d.RecordGaugeFloat64(fmt.Sprintf("docker.cpu.user.%s", name), userPerc)
		d.RecordGaugeFloat64(fmt.Sprintf("docker.cpu.system.%s", name), sysPerc)
	}
	return nil

}

func (d *DockerCollector) collectDockerMemory(containers []dockerapi.APIContainers) error {
	hostVM, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	for _, container := range containers {
		id := container.ID
		cMem, err := docker.CgroupMemDocker(id)
		if err != nil {
			return err
		}

		total := cMem.HierarchicalMemoryLimit
		if total == math.MaxUint64 {
			total = hostVM.Total
		}

		name := d.safeName(container.Names[0][1:])
		d.RecordGauge(fmt.Sprintf("docker.mem.total.%s", name),
			int64(cMem.Cache+cMem.RSS))

		d.RecordGauge(fmt.Sprintf("docker.mem.cache.%s", name),
			int64(cMem.Cache))

		d.RecordGauge(fmt.Sprintf("docker.mem.rss.%s", name),
			int64(cMem.RSS))
	}
	return nil
}

func (d *DockerCollector) collectDockerImages() error {
	for {

		client, err := d.getDockerClient()
		if err != nil {
			log.Printf("ERROR: Unable to collect docker image stats: %s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		images, err := client.ListImages(dockerapi.ListImagesOptions{
			All: false,
		})
		if err != nil {
			return err
		}
		d.RecordGauge("docker.images", int64(len(images)))

		layers, err := client.ListImages(dockerapi.ListImagesOptions{
			All: true,
		})
		if err != nil {
			return err
		}

		d.RecordGauge("docker.layers", int64(len(layers)))

		time.Sleep(60 * time.Second)
	}
}

func (d *DockerCollector) onDockerEvent(client *dockerapi.Client, event *dockerapi.APIEvents) {
	d.RecordCount(fmt.Sprintf("docker.events.%s", event.Status), 1)
}

func (d *DockerCollector) HandleLog(log *LogRecord) error {
	d.RecordCount(fmt.Sprintf("docker.logs.total.%s", d.safeName(log.ContainerName)), 1)
	d.RecordCount(fmt.Sprintf("docker.logs.%s.%s", log.Stream, d.safeName(log.ContainerName)), 1)
	return nil
}

func (d *DockerCollector) safeName(name string) string {
	return strings.Replace(name, ".", "_", -1)
}
