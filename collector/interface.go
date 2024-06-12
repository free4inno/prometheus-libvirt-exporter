package collector

import (
	"sync"

	libvirt "github.com/digitalocean/go-libvirt"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

const interfaceSubsystemName = "domain_interface"

type interfaceCollector struct {
	receiveBytesTotal    typedDesc
	receivePacketsTotal  typedDesc
	receiveErrorsTotal   typedDesc
	receiveDropsTotal    typedDesc
	transmitBytesTotal   typedDesc
	transmitPacketsTotal typedDesc
	transmitErrorsTotal  typedDesc
	transmitDropsTotal   typedDesc
	logger               log.Logger
}

func init() {
	registerCollector("interface", defaultEnabled, NewInterfaceCollector)
}

func NewInterfaceCollector(logger log.Logger) (Collector, error) {
	return &interfaceCollector{
		receiveBytesTotal: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, interfaceSubsystemName, "receive_bytes_total"),
				"Total number of bytes received",
				[]string{"domain_uuid", "bridge", "interface"},
				nil),
			valueType: prometheus.CounterValue,
		},
		receivePacketsTotal: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, interfaceSubsystemName, "receive_packets_total"),
				"Total number of packets received",
				[]string{"domain_uuid", "bridge", "interface"},
				nil),
			valueType: prometheus.CounterValue,
		},
		receiveErrorsTotal: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, interfaceSubsystemName, "receive_errors_total"),
				"Total number of receive errors",
				[]string{"domain_uuid", "bridge", "interface"},
				nil),
			valueType: prometheus.CounterValue,
		},
		receiveDropsTotal: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, interfaceSubsystemName, "receive_drops_total"),
				"Total number of receive drops",
				[]string{"domain_uuid", "bridge", "interface"},
				nil),
			valueType: prometheus.CounterValue,
		},
		transmitBytesTotal: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, interfaceSubsystemName, "transmit_bytes_total"),
				"Total number of bytes transmitted",
				[]string{"domain_uuid", "bridge", "interface"},
				nil),
			valueType: prometheus.CounterValue,
		},
		transmitPacketsTotal: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, interfaceSubsystemName, "transmit_packets_total"),
				"Total number of packets transmitted",
				[]string{"domain_uuid", "bridge", "interface"},
				nil),
			valueType: prometheus.CounterValue,
		},
		transmitErrorsTotal: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, interfaceSubsystemName, "transmit_errors_total"),
				"Total number of transmit errors",
				[]string{"domain_uuid", "bridge", "interface"},
				nil),
			valueType: prometheus.CounterValue,
		},
		transmitDropsTotal: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, interfaceSubsystemName, "transmit_drops_total"),
				"Total number of transmit drops",
				[]string{"domain_uuid", "bridge", "interface"},
				nil),
			valueType: prometheus.CounterValue,
		},
		logger: logger,
	}, nil
}

func (c *interfaceCollector) Update(ch chan<- prometheus.Metric, opts ...CollectorOption) error {
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

	wgCounter := 0
	for _, lvDomain := range lvDomains {
		wgCounter += len(lvDomain.Schema.Devices.Interfaces)
	}
	wg := sync.WaitGroup{}
	wg.Add(wgCounter)

	for _, lvDomain := range lvDomains {
		domainUUID := lvDomain.Schema.UUID
		for _, iface := range lvDomain.Schema.Devices.Interfaces {
			if iface.Target.Device == "" {
				level.Debug(c.logger).Log("msg", "interface has no target device", "domain", lvDomain.Domain.Name)
				wg.Done()
				continue
			}

			interfaceName := iface.Target.Device
			bridgeName := iface.Source.Bridge
			go func(domain libvirt.Domain, domainUUID, bridgeName, interfaceName string) {
				rRxBytes, rRxPackets, rRxErrs, rRxDrop, rTxBytes, rTxPackets, rTxErrs, rTxDrop, err := pLibvirt.DomainInterfaceStats(domain, interfaceName)
				if err != nil {
					level.Error(c.logger).Log("msg", "failed to get interface stats", "domain", domain.Name, "interface", interfaceName, "err", err)
					wg.Done()
					return
				}
				promLabels := []string{domainUUID, bridgeName, interfaceName}
				ch <- c.receiveBytesTotal.mustNewConstMetric(float64(rRxBytes), promLabels...)
				ch <- c.receivePacketsTotal.mustNewConstMetric(float64(rRxPackets), promLabels...)
				ch <- c.receiveErrorsTotal.mustNewConstMetric(float64(rRxErrs), promLabels...)
				ch <- c.receiveDropsTotal.mustNewConstMetric(float64(rRxDrop), promLabels...)
				ch <- c.transmitBytesTotal.mustNewConstMetric(float64(rTxBytes), promLabels...)
				ch <- c.transmitPacketsTotal.mustNewConstMetric(float64(rTxPackets), promLabels...)
				ch <- c.transmitErrorsTotal.mustNewConstMetric(float64(rTxErrs), promLabels...)
				ch <- c.transmitDropsTotal.mustNewConstMetric(float64(rTxDrop), promLabels...)

				wg.Done()
			}(lvDomain.Domain, domainUUID, bridgeName, interfaceName)
		}
	}
	wg.Wait()

	return nil
}
