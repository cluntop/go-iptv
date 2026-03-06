package dto

type AdminLicenseDto struct {
	LoginUser   string `json:"loginuser"`
	Title       string `json:"title"`
	Scheme      string `json:"scheme"`
	Proxy       int64  `json:"proxy"`
	Port        int64  `json:"port"`
	ProxyAddr   string `json:"proxy_addr"`
	Status      int64  `json:"status"`
	Online      int64  `json:"online"`
	Version     string `json:"version"`
	AutoRes     int64  `json:"auto_res"`
	DisCh       int64  `json:"dis_ch"`
	EpgFuzz     int64  `json:"epg_fuzz"`
	Aggregation int64  `json:"aggregation"`
	ShortURL    int64  `json:"short_url"`
	StartPHP    int64  `json:"start_php"`
	DisPay      int64  `json:"dis_pay"`
}
