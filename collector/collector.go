package collector

import (
	"fmt"
	"sync"
	"time"

	"github.com/alecthomas/kingpin/v2"
	libvirt "github.com/digitalocean/go-libvirt"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/nee541/libvirt-exporter/libvirt_schema"
	"github.com/prometheus/client_golang/prometheus"
)

// Namespace defines the common namespace to be used by all metrics.
const namespace = "libvirt"

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"node_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"node_exporter: Whether a collector succeeded.",
		[]string{"collector"},
		nil,
	)
)

const (
	defaultEnabled  = true
	defaultDisabled = false
)

var (
	factories              = make(map[string]func(logger log.Logger) (Collector, error))
	initiatedCollectorsMtx = sync.Mutex{}
	initiatedCollectors    = make(map[string]Collector)
	collectorState         = make(map[string]*bool)
	forcedCollectors       = map[string]bool{} // collectors which have been explicitly enabled or disabled
)

func registerCollector(collector string, isDefaultEnabled bool, factory func(logger log.Logger) (Collector, error)) {
	var helpDefaultState string
	if isDefaultEnabled {
		helpDefaultState = "enabled"
	} else {
		helpDefaultState = "disabled"
	}

	flagName := fmt.Sprintf("collector.%s", collector)
	flagHelp := fmt.Sprintf("Enable the %s collector (default: %s).", collector, helpDefaultState)
	defaultValue := fmt.Sprintf("%v", isDefaultEnabled)

	flag := kingpin.Flag(flagName, flagHelp).Default(defaultValue).Action(collectorFlagAction(collector)).Bool()
	collectorState[collector] = flag

	factories[collector] = factory
}

// LibvirtCollector implements the prometheus.Collector interface.
type LibvirtCollector struct {
	Collectors map[string]Collector
	pLibvirt   *libvirt.Libvirt
	logger     log.Logger
}

// DisableDefaultCollectors sets the collector state to false for all collectors which
// have not been explicitly enabled on the command line.
func DisableDefaultCollectors() {
	for c := range collectorState {
		if _, ok := forcedCollectors[c]; !ok {
			*collectorState[c] = false
		}
	}
}

// collectorFlagAction generates a new action function for the given collector
// to track whether it has been explicitly enabled or disabled from the command line.
// A new action function is needed for each collector flag because the ParseContext
// does not contain information about which flag called the action.
// See: https://github.com/alecthomas/kingpin/issues/294
func collectorFlagAction(collector string) func(ctx *kingpin.ParseContext) error {
	return func(ctx *kingpin.ParseContext) error {
		forcedCollectors[collector] = true
		return nil
	}
}

// NewLibvirtCollector creates a new LibvirtCollector.
func NewLibvirtCollector(pLibvirt *libvirt.Libvirt, logger log.Logger, filters ...string) (*LibvirtCollector, error) {
	f := make(map[string]bool)
	for _, filter := range filters {
		enabled, exist := collectorState[filter]
		if !exist {
			return nil, fmt.Errorf("missing collector: %s", filter)
		}
		if !*enabled {
			return nil, fmt.Errorf("disabled collector: %s", filter)
		}
		f[filter] = true
	}
	collectors := make(map[string]Collector)
	initiatedCollectorsMtx.Lock()
	defer initiatedCollectorsMtx.Unlock()
	for key, enabled := range collectorState {
		if !*enabled || (len(f) > 0 && !f[key]) {
			continue
		}
		if collector, ok := initiatedCollectors[key]; ok {
			collectors[key] = collector
		} else {
			collector, err := factories[key](log.With(logger, "collector", key))
			if err != nil {
				return nil, err
			}
			collectors[key] = collector
			initiatedCollectors[key] = collector
		}
	}
	return &LibvirtCollector{Collectors: collectors, pLibvirt: pLibvirt, logger: logger}, nil
}

