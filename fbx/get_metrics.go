package fbx

// MetricsFreeboxSystem https://dev.freebox.fr/sdk/os/system/
type MetricsFreeboxSystem struct {
	FirmwareVersion  string `json:"firmware_version"`
	Mac              string `json:"mac"`
	Serial           string `json:"serial"`
	Uptime           string `json:"uptime"`
	UptimeValue      *int64 `json:"uptime_val"`
	BoardName        string `json:"board_name"`
	TempCPUM         *int64 `json:"temp_cpum"`
	TempSW           *int64 `json:"temp_sw"`
	TempCPUB         *int64 `json:"temp_cpub"`
	FanRpm           *int64 `json:"fan_rpm"`
	BoxAuthenticated *bool  `json:"box_authenticated"`
	DiskStatus       string `json:"disk_status"`
	BoxFlavor        string `json:"box_flavor"`
	UserMainStorage  string `json:"user_main_storage"`
}

// MetricsFreeboxConnection https://dev.freebox.fr/sdk/os/connection/
type MetricsFreeboxConnection struct {
	State        string `json:"state"`
	Type         string `json:"type"`
	Media        string `json:"media"`
	IPv4         string `json:"ipv4"`
	IPv6         string `json:"ipv6"`
	RateUp       *int64 `json:"rate_up"`
	RateDown     *int64 `json:"rate_down"`
	BandwithUp   *int64 `json:"bandwith_up"`
	BandwithDown *int64 `json:"bandwith_down"`
	BytesUp      *int64 `json:"bytes_up"`
	BytesDown    *int64 `json:"bytes_down"`
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

// GetMetricsSystem http://mafreebox.freebox.fr/api/v5/system/
func (f *FreeboxConnection) GetMetricsSystem() (*MetricsFreeboxSystem, error) {
	res := new(MetricsFreeboxSystem)
	err := f.get(res, "system")
	return res, err
}

// GetMetricsConnection http://mafreebox.freebox.fr/api/v5/connection/
func (f *FreeboxConnection) GetMetricsConnection() (*MetricsFreeboxConnectionAll, error) {
	result := new(MetricsFreeboxConnectionAll)
	if err := f.get(result, "connection"); err != nil {
		return nil, err
	}

	switch result.Media {
	case "xdsl":
		// http://mafreebox.freebox.fr/api/v5/connection/xdsl/
		// https://dev.freebox.fr/sdk/os/connection/#get-the-current-xdsl-infos
		xdsl := new(MetricsFreeboxConnectionXdsl)
		if err := f.get(xdsl, "connection/xdsl"); err != nil {
			return nil, err
		}
		result.Xdsl = xdsl
	case "ftth":
		// http://mafreebox.freebox.fr/api/v5/connection/ftth/
		// https://dev.freebox.fr/sdk/os/connection/#get-the-current-ftth-status
		ftth := new(MetricsFreeboxConnectionFtth)
		if err := f.get(ftth, "connection/ftth"); err != nil {
			return nil, err
		}
		result.Ftth = ftth
	}

	return result, nil
}
