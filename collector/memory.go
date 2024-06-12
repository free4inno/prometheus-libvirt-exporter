package collector

import (
	"sync"

	libvirt "github.com/digitalocean/go-libvirt"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

type memoryCollector struct {
	swapInBytes         typedDesc
	swapOutBytes        typedDesc
	majorPageFaults     typedDesc
	minorPageFaults     typedDesc
	unusedBytes         typedDesc
	availableBytes      typedDesc
	actualBallonBytes   typedDesc
	rssBytes            typedDesc
	usableBytes         typedDesc
	lastUpdateTimestamp typedDesc
	diskCacheBytes      typedDesc
	hugetlbPagesAlloc   typedDesc
	hugetlbPageFaults   typedDesc
	logger              log.Logger
}

const memorySubsystemName = "domain_memory_stat"

func init() {
	registerCollector("memory", defaultEnabled, NewMemoryCollector)
}

// NewMemoryCollector returns a new Collector exposing memory stats.
func NewMemoryCollector(logger log.Logger) (Collector, error) {
	return &memoryCollector{
		swapInBytes: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memorySubsystemName, "swap_in_bytes"),
				"Total amount of data read from swap space (in bytes)",
				[]string{"domain_uuid"},
				nil),
			valueType: prometheus.GaugeValue,
		},
		swapOutBytes: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memorySubsystemName, "swap_out_bytes"),
				"Total amount of memory written out to swap space (in bytes)",
				[]string{"domain_uuid"},
				nil),
			valueType: prometheus.GaugeValue,
		},
		majorPageFaults: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memorySubsystemName, "major_page_faults_number"),
				"Number of major page faults",
				[]string{"domain_uuid"},
				nil),
			valueType: prometheus.GaugeValue,
		},
		minorPageFaults: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memorySubsystemName, "minor_page_faults_number"),
				"Number of minor page faults",
				[]string{"domain_uuid"},
				nil),
			valueType: prometheus.GaugeValue,
		},
		unusedBytes: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memorySubsystemName, "unused_bytes"),
				"Amount of memory left completely unused by the system (in bytes)",
				[]string{"domain_uuid"},
				nil),
			valueType: prometheus.GaugeValue,
		},
		availableBytes: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memorySubsystemName, "available_bytes"),
				"Total amount of usable memory (in bytes)",
				[]string{"domain_uuid"},
				nil),
			valueType: prometheus.GaugeValue,
		},
		actualBallonBytes: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memorySubsystemName, "actual_ballon_bytes"),
				"Current balloon value (in bytes)",
				[]string{"domain_uuid"},
				nil),
			valueType: prometheus.GaugeValue,
		},
		rssBytes: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memorySubsystemName, "rss_bytes"),
				"Resident Set Size of the process running the domain (in bytes)",
				[]string{"domain_uuid"},
				nil),
			valueType: prometheus.GaugeValue,
		},
		usableBytes: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memorySubsystemName, "usable_bytes"),
				"Amount of memory reclaimable by the memory reclamation subsystem (in bytes)",
				[]string{"domain_uuid"},
				nil),
			valueType: prometheus.GaugeValue,
		},
		lastUpdateTimestamp: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memorySubsystemName, "last_update_timestamp_seconds"),
				"Timestamp of the last update of statistics, in seconds",
				[]string{"domain_uuid"},
				nil),
			valueType: prometheus.GaugeValue,
		},
		diskCacheBytes: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memorySubsystemName, "disk_cache_bytes"),
				"Amount of memory used as disk cache (in bytes)",
				[]string{"domain_uuid"},
				nil),
			valueType: prometheus.GaugeValue,
		},
		hugetlbPagesAlloc: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memorySubsystemName, "hugetlb_pages_alloc_number"),
				"Number of hugepages allocated",
				[]string{"domain_uuid"},
				nil),
			valueType: prometheus.GaugeValue,
		},
		hugetlbPageFaults: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memorySubsystemName, "hugetlb_page_faults_number"),
				"Number of hugepages page faults",
				[]string{"domain_uuid"},
				nil),
			valueType: prometheus.GaugeValue,
		},

		logger: logger,
	}, nil
}

