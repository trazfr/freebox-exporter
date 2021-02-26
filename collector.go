package main

import (
	"os"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/trazfr/freebox-exporter/fbx"
	"github.com/trazfr/freebox-exporter/log"
)

const (
	metricPrefix = "freebox_"
)

var (
	promDescInfo = prometheus.NewDesc(
		metricPrefix+"info",
		"constant metric with value=0. Various information about the Freebox",
		[]string{"firmware", "mac", "serial", "boardname", "box_flavor", "connection_type", "connection_state", "connection_media", "ipv4", "ipv6"}, nil)

	promDescSystemUptime = prometheus.NewDesc(
		metricPrefix+"system_uptime",
		"freebox uptime (in seconds)",
		nil, nil)
	promDescSystemTemp = prometheus.NewDesc(
		metricPrefix+"system_temp_degrees",
		"temperature (°C)",
		[]string{"id"}, nil)
	promDescSystemFanRpm = prometheus.NewDesc(
		metricPrefix+"system_fan_rpm",
		"fan rpm",
		[]string{"id"}, nil)
	promDescConnectionRate = prometheus.NewDesc(
		metricPrefix+"connection_rate",
		"current upload/download rate in byte/s",
		[]string{"dir"}, nil) // up/down. XXX delme?
	promDescConnectionBandwith = prometheus.NewDesc(
		metricPrefix+"connection_bandwith_bps",
		"available upload/download bandwidth in bit/s",
		[]string{"dir"}, nil) // up/down
	promDescConnectionBytes = prometheus.NewDesc(
		metricPrefix+"connection_bytes",
		"total uploaded/downloaded bytes since last connection",
		[]string{"dir"}, nil) // up/down
	promDescConnectionXdslInfo = prometheus.NewDesc(
		metricPrefix+"connection_xdsl_info",
		"constant metric with value=0. Various information about the XDSL connection",
		[]string{"status", "protocol", "modulation"}, nil)
	promDescConnectionXdslUptime = prometheus.NewDesc(
		metricPrefix+"connection_xdsl_uptime",
		"uptime in seconds",
		nil, nil)
	promDescConnectionXdslMaxRate = prometheus.NewDesc(
		metricPrefix+"connection_xdsl_maxrate_bps",
		"ATM max rate in bit/s",
		[]string{"dir"}, nil) // up/down
	promDescConnectionXdslRate = prometheus.NewDesc(
		metricPrefix+"connection_xdsl_rate_bps",
		"ATM rate in bit/s",
		[]string{"dir"}, nil) // up/down
	promDescConnectionXdslSnr = prometheus.NewDesc(
		metricPrefix+"connection_xdsl_snr_db",
		"in Db",
		[]string{"dir"}, nil) // up/down
	promDescConnectionXdslAttn = prometheus.NewDesc(
		metricPrefix+"connection_xdsl_attn_db",
		"in Db",
		[]string{"dir"}, nil) // up/down
	promDescConnectionFtthInfo = prometheus.NewDesc(
		metricPrefix+"connection_ftth_info",
		"constant metric with value=0. Various information about the FTTH connection",
		[]string{"sfp_present", "sfp_alim_ok", "sfp_has_power_report", "sfp_has_signal", "link", "sfp_serial", "sfp_model", "sfp_vendor"}, nil)
	promDescConnectionFtthSfpPwr = prometheus.NewDesc(
		metricPrefix+"connection_fttp_sfp_pwr_dbm",
		"in Dbm",
		[]string{"dir"}, nil) // rx/tx
)

