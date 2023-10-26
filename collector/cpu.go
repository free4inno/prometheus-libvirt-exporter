package collector

import (
	"strconv"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

type cpuCollector struct {
	secondsTotal typedDesc
	vCPUNumber   typedDesc
	logger       log.Logger
}

func init() {
	registerCollector("cpu", defaultEnabled, NewCPUCollector)
}

func NewCPUCollector(logger log.Logger) (Collector, error) {
	return &cpuCollector{
		secondsTotal: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "domain_cpu", "seconds_total"),
				"Seconds the vCPUs in VMs for each domain",
				[]string{"domain_uuid", "state"},
				nil),
			prometheus.CounterValue,
		},
		vCPUNumber: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "domain_cpu", "vcpu_number"),
				"Number of vCPUs in VMs for each domain",
				[]string{"domain_uuid", "state"},
				nil),
			prometheus.GaugeValue,
		},
		logger: logger,
	}, nil
}

func (c *cpuCollector) Update(ch chan<- prometheus.Metric, opts ...CollectorOption) error {
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

	for _, lvDomain := range lvDomains {
		// state meaning explained here: https://libvirt.org/html/libvirt-libvirt-domain.html#virDomainState
		state, _, _, nrVirtCPU, cpuTime, err := pLibvirt.DomainGetInfo(lvDomain.domain)
		if err != nil {
			level.Error(c.logger).Log("msg", "failed to get domain info", "domain", lvDomain.domain.Name, "err", err)
			continue
		}
		level.Debug(c.logger).Log("msg", "get domain info", "domain", lvDomain.domain.Name, "nrVirtCPU", nrVirtCPU, "cpuTime", cpuTime)

		ch <- c.secondsTotal.mustNewConstMetric(float64(cpuTime)/1e9, lvDomain.schema.UUID, strconv.Itoa(int(state)))
		ch <- c.vCPUNumber.mustNewConstMetric(float64(nrVirtCPU), lvDomain.schema.UUID, strconv.Itoa(int(state)))
	}

	return nil
}
