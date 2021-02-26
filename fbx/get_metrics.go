package fbx

import (
	"fmt"
	"sync"

	"github.com/trazfr/freebox-exporter/log"
)

// MetricsFreeboxSystem https://dev.freebox.fr/sdk/os/system/
type MetricsFreeboxSystem struct {
	FirmwareVersion  string                       `json:"firmware_version"`
	Mac              string                       `json:"mac"`
	Serial           string                       `json:"serial"`
	Uptime           string                       `json:"uptime"`
	UptimeValue      *int64                       `json:"uptime_val"`
	BoardName        string                       `json:"board_name"`
	TempCPUM         *int64                       `json:"temp_cpum"` // seems deprecated
	TempSW           *int64                       `json:"temp_sw"`   // seems deprecated
	TempCPUB         *int64                       `json:"temp_cpub"` // seems deprecated
	FanRpm           *int64                       `json:"fan_rpm"`
	BoxAuthenticated *bool                        `json:"box_authenticated"`
	DiskStatus       string                       `json:"disk_status"`
	BoxFlavor        string                       `json:"box_flavor"`
	UserMainStorage  string                       `json:"user_main_storage"`
	Sensors          []MetricsFreeboxSystemSensor `json:"sensors"` // undocumented
	Fans             []MetricsFreeboxSystemSensor `json:"fans"`    // undocumented
}

// MetricsFreeboxSystemSensor undocumented
type MetricsFreeboxSystemSensor struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value *int64 `json:"value"`
}

// MetricsFreeboxConnection https://dev.freebox.fr/sdk/os/connection/
type MetricsFreeboxConnection struct {
	State         string `json:"state"`
	Type          string `json:"type"`
	Media         string `json:"media"`
	IPv4          string `json:"ipv4"`
	IPv6          string `json:"ipv6"`
	RateUp        *int64 `json:"rate_up"`
	RateDown      *int64 `json:"rate_down"`
	BandwidthUp   *int64 `json:"bandwidth_up"`
	BandwidthDown *int64 `json:"bandwidth_down"`
	BytesUp       *int64 `json:"bytes_up"`
	BytesDown     *int64 `json:"bytes_down"`
	// ipv4_port_range
}

// MetricsFreeboxConnectionXdslStats https://dev.freebox.fr/sdk/os/connection/#XdslStats
type MetricsFreeboxConnectionXdslStats struct {
	Maxrate    *int64 `json:"maxrate"`
	Rate       *int64 `json:"rate"`
	Snr        *int64 `json:"snr"`
	Attn       *int64 `json:"attn"`
	Snr10      *int64 `json:"snr_10"`
	Attn10     *int64 `json:"attn_10"`
	Fec        *int64 `json:"fec"`
	Crc        *int64 `json:"crc"`
	Hec        *int64 `json:"hec"`
	Es         *int64 `json:"es"`
	Ses        *int64 `json:"ses"`
	Phyr       *bool  `json:"phyr"`
	Ginp       *bool  `json:"ginp"`
	Nitro      *bool  `json:"nitro"`
	Rxmt       *int64 `json:"rxmt"`        // phyr
	RxmtCorr   *int64 `json:"rxmt_corr"`   // phyr
	RxmtUncorr *int64 `json:"rxmt_uncorr"` // phyr
	RtxTx      *int64 `json:"rtx_tx"`      // ginp
	RtxC       *int64 `json:"rtx_c"`       // ginp
	RtxUc      *int64 `json:"rtx_uc"`      // ginp
}

// MetricsFreeboxConnectionXdsl https://dev.freebox.fr/sdk/os/connection/#XdslInfos
type MetricsFreeboxConnectionXdsl struct {
	// https://dev.freebox.fr/sdk/os/connection/#XdslStatus
	Status *struct {
		Status     string `json:"status"`
		Protocol   string `json:"protocol"`
		Modulation string `json:"modulation"`
		Uptime     *int64 `json:"uptime"`
	} `json:"status"`
	Down *MetricsFreeboxConnectionXdslStats `json:"down"`
	Up   *MetricsFreeboxConnectionXdslStats `json:"up"`
}

// MetricsFreeboxConnectionFtth https://dev.freebox.fr/sdk/os/connection/#FtthStatus
type MetricsFreeboxConnectionFtth struct {
	SfpPresent        *bool  `json:"sfp_present"`
	SfpAlimOk         *bool  `json:"sfp_alim_ok"`
	SfpHasPowerReport *bool  `json:"sfp_has_power_report"`
	SfpHasSignal      *bool  `json:"sfp_has_signal"`
	Link              *bool  `json:"link"`
	SfpSerial         string `json:"sfp_serial"`
	SfpModel          string `json:"sfp_model"`
	SfpVendor         string `json:"sfp_vendor"`
	SfpPwrTx          *int64 `json:"sfp_pwr_tx"`
	SfpPwrRx          *int64 `json:"sfp_pwr_rx"`
}

// MetricsFreeboxConnectionAll is the result of GetMetricsConnection()
type MetricsFreeboxConnectionAll struct {
	MetricsFreeboxConnection
	Xdsl *MetricsFreeboxConnectionXdsl
	Ftth *MetricsFreeboxConnectionFtth
}

