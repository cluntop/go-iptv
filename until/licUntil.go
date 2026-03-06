package until

import (
	"go-iptv/dao"
	"strings"
)

func InitProxy() {
	var scheme, pAddr string
	var port int64

	cfg := dao.GetConfig()
	if cfg.Proxy.PAddr == "" {
		scheme, pAddr, port = ParseURL(cfg.ServerUrl)
	} else {
		scheme = cfg.Proxy.Scheme
		pAddr = cfg.Proxy.PAddr
		port = cfg.Proxy.Port
	}

	if scheme == "" || scheme == "http" {
		scheme = "http"
		if port == 0 {
			port = 80
		}
	} else {
		scheme = "https"
		if port == 0 {
			port = 443
		}
	}

	pAddr = strings.TrimPrefix(strings.TrimPrefix(pAddr, "https://"), "http://")

	if scheme != cfg.Proxy.Scheme || pAddr != cfg.Proxy.PAddr || port != cfg.Proxy.Port {
		cfg.Proxy.Scheme = scheme
		cfg.Proxy.PAddr = pAddr
		cfg.Proxy.Port = port
		dao.SetConfig(cfg)
	}
}

func CheckLicVer(latest string) (bool, error) {
	return true, nil
}
