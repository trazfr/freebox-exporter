package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/trazfr/freebox-exporter/fbx"
	"github.com/trazfr/freebox-exporter/log"
)

const (
	namespace = "freebox"
)

var (
	info = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "info",
			Help:      "A constant metric with value=0. Various information about the Freebox",
		}, []string{"firmware", "mac", "serial", "boardname", "box_flavor", "state", "media", "ipv4", "ipv6"})

	collectorSystemUptimeValue = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "system_uptime",
		Help:      "freebox uptime (in seconds)",
	})
	collectorVecSystemTemp = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "system_temp_degrees",
		Help:      "temperature (°C)",
	}, []string{"probe"})
	collectorSystemTempCpum = collectorVecSystemTemp.WithLabelValues("cpum")
	collectorSystemTempCpub = collectorVecSystemTemp.WithLabelValues("cpub")
	collectorSystemTempSw   = collectorVecSystemTemp.WithLabelValues("sw")
	collectorSystemFanRpm   = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "system_fan_rpm",
		Help:      "fan rpm",
	})
	collectorVecConnectionRate = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "connection_rate",
		Help:      "current upload/download rate in byte/s",
	}, []string{"dir"})
	collectorConnectionRateUp      = collectorVecConnectionRate.WithLabelValues("up")
	collectorConnectionRateDown    = collectorVecConnectionRate.WithLabelValues("down")
	collectorVecConnectionBandwith = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "connection_bandwith",
		Help:      "   available upload/download bandwidth in bit/s",
	}, []string{"dir"})
	collectorConnectionBandwithUp   = collectorVecConnectionBandwith.WithLabelValues("up")
	collectorConnectionBandwithDown = collectorVecConnectionBandwith.WithLabelValues("down")
	collectorVecConnectionBytes     = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "connection_bytes",
		Help:      "total uploaded/downloaded bytes since last connection",
	}, []string{"dir"})
	collectorConnectionBytesUp   = collectorVecConnectionBytes.WithLabelValues("up")
	collectorConnectionBytesDown = collectorVecConnectionBytes.WithLabelValues("down")
	collectorVecConnectionUptime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "connection_uptime",
		Help:      "Uptime in seconds",
	}, []string{"media"})
	collectorConnectionUptimeXdsl     = collectorVecConnectionUptime.WithLabelValues("xdsl")
	collectorVecConnectionXdslMaxrate = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "connection_xdsl_maxrate_kbps",
		Help:      "ATM max rate in kbit/s",
	}, []string{"dir"})
	collectorConnectionXdslMaxrateUp   = collectorVecConnectionXdslMaxrate.WithLabelValues("up")
	collectorConnectionXdslMaxrateDown = collectorVecConnectionXdslMaxrate.WithLabelValues("down")
	collectorVecConnectionXdslRate     = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "connection_xdsl_rate_kbps",
		Help:      "ATM rate in kbit/s",
	}, []string{"dir"})
	collectorConnectionXdslRateUp   = collectorVecConnectionXdslRate.WithLabelValues("up")
	collectorConnectionXdslRateDown = collectorVecConnectionXdslRate.WithLabelValues("down")
	collectorVecConnectionXdslSnr   = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "connection_xdsl_snr_db",
		Help:      "in Db",
	}, []string{"dir"})
	collectorConnectionXdslSnrUp   = collectorVecConnectionXdslSnr.WithLabelValues("up")
	collectorConnectionXdslSnrDown = collectorVecConnectionXdslSnr.WithLabelValues("down")
	collectorVecConnectionXdslAttn = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "connection_xdsl_attn_db",
		Help:      "in Db",
	}, []string{"dir"})
	collectorConnectionXdslAttnUp   = collectorVecConnectionXdslAttn.WithLabelValues("up")
	collectorConnectionXdslAttnDown = collectorVecConnectionXdslAttn.WithLabelValues("down")
)

type context struct {
	freebox *fbx.FreeboxConnection
	used    map[prometheus.Metric]bool
}

func (c *context) Describe(ch chan<- *prometheus.Desc) {
	log.Debug.Println("Describe")
	ch2 := make(chan prometheus.Metric)
	go func() {
		c.Collect(ch2)
		close(ch2)
	}()
	metrics := make([]prometheus.Metric, 16)
	for v := range ch2 {
		metrics = append(metrics, v)
		ch <- v.Desc()
	}
	for _, v := range metrics {
		c.used[v] = true
	}
}

