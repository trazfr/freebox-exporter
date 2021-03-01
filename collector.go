package main

import (
	"os"
	"strconv"
	"strings"
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
		"constant metric with value=1. Various information about the Freebox",
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
	promDescConnectionBandwidth = prometheus.NewDesc(
		metricPrefix+"connection_bandwith_bps",
		"available upload/download bandwidth in bit/s",
		[]string{"dir"}, nil) // up/down
	promDescConnectionBytes = prometheus.NewDesc(
		metricPrefix+"connection_bytes",
		"total uploaded/downloaded bytes since last connection",
		[]string{"dir"}, nil) // up/down

	promDescConnectionXdslInfo = prometheus.NewDesc(
		metricPrefix+"connection_xdsl_info",
		"constant metric with value=1. Various information about the XDSL connection",
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

	promDescConnectionFtthSfpPresent = prometheus.NewDesc(
		metricPrefix+"connection_ftth_sfp_present",
		"value=1 if the SFP is present",
		[]string{"sfp_serial", "sfp_model", "sfp_vendor"}, nil)
	promDescConnectionFtthSfpAlimOk = prometheus.NewDesc(
		metricPrefix+"connection_ftth_sfp_alim_ok",
		"value=1 if the SFP's alimentation is OK",
		[]string{"sfp_serial", "sfp_model", "sfp_vendor"}, nil)
	promDescConnectionFtthSfpHasPowerReport = prometheus.NewDesc(
		metricPrefix+"connection_ftth_sfp_has_power_report",
		"value=1 if the SFP has a power report ("+metricPrefix+"connection_fttp_sfp_pwr_dbm)",
		[]string{"sfp_serial", "sfp_model", "sfp_vendor"}, nil)
	promDescConnectionFtthSfpHasSignal = prometheus.NewDesc(
		metricPrefix+"connection_ftth_sfp_has_signal",
		"value=1 if the SFP has a signal",
		[]string{"sfp_serial", "sfp_model", "sfp_vendor"}, nil)
	promDescConnectionFtthLink = prometheus.NewDesc(
		metricPrefix+"connection_ftth_link",
		"value=1 if the link is OK",
		[]string{"sfp_serial", "sfp_model", "sfp_vendor"}, nil)
	promDescConnectionFtthSfpPwr = prometheus.NewDesc(
		metricPrefix+"connection_fttp_sfp_pwr_dbm",
		"SFP power report in Dbm",
		[]string{"sfp_serial", "sfp_model", "sfp_vendor", "dir"}, nil) // rx/tx

	promDescSwitchPortConnectedTotal = prometheus.NewDesc(
		metricPrefix+"switch_port_connected_total",
		"number of ports connnected",
		nil, nil)
	promDescSwitchPortBandwidth = prometheus.NewDesc(
		metricPrefix+"switch_port_bandwidth",
		"in bps",
		[]string{"id", "link", "duplex"}, nil) // rx/tx
	promDescSwitchPortBytes = prometheus.NewDesc(
		metricPrefix+"switch_port_bytes",
		"total rx/tx bytes",
		[]string{"id", "dir", "state"}, nil) // rx/tx, good/bad
	promDescSwitchHostTotal = prometheus.NewDesc(
		metricPrefix+"switch_host_total",
		"number of hosts connected to the switch",
		[]string{"id"}, nil)
	promDescSwitchHost = prometheus.NewDesc(
		metricPrefix+"switch_host",
		"constant metric with value=1. List of MAC addresses connected to the switch",
		[]string{"id", "mac", "hostname"}, nil)

	promDescWifiBssInfo = prometheus.NewDesc(
		metricPrefix+"wifi_bss_info",
		"constant metric with value=1. Various information about the BSS",
		[]string{"bssid", "ap_id", "state", "enabled", "ssid", "hide_ssid", "encryption"}, nil)
	promDescWifiBssStationTotal = prometheus.NewDesc(
		metricPrefix+"wifi_bss_station_total",
		"number of stations on this BSS",
		[]string{"bssid", "ap_id"}, nil)
	promDescWifiBssAuthorizedStationTotal = prometheus.NewDesc(
		metricPrefix+"wifi_bss_authorized_station_total",
		"number of stations authorized on this BSS",
		[]string{"bssid", "ap_id"}, nil)

	promDescWifiApChannel = prometheus.NewDesc(
		metricPrefix+"wifi_ap_channel",
		"channel number",
		[]string{"ap_id", "ap_band", "ap_name", "channel_type"}, nil)
	promDescWifiApStationTotal = prometheus.NewDesc(
		metricPrefix+"wifi_station_total",
		"number of stations connected to the AP",
		[]string{"ap_id", "ap_band", "ap_name"}, nil)
	promDescWifiApStation = prometheus.NewDesc(
		metricPrefix+"wifi_station",
		"1 if active, 0 if not",
		[]string{"ap_id", "ap_band", "ap_name", "id", "bssid", "ssid", "encryption", "hostname", "mac"}, nil)
	promDescWifiApStationBytes = prometheus.NewDesc(
		metricPrefix+"wifi_station_bytes",
		"total rx/tx bytes",
		[]string{"id", "dir"}, nil)
	promDescWifiApStationSignalDb = prometheus.NewDesc(
		metricPrefix+"wifi_station_signal_db",
		"signal attenuation in dB",
		[]string{"id"}, nil)

	promDescLanHostTotal = prometheus.NewDesc(
		metricPrefix+"lan_host_total",
		"number of hosts detected",
		[]string{"interface", "active"}, nil)
	promDescLanHostActiveL2 = prometheus.NewDesc(
		metricPrefix+"lan_host_active_l2",
		"1 if active, 0 if not. Various information about the l2 addresses",
		[]string{"interface", "vendor_name", "primary_name", "host_type", "l2_type", "l2_id"}, nil)
	promDescLanHostActiveL3 = prometheus.NewDesc(
		metricPrefix+"lan_host_active_l3",
		"1 if active, 0 if not. Various information about the l3 addresses",
		[]string{"interface", "vendor_name", "primary_name", "host_type", "l2_type", "l2_id", "l3_type", "l3_address"}, nil)
)

// Collector is the prometheus collector for the freebox exporter
type Collector struct {
	hostDetails bool
	freebox     *fbx.FreeboxConnection
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
	wg.Add(5)

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
			mac = strings.ToLower(m.Mac)
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

			c.collectGauge(ch, m.BandwidthUp, promDescConnectionBandwidth, "up")
			c.collectGauge(ch, m.BandwidthDown, promDescConnectionBandwidth, "down")
			c.collectCounter(ch, m.BytesUp, promDescConnectionBytes, "up")
			c.collectCounter(ch, m.BytesDown, promDescConnectionBytes, "down")
			if m.Xdsl != nil {
				if m.Xdsl.Status != nil {
					ch <- prometheus.MustNewConstMetric(promDescConnectionXdslInfo, prometheus.GaugeValue, 1,
						m.Xdsl.Status.Status,
						m.Xdsl.Status.Protocol,
						m.Xdsl.Status.Modulation)

					c.collectCounter(ch, m.Xdsl.Status.Uptime, promDescConnectionXdslUptime)
				}
				c.collectXdslStats(ch, m.Xdsl.Up, "up")
				c.collectXdslStats(ch, m.Xdsl.Down, "down")
			}
			if m.Ftth != nil {
				c.collectBool(ch, m.Ftth.SfpPresent, promDescConnectionFtthSfpPresent,
					m.Ftth.SfpSerial,
					m.Ftth.SfpModel,
					m.Ftth.SfpVendor)
				c.collectBool(ch, m.Ftth.SfpAlimOk, promDescConnectionFtthSfpAlimOk,
					m.Ftth.SfpSerial,
					m.Ftth.SfpModel,
					m.Ftth.SfpVendor)
				c.collectBool(ch, m.Ftth.SfpHasPowerReport, promDescConnectionFtthSfpHasPowerReport,
					m.Ftth.SfpSerial,
					m.Ftth.SfpModel,
					m.Ftth.SfpVendor)
				c.collectBool(ch, m.Ftth.SfpHasSignal, promDescConnectionFtthSfpHasSignal,
					m.Ftth.SfpSerial,
					m.Ftth.SfpModel,
					m.Ftth.SfpVendor)
				c.collectBool(ch, m.Ftth.Link, promDescConnectionFtthLink,
					m.Ftth.SfpSerial,
					m.Ftth.SfpModel,
					m.Ftth.SfpVendor)
				c.collectGaugeWithFactor(ch, m.Ftth.SfpPwrTx, 0.01, promDescConnectionFtthSfpPwr,
					m.Ftth.SfpSerial,
					m.Ftth.SfpModel,
					m.Ftth.SfpVendor,
					"tx")
				c.collectGaugeWithFactor(ch, m.Ftth.SfpPwrRx, 0.01, promDescConnectionFtthSfpPwr,
					m.Ftth.SfpSerial,
					m.Ftth.SfpModel,
					m.Ftth.SfpVendor,
					"rx")
			}
		} else {
			log.Error.Println(err)
		}
	}()

	go func() {
		defer wg.Done()
		log.Debug.Println("Collect switch")

		if m, err := c.freebox.GetMetricsSwitch(); err == nil {
			numPortsConnected := 0

			for _, port := range m.Ports {
				if port.Link == "up" {
					numPortsConnected++
				}

				speed, _ := strconv.Atoi(port.Speed)
				portID := strconv.FormatInt(port.ID, 10)
				ch <- prometheus.MustNewConstMetric(promDescSwitchPortBandwidth, prometheus.GaugeValue, float64(speed),
					portID,
					port.Link,
					port.Duplex)
				if port.Stats != nil {
					c.collectCounter(ch, port.Stats.RxGoodBytes, promDescSwitchPortBytes, portID, "rx", "good")
					c.collectCounter(ch, port.Stats.RxBadBytes, promDescSwitchPortBytes, portID, "rx", "bad")
					c.collectCounter(ch, port.Stats.TxBytes, promDescSwitchPortBytes, portID, "tx", "")
				}

				ch <- prometheus.MustNewConstMetric(promDescSwitchHostTotal, prometheus.GaugeValue, float64(len(port.MacList)), portID)
				if c.hostDetails {
					for _, mac := range port.MacList {
						ch <- prometheus.MustNewConstMetric(promDescSwitchHost, prometheus.GaugeValue, 1,
							portID,
							strings.ToLower(mac.Mac),
							mac.Hostname)
					}
				}
			}

			ch <- prometheus.MustNewConstMetric(promDescSwitchPortConnectedTotal, prometheus.GaugeValue, float64(numPortsConnected))
		} else {
			log.Error.Println(err)
		}
	}()

	go func() {
		defer wg.Done()
		log.Debug.Println("Collect wifi")
		if m, err := c.freebox.GetMetricsWifi(); err == nil {

			for _, bss := range m.Bss {
				phyID := strconv.FormatInt(bss.PhyID, 10)
				bssid := strings.ToLower(bss.ID)
				ch <- prometheus.MustNewConstMetric(promDescWifiBssInfo, prometheus.GaugeValue, 1,
					bssid,
					phyID,
					bss.Status.State,
					c.toString(bss.Config.Enabled),
					bss.Config.Ssid,
					c.toString(bss.Config.HideSsid),
					bss.Config.Encryption)
				c.collectGauge(ch, bss.Status.StaCount, promDescWifiBssStationTotal, bssid, phyID)
				c.collectGauge(ch, bss.Status.AuthorizedStaCount, promDescWifiBssAuthorizedStationTotal, bssid, phyID)
			}

			for _, ap := range m.Ap {
				apID := strconv.FormatInt(ap.ID, 10)

				if capabilities, ok := ap.Capabilities[ap.Config.Band]; ok {
					labels := prometheus.Labels{}
					labels["ap_id"] = apID
					labels["ap_band"] = ap.Config.Band
					labels["ap_name"] = ap.Name
					labels["ap_state"] = ap.Status.State

					for k, v := range capabilities {
						labels[k] = strconv.FormatBool(v)
					}

					promDescWifiApCapabilities := prometheus.NewDesc(
						metricPrefix+"wifi_ap_capabilities",
						"constant metric with value=1. List of AP capabilities",
						nil, labels)
					ch <- prometheus.MustNewConstMetric(promDescWifiApCapabilities, prometheus.GaugeValue, 1)

				}

				c.collectGauge(ch, ap.Status.PrimaryChannel, promDescWifiApChannel,
					apID,
					ap.Config.Band,
					ap.Name,
					"primary")
				c.collectGauge(ch, ap.Status.SecondaryChannel, promDescWifiApChannel,
					apID,
					ap.Config.Band,
					ap.Name,
					"secondary")
				ch <- prometheus.MustNewConstMetric(promDescWifiApStationTotal, prometheus.GaugeValue, float64(len(ap.Stations)),
					apID,
					ap.Config.Band,
					ap.Name)
				if c.hostDetails {
					for _, station := range ap.Stations {
						stationActive := float64(0)
						if station.Host != nil && c.toBool(station.Host.Active) {
							stationActive = 1
						}
						bssid := strings.ToLower(station.Bssid)
						mac := strings.ToLower(station.Mac)
						stationID := strings.ToLower(station.ID)
						ssid := ""
						encryption := ""
						if station.Bss != nil {
							ssid = station.Bss.Config.Ssid
							encryption = station.Bss.Config.Encryption
						}

						ch <- prometheus.MustNewConstMetric(promDescWifiApStation, prometheus.GaugeValue, stationActive,
							apID,
							ap.Config.Band,
							ap.Name,
							stationID,
							bssid,
							ssid,
							encryption,
							station.Hostname,
							mac)
						c.collectCounter(ch, station.RxBytes, promDescWifiApStationBytes,
							stationID,
							"rx",
						)
						c.collectCounter(ch, station.TxBytes, promDescWifiApStationBytes,
							stationID,
							"tx",
						)
						c.collectGauge(ch, station.Signal, promDescWifiApStationSignalDb,
							stationID)
					}
				}
			}
		} else {
			log.Error.Println(err)
		}

	}()

	go func() {
		defer wg.Done()
		log.Debug.Println("Collect lan")

		if m, err := c.freebox.GetMetricsLan(); err == nil {
			for name, hosts := range m.Hosts {
				hostsUnknown := 0
				hostsActive := 0
				hostsInactive := 0

				for _, host := range hosts {
					if host != nil {
						active := c.toBool(host.Active)
						if active {
							hostsActive++
						} else {
							hostsInactive++
						}

						if c.hostDetails {
							l2Active := float64(0)
							if active {
								l2Active = 1
							}
							ch <- prometheus.MustNewConstMetric(promDescLanHostActiveL2, prometheus.GaugeValue, l2Active,
								name,
								host.VendorName,
								host.PrimaryName,
								host.HostType,
								host.L2Ident.Type,
								strings.ToLower(host.L2Ident.ID))

							for _, l3 := range host.L3Connectivities {
								l3Active := float64(0)
								if c.toBool(l3.Active) {
									l3Active = 1
								}
								ch <- prometheus.MustNewConstMetric(promDescLanHostActiveL3, prometheus.GaugeValue, l3Active,
									name,
									host.VendorName,
									host.PrimaryName,
									host.HostType,
									host.L2Ident.Type,
									strings.ToLower(host.L2Ident.ID),
									l3.Af,
									l3.Addr)
							}
						}
					} else {
						hostsUnknown += len(hosts)
					}
				}

				ch <- prometheus.MustNewConstMetric(promDescLanHostTotal, prometheus.GaugeValue, float64(hostsActive), name, "true")
				ch <- prometheus.MustNewConstMetric(promDescLanHostTotal, prometheus.GaugeValue, float64(hostsInactive), name, "false")
				if hostsUnknown > 0 {
					// there should not be any
					ch <- prometheus.MustNewConstMetric(promDescLanHostTotal, prometheus.GaugeValue, float64(hostsUnknown), name, "unknown")
				}
			}
		} else {
			log.Error.Println(err)
		}
	}()

	wg.Wait()
	ch <- prometheus.MustNewConstMetric(promDescInfo, prometheus.GaugeValue, 1,
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

func (c *Collector) collectBool(ch chan<- prometheus.Metric, value *bool, desc *prometheus.Desc, labels ...string) {
	if value != nil {
		v := float64(0)
		if *value {
			v = 1
		}
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, labels...)
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
	return "unknown"
}

func (c *Collector) toBool(b *bool) bool {
	return b != nil && *b
}

func NewCollector(filename string, discovery fbx.FreeboxDiscovery, hostDetails, debug bool) *Collector {
	result := &Collector{
		hostDetails: hostDetails,
	}
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
		result.freebox, err = fbx.NewFreeboxConnectionFromServiceDiscovery(debug, discovery)
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