// Collector is the prometheus collector for the freebox exporter
type Collector struct {
	freebox *fbx.FreeboxConnection
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
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
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
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

			c.collectCounter(ch, m.UptimeValue, promDescSystemUptime)
			for _, sensor := range m.Sensors {
				c.collectGauge(ch, sensor.Value, promDescSystemTemp, sensor.ID)
			}
			if len(m.Sensors) == 0 {
				c.collectGauge(ch, m.TempCPUM, promDescSystemTemp, "temp_cpum")
				c.collectGauge(ch, m.TempCPUB, promDescSystemTemp, "temp_cpub")
				c.collectGauge(ch, m.TempSW, promDescSystemTemp, "temp_sw")
			}
			for _, fan := range m.Fans {
				c.collectGauge(ch, fan.Value, promDescSystemFanRpm, fan.ID)
			}
			if len(m.Fans) == 0 {
				c.collectGauge(ch, m.FanRpm, promDescSystemFanRpm, "fan")
			}
		} else {
			log.Error.Println(err)
		}
	}()

	var cnxType string
	var cnxState string
	var cnxMedia string
	var cnxIPv4 string
	var cnxIPv6 string
	go func() {
		defer wg.Done()
		log.Debug.Println("Collect connection")
		if m, err := c.freebox.GetMetricsConnection(); err == nil {
			cnxType = m.Type
			cnxState = m.State
			cnxMedia = m.Media
			cnxIPv4 = m.IPv4
			cnxIPv6 = m.IPv6

			c.collectGauge(ch, m.RateUp, promDescConnectionRate, "up")
			c.collectGauge(ch, m.RateDown, promDescConnectionRate, "down")
			c.collectGauge(ch, m.BandwidthUp, promDescConnectionBandwith, "up")
			c.collectGauge(ch, m.BandwidthDown, promDescConnectionBandwith, "down")
			c.collectCounter(ch, m.BytesUp, promDescConnectionBytes, "up")
			c.collectCounter(ch, m.BytesDown, promDescConnectionBytes, "down")
			if m.Xdsl != nil {
				if m.Xdsl.Status != nil {
					ch <- prometheus.MustNewConstMetric(promDescConnectionXdslInfo, prometheus.GaugeValue, 0,
						m.Xdsl.Status.Status,
						m.Xdsl.Status.Protocol,
						m.Xdsl.Status.Modulation)

					c.collectCounter(ch, m.Xdsl.Status.Uptime, promDescConnectionXdslUptime)
				}
				c.collectXdslStats(ch, m.Xdsl.Up, "up")
				c.collectXdslStats(ch, m.Xdsl.Down, "down")
			}
			if m.Ftth != nil {
				ch <- prometheus.MustNewConstMetric(promDescConnectionFtthInfo, prometheus.GaugeValue, 0,
					c.toString(m.Ftth.SfpPresent),
					c.toString(m.Ftth.SfpAlimOk),
					c.toString(m.Ftth.SfpHasPowerReport),
					c.toString(m.Ftth.SfpHasSignal),
					c.toString(m.Ftth.Link),
					m.Ftth.SfpSerial,
					m.Ftth.SfpModel,
					m.Ftth.SfpVendor)

				c.collectGaugeWithFactor(ch, m.Ftth.SfpPwrTx, 0.01, promDescConnectionFtthSfpPwr, "tx")
				c.collectGaugeWithFactor(ch, m.Ftth.SfpPwrRx, 0.01, promDescConnectionFtthSfpPwr, "rx")
			}
		} else {
			log.Error.Println(err)
		}
	}()

	wg.Wait()
	ch <- prometheus.MustNewConstMetric(promDescInfo, prometheus.GaugeValue, 0,
		firmwareVersion,
		mac,
		serial,
		boardName,
		boxFlavor,
		cnxType,
		cnxState,
		cnxMedia,
		cnxIPv4,
		cnxIPv6)
}

func (c *Collector) collectXdslStats(ch chan<- prometheus.Metric, stats *fbx.MetricsFreeboxConnectionXdslStats, dir string) {
	if stats != nil {
		c.collectGaugeWithFactor(ch, stats.Maxrate, 1000, promDescConnectionXdslMaxRate)
		c.collectGaugeWithFactor(ch, stats.Rate, 1000, promDescConnectionXdslRate)
		if stats.Snr10 != nil {
			c.collectGaugeWithFactor(ch, stats.Snr10, 0.1, promDescConnectionXdslSnr)
		} else {
			c.collectGauge(ch, stats.Snr, promDescConnectionXdslSnr, dir)
		}
		if stats.Attn10 != nil {
			c.collectGaugeWithFactor(ch, stats.Attn10, 0.1, promDescConnectionXdslAttn)
		} else {
			c.collectGauge(ch, stats.Attn, promDescConnectionXdslAttn, dir)
		}
	}
}

func (c *Collector) collectGauge(ch chan<- prometheus.Metric, value *int64, desc *prometheus.Desc, labels ...string) {
	c.collectConst(ch, prometheus.GaugeValue, value, desc, labels...)
}

func (c *Collector) collectCounter(ch chan<- prometheus.Metric, value *int64, desc *prometheus.Desc, labels ...string) {
	c.collectConst(ch, prometheus.CounterValue, value, desc, labels...)
}

func (c *Collector) collectConst(ch chan<- prometheus.Metric, valueType prometheus.ValueType, value *int64, desc *prometheus.Desc, labels ...string) {
	if value != nil {
		ch <- prometheus.MustNewConstMetric(desc, valueType, float64(*value), labels...)
	}
}

func (c *Collector) collectGaugeWithFactor(ch chan<- prometheus.Metric, value *int64, factor float64, desc *prometheus.Desc, labels ...string) {
	c.collectConstWithFactor(ch, prometheus.GaugeValue, value, factor, desc, labels...)
}

func (c *Collector) collectConstWithFactor(ch chan<- prometheus.Metric, valueType prometheus.ValueType, value *int64, factor float64, desc *prometheus.Desc, labels ...string) {
	if value != nil {
		ch <- prometheus.MustNewConstMetric(desc, valueType, float64(*value)*factor, labels...)
	}
}

func (c *Collector) toString(b *bool) string {
	if b != nil {
		return strconv.FormatBool(*b)
	}
	return ""
}

func NewCollector(debug bool, filename string) *Collector {
	result := &Collector{}
	newConfig := false
	if r, err := os.Open(filename); err == nil {
		log.Info.Println("Use configuration file", filename)
		defer r.Close()
		result.freebox, err = fbx.NewFreeboxConnectionFromConfig(debug, r)
		if err != nil {
			panic(err)
		}
	} else {
		log.Info.Println("Could not find the configuration file", filename)
		newConfig = true
		result.freebox, err = fbx.NewFreeboxConnection(debug)
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

func (c *Collector) Close() {
	c.freebox.Close()
}