func (c *context) Collect(ch chan<- prometheus.Metric) {
	log.Debug.Println("Collect")
	wg := sync.WaitGroup{}
	wg.Add(2)

	var firmwareVersion string
	var mac string
	var serial string
	var boardName string
	var boxFlavor string
	go func() {
		defer wg.Done()
		log.Debug.Println("Collect system")
		if m, err := c.freebox.GetMetricsSystem(); err == nil {
			firmwareVersion = m.FirmwareVersion
			mac = m.Mac
			serial = m.Serial
			boardName = m.BoardName
			boxFlavor = m.BoxFlavor

			c.collectGauge(ch, collectorSystemUptimeValue, m.UptimeValue)
			c.collectGauge(ch, collectorSystemTempCpum, m.TempCPUM)
			c.collectGauge(ch, collectorSystemTempCpub, m.TempCPUB)
			c.collectGauge(ch, collectorSystemTempSw, m.TempSW)
			c.collectGauge(ch, collectorSystemFanRpm, m.FanRpm)
		} else {
			log.Info.Println(err)
		}
	}()

	var cnxState string
	var cnxMedia string
	var cnxIPv4 string
	var cnxIPv6 string
	go func() {
		defer wg.Done()
		log.Debug.Println("Collect connection")
		if m, err := c.freebox.GetMetricsConnection(); err == nil {
			cnxState = m.State
			cnxMedia = m.Media
			cnxIPv4 = m.IPv4
			cnxIPv6 = m.IPv6

			c.collectGauge(ch, collectorConnectionRateUp, m.RateUp)
			c.collectGauge(ch, collectorConnectionRateDown, m.RateDown)
			c.collectGauge(ch, collectorConnectionBandwithUp, m.BandwithUp)
			c.collectGauge(ch, collectorConnectionBandwithDown, m.BandwithDown)
			c.collectGauge(ch, collectorConnectionBytesUp, m.BytesUp)
			c.collectGauge(ch, collectorConnectionBytesDown, m.BytesDown)
			if m.Xdsl != nil {
				if m.Xdsl.Status != nil {
					c.collectGauge(ch, collectorConnectionUptimeXdsl, m.Xdsl.Status.Uptime)
				}
				if m.Xdsl.Up != nil {
					x := m.Xdsl.Up
					c.collectGauge(ch, collectorConnectionXdslMaxrateUp, x.Maxrate)
					c.collectGauge(ch, collectorConnectionXdslRateUp, x.Rate)
					if c.use(collectorConnectionXdslSnrUp) && x.Snr10 != nil {
						collectorConnectionXdslSnrUp.Set(float64(*x.Snr10) / 10)
						collectorConnectionXdslSnrUp.Collect(ch)
					} else {
						c.collectGauge(ch, collectorConnectionXdslSnrUp, x.Snr)
					}
					if c.use(collectorConnectionXdslAttnUp) && x.Attn10 != nil {
						collectorConnectionXdslAttnUp.Set(float64(*x.Attn10) / 10)
						collectorConnectionXdslAttnUp.Collect(ch)
					} else {
						c.collectGauge(ch, collectorConnectionXdslAttnUp, x.Attn)
					}
				}
				if m.Xdsl.Down != nil {
					x := m.Xdsl.Down
					c.collectGauge(ch, collectorConnectionXdslMaxrateDown, x.Maxrate)
					c.collectGauge(ch, collectorConnectionXdslRateDown, x.Rate)
					if c.use(collectorConnectionXdslSnrDown) && x.Snr10 != nil {
						collectorConnectionXdslSnrDown.Set(float64(*x.Snr10) / 10)
						collectorConnectionXdslSnrDown.Collect(ch)
					} else {
						c.collectGauge(ch, collectorConnectionXdslSnrDown, x.Snr)
					}
					if c.use(collectorConnectionXdslAttnDown) && x.Attn10 != nil {
						collectorConnectionXdslAttnDown.Set(float64(*x.Attn10) / 10)
						collectorConnectionXdslAttnDown.Collect(ch)
					} else {
						c.collectGauge(ch, collectorConnectionXdslAttnDown, x.Attn)
					}
				}
			}
		} else {
			log.Info.Println(err)
		}
	}()

	wg.Wait()
	info.WithLabelValues(firmwareVersion,
		mac,
		serial,
		boardName,
		boxFlavor,
		cnxState,
		cnxMedia,
		cnxIPv4,
		cnxIPv6).Collect(ch)
}

func (c *context) collectGauge(ch chan<- prometheus.Metric, i prometheus.Gauge, value *int64) {
	if c.use(i) && value != nil {
		i.Set(float64(*value))
		i.Collect(ch)
	}
}

func (c *context) use(i prometheus.Metric) bool {
	found, _ := c.used[i]
	return found || len(c.used) == 0
}

func getContext(filename string) context {
	result := context{
		used: make(map[prometheus.Metric]bool),
	}
	newConfig := false
	if r, err := os.Open(filename); err == nil {
		log.Info.Println("Use configuration file", filename)
		defer r.Close()
		result.freebox, err = fbx.NewFreeboxConnectionFromConfig(r)
		if err != nil {
			panic(err)
		}
	} else {
		log.Info.Println("Could not find the configuration file", filename)
		newConfig = true
		result.freebox, err = fbx.NewFreeboxConnection()
		if err != nil {
			panic(err)
		}
	}

	if newConfig {
		log.Info.Println("Write the configuration file", filename)
		w, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
		defer w.Close()
		if err := result.freebox.WriteConfig(w); err != nil {
			panic(err)
		}
	}
	return result
}

func usage(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
	fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "[options] <configfile>")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Options:")
	flag.PrintDefaults()
	os.Exit(-1)
}

func main() {
	debugPtr := flag.Bool("debug", false, "enable the debug mode")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		usage(errors.New("configfile not defined"))
	} else if len(args) > 1 {
		usage(errors.New("invalid configfile"))
	}
	if *debugPtr {
		log.Init(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	} else {
		log.Init(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	}
	context := getContext(args[0])
	defer func() { context.freebox.Close() }()

	prometheus.MustRegister(&context)

	http.Handle("/metrics", promhttp.Handler())
	log.Error.Println(http.ListenAndServe(":9091", nil))
}