// Describe implements the prometheus.Collector interface.
func (n LibvirtCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (n LibvirtCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(n.Collectors))

	// manage libvirt connection
	if n.pLibvirt == nil {
		level.Error(n.logger).Log("msg", "libvirt not created")
		return
	}
	if n.pLibvirt.Connect() != nil {
		level.Error(n.logger).Log("msg", "libvirt not connected")
		return
	}
	defer n.pLibvirt.Disconnect()

	flags := libvirt.ConnectListDomainsActive | libvirt.ConnectListDomainsInactive
	domains, num, err := n.pLibvirt.ConnectListAllDomains(1, flags)
	if err != nil {
		level.Error(n.logger).Log("msg", "failed to list domains", "err", err)
		return
	}
	level.Debug(n.logger).Log("msg", "list domains", "num", num)
	lvDomains := make([]lvDomain, num)
	for i, domain := range domains {
		xmlDesc, err := n.pLibvirt.DomainGetXMLDesc(domain, 0)
		if err != nil {
			level.Error(n.logger).Log("msg", "failed to get domain xml", "err", err)
			return
		}
		schema, err := libvirt_schema.NewDomainFromXML([]byte(xmlDesc))
		if err != nil {
			level.Error(n.logger).Log("msg", "failed to parse domain xml", "err", err)
			return
		}

		lvDomains[i] = lvDomain{
			domain: &domain,
			schema: schema,
		}
	}

	for name, c := range n.Collectors {
		go func(name string, c Collector) {
			execute(name, c, ch, n.pLibvirt, lvDomains, n.logger)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
}

func execute(name string, c Collector, ch chan<- prometheus.Metric, pLibvirt *libvirt.Libvirt, lvDomains []lvDomain, logger log.Logger) {
	begin := time.Now()

	// prepare data for collector and Update data
	// TODO: select data for collector
	err := c.Update(ch, WithLibvirt(pLibvirt), WithDomains(lvDomains))

	duration := time.Since(begin)
	var success float64

	if err != nil {
		if IsNoDataError(err) {
			level.Debug(logger).Log("msg", "collector returned no data", "name", name, "duration_seconds", duration.Seconds(), "err", err)
		} else if IsNotProvidedError(err) {
			level.Debug(logger).Log("msg", "collector not provided with necessary data", "name", name, "duration_seconds", duration.Seconds(), "err", err)
		} else {
			level.Error(logger).Log("msg", "collector failed", "name", name, "duration_seconds", duration.Seconds(), "err", err)
		}
		success = 0
	} else {
		level.Debug(logger).Log("msg", "collector succeeded", "name", name, "duration_seconds", duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}

// Collector is the interface a collector has to implement.
type Collector interface {
	// Get new metrics and expose them via prometheus registry.
	Update(ch chan<- prometheus.Metric, opts ...CollectorOption) error
}

// Function Options/Functional Arguments
type CollectorConfig struct {
	pLibvirt  *libvirt.Libvirt
	lvDomains []lvDomain
}

type CollectorOption func(*CollectorConfig)

func WithLibvirt(lv *libvirt.Libvirt) CollectorOption {
	return func(c *CollectorConfig) {
		c.pLibvirt = lv
	}
}

func WithDomains(lvDomains []lvDomain) CollectorOption {
	return func(c *CollectorConfig) {
		c.lvDomains = lvDomains
	}
}

type typedDesc struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}

func (d *typedDesc) mustNewConstMetric(value float64, labels ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(d.desc, d.valueType, value, labels...)
}

type lvDomain struct {
	domain *libvirt.Domain
	schema *libvirt_schema.Domain
}

// pushMetric helps construct and convert a variety of value types into Prometheus float64 metrics.
// func pushMetric(ch chan<- prometheus.Metric, fieldDesc *prometheus.Desc, name string, value interface{}, valueType prometheus.ValueType, labelValues ...string) {
// 	var fVal float64
// 	switch val := value.(type) {
// 	case uint8:
// 		fVal = float64(val)
// 	case uint16:
// 		fVal = float64(val)
// 	case uint32:
// 		fVal = float64(val)
// 	case uint64:
// 		fVal = float64(val)
// 	case int64:
// 		fVal = float64(val)
// 	case *uint8:
// 		if val == nil {
// 			return
// 		}
// 		fVal = float64(*val)
// 	case *uint16:
// 		if val == nil {
// 			return
// 		}
// 		fVal = float64(*val)
// 	case *uint32:
// 		if val == nil {
// 			return
// 		}
// 		fVal = float64(*val)
// 	case *uint64:
// 		if val == nil {
// 			return
// 		}
// 		fVal = float64(*val)
// 	case *int64:
// 		if val == nil {
// 			return
// 		}
// 		fVal = float64(*val)
// 	default:
// 		return
// 	}

// 	ch <- prometheus.MustNewConstMetric(fieldDesc, valueType, fVal, labelValues...)
// }