// MetricsFreeboxSwitch https://dev.freebox.fr/sdk/os/switch/
type MetricsFreeboxSwitch struct {
	Ports []*MetricsFreeboxSwitchStatus
}

// MetricsFreeboxSwitchStatus https://dev.freebox.fr/sdk/os/switch/
type MetricsFreeboxSwitchStatus struct {
	ID      int64  `json:"id"`
	Duplex  string `json:"duplex"`
	Link    string `json:"link"`
	Mode    string `json:"mode"`
	Speed   string `json:"speed"`
	MacList []struct {
		Mac      string `json:"mac"`
		Hostname string `json:"hostname"`
	} `json:"mac_list"`
	Stats *MetricsFreeboxSwitchPortStats `json:"-"`
}

// MetricsFreeboxSwitchPortStats https://dev.freebox.fr/sdk/os/switch/#switch-port-stats-object-unstable
type MetricsFreeboxSwitchPortStats struct {
	RxBadBytes         *int64 `json:"rx_bad_bytes"`
	RxBroadcastPackets *int64 `json:"rx_broadcast_packets"`
	RxBytesRate        *int64 `json:"rx_bytes_rate"`
	RxErrPackets       *int64 `json:"rx_err_packets"`
	RxFcsPackets       *int64 `json:"rx_fcs_packets"`
	RxFragmentsPackets *int64 `json:"rx_fragments_packets"`
	RxGoodBytes        *int64 `json:"rx_good_bytes"`
	RxGoodPackets      *int64 `json:"rx_good_packets"`
	RxJabberPackets    *int64 `json:"rx_jabber_packets"`
	RxMulticastPackets *int64 `json:"rx_multicast_packets"`
	RxOversizePackets  *int64 `json:"rx_oversize_packets"`
	RxPacketsRate      *int64 `json:"rx_packets_rate"`
	RxPause            *int64 `json:"rx_pause"`
	RxUndersizePackets *int64 `json:"rx_undersize_packets"`
	RxUnicastPackets   *int64 `json:"rx_unicast_packets"`

	TxBroadcastPackets *int64 `json:"tx_broadcast_packets"`
	TxBytes            *int64 `json:"tx_bytes"`
	TxBytesRate        *int64 `json:"tx_bytes_rate"`
	TxCollisions       *int64 `json:"tx_collisions"`
	TxDeferred         *int64 `json:"tx_deferred"`
	TxExcessive        *int64 `json:"tx_excessive"`
	TxFcs              *int64 `json:"tx_fcs"`
	TxLate             *int64 `json:"tx_late"`
	TxMulticastPackets *int64 `json:"tx_multicast_packets"`
	TxMultiple         *int64 `json:"tx_multiple"`
	TxPackets          *int64 `json:"tx_packets"`
	TxPacketsRate      *int64 `json:"tx_packets_rate"`
	TxPause            *int64 `json:"tx_pause"`
	TxSingle           *int64 `json:"tx_single"`
	TxUnicastPackets   *int64 `json:"tx_unicast_packets"`
}

// GetMetricsSystem http://mafreebox.freebox.fr/api/v5/system/
func (f *FreeboxConnection) GetMetricsSystem() (*MetricsFreeboxSystem, error) {
	res := new(MetricsFreeboxSystem)
	err := f.get("system/", res)
	return res, err
}

// GetMetricsConnection http://mafreebox.freebox.fr/api/v5/connection/
func (f *FreeboxConnection) GetMetricsConnection() (*MetricsFreeboxConnectionAll, error) {
	result := new(MetricsFreeboxConnectionAll)
	if err := f.get("connection/", result); err != nil {
		return nil, err
	}

	switch result.Media {
	case "xdsl":
		// http://mafreebox.freebox.fr/api/v5/connection/xdsl/
		// https://dev.freebox.fr/sdk/os/connection/#get-the-current-xdsl-infos
		xdsl := new(MetricsFreeboxConnectionXdsl)
		if err := f.get("connection/xdsl/", xdsl); err != nil {
			return nil, err
		}
		result.Xdsl = xdsl
	case "ftth":
		// http://mafreebox.freebox.fr/api/v5/connection/ftth/
		// https://dev.freebox.fr/sdk/os/connection/#get-the-current-ftth-status
		ftth := new(MetricsFreeboxConnectionFtth)
		if err := f.get("connection/ftth/", ftth); err != nil {
			return nil, err
		}
		result.Ftth = ftth
	}

	return result, nil
}

// GetMetricsSwitch http://mafreebox.freebox.fr/api/v5/switch/status/
func (f *FreeboxConnection) GetMetricsSwitch() (*MetricsFreeboxSwitch, error) {
	res := new(MetricsFreeboxSwitch)
	err := f.get("switch/status/", &res.Ports)
	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	wg.Add(len(res.Ports))

	for _, port := range res.Ports {
		go func(port *MetricsFreeboxSwitchStatus) {
			defer wg.Done()
			stats := new(MetricsFreeboxSwitchPortStats)
			// http://mafreebox.freebox.fr/api/v5/switch/port/1/stats
			err := f.get(fmt.Sprintf("switch/port/%d/stats", port.ID), stats)
			if err != nil {
				log.Warning.Println("Could not get status of port", port.ID)
				return
			}
			port.Stats = stats
		}(port)
	}

	wg.Wait()
	return res, err
}