func (c *memoryCollector) Update(ch chan<- prometheus.Metric, opts ...CollectorOption) error {
	config := &CollectorConfig{}
	for _, opt := range opts {
		opt(config)
	}
	if config.pLibvirt == nil {
		level.Error(c.logger).Log("msg", "libvirt not created")
		return ErrNotProvided
	}
	if !config.pLibvirt.IsConnected() {
		level.Error(c.logger).Log("msg", "libvirt not connected")
		return ErrNotProvided
	}
	if config.lvDomains == nil || len(config.lvDomains) == 0 {
		level.Error(c.logger).Log("msg", "no domains found")
		return ErrNotProvided
	}
	pLibvirt := config.pLibvirt
	lvDomains := config.lvDomains

	wg := sync.WaitGroup{}
	wg.Add(len(lvDomains))

	for _, lvDomain := range lvDomains {
		domainUUID := lvDomain.Schema.UUID
		go func(domain libvirt.Domain, domainUUID string) {
			stats, err := pLibvirt.DomainMemoryStats(domain, uint32(libvirt.DomainMemoryStatNr), 0)
			if err != nil {
				level.Error(c.logger).Log("msg", "failed to get memory stats", "domain", domain.Name, "err", err)
				wg.Done()
				return
			}

			for _, stat := range stats {
				tag := libvirt.DomainMemoryStatTags(stat.Tag)
				switch tag {
				case libvirt.DomainMemoryStatSwapIn:
					ch <- c.swapInBytes.mustNewConstMetric(float64(stat.Val), domainUUID)
				case libvirt.DomainMemoryStatSwapOut:
					ch <- c.swapOutBytes.mustNewConstMetric(float64(stat.Val), domainUUID)
				case libvirt.DomainMemoryStatMajorFault:
					ch <- c.majorPageFaults.mustNewConstMetric(float64(stat.Val), domainUUID)
				case libvirt.DomainMemoryStatMinorFault:
					ch <- c.minorPageFaults.mustNewConstMetric(float64(stat.Val), domainUUID)
				case libvirt.DomainMemoryStatUnused:
					ch <- c.unusedBytes.mustNewConstMetric(float64(stat.Val), domainUUID)
				case libvirt.DomainMemoryStatAvailable:
					ch <- c.availableBytes.mustNewConstMetric(float64(stat.Val), domainUUID)
				case libvirt.DomainMemoryStatActualBalloon:
					ch <- c.actualBallonBytes.mustNewConstMetric(float64(stat.Val), domainUUID)
				case libvirt.DomainMemoryStatRss:
					ch <- c.rssBytes.mustNewConstMetric(float64(stat.Val), domainUUID)
				case libvirt.DomainMemoryStatUsable:
					ch <- c.usableBytes.mustNewConstMetric(float64(stat.Val), domainUUID)
				case libvirt.DomainMemoryStatDiskCaches:
					ch <- c.diskCacheBytes.mustNewConstMetric(float64(stat.Val), domainUUID)
				case libvirt.DomainMemoryStatLastUpdate:
					ch <- c.lastUpdateTimestamp.mustNewConstMetric(float64(stat.Val), domainUUID)
				case libvirt.DomainMemoryStatHugetlbPgalloc:
					ch <- c.hugetlbPagesAlloc.mustNewConstMetric(float64(stat.Val), domainUUID)
				case libvirt.DomainMemoryStatHugetlbPgfail:
					ch <- c.hugetlbPageFaults.mustNewConstMetric(float64(stat.Val), domainUUID)
				default:
					level.Error(c.logger).Log("msg", "unknown memory stat", "domain", domain.Name, "tag", stat.Tag)
				}
			}
			wg.Done()
		}(lvDomain.Domain, domainUUID)
	}
	wg.Wait()
	return nil
}
