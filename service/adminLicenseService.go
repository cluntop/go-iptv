package service

import (
	"encoding/json"
	"errors"
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/until"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func Proxy(params url.Values) dto.ReturnJsonDto {
	cfg := dao.GetConfig()
	scheme := params.Get("scheme")

	if scheme == "" || (scheme != "http" && scheme != "https") {
		return dto.ReturnJsonDto{Code: 0, Msg: "中转协议不正确", Type: "danger"}
	}
	port := params.Get("port")
	proxy := params.Get("proxy")
	pAddr := params.Get("pAddr")

	if proxy == "1" || proxy == "true" || proxy == "on" {
		if port == "" {
			return dto.ReturnJsonDto{Code: 0, Msg: "中转端口不能为空", Type: "danger"}
		}

		if pAddr == "" {
			return dto.ReturnJsonDto{Code: 0, Msg: "中转地址不能为空", Type: "danger"}
		}
		pAddr = strings.TrimPrefix(strings.TrimPrefix(pAddr, "https://"), "http://")

		portInt64, err := strconv.ParseInt(port, 10, 64)
		if err != nil {
			return dto.ReturnJsonDto{Code: 0, Msg: "中转端口为数字", Type: "danger"}
		}
		if portInt64 < 80 || portInt64 > 65535 {
			return dto.ReturnJsonDto{Code: 0, Msg: "中转端口范围为80-65535", Type: "danger"}
		}
		cfg.Proxy.Scheme = scheme
		cfg.Proxy.Port = portInt64
		cfg.Proxy.PAddr = pAddr

		res, err := dao.WS.SendWS(dao.Request{Action: "startProxy"})
		if err != nil {
			return startError(cfg, err)
		} else {
			if res.Code == 1 {
				time.Sleep(1 * time.Second)
				res, err := dao.WS.SendWS(dao.Request{Action: "getProxyStatus"})
				if err != nil {
					return startError(cfg, err)
				} else {
					var status bool
					if err := json.Unmarshal(res.Data, &status); err != nil {
						return startError(cfg, err)
					}
					if !status {
						return startError(cfg, err)
					}
				}

				tmpRes := until.GetUrlData(scheme + "://" + pAddr + ":" + port + "/status")
				if tmpRes == "ok" {
					cfg.Proxy.Status = 1
					dao.SetConfig(cfg)
					go until.CleanAutoCacheAll() // 清理缓存
					return dto.ReturnJsonDto{Code: 1, Msg: "启动成功，可以到频道分组管理中开启中转啦", Type: "success"}
				} else {
					tmpRes := until.GetUrlData("http://127.0.0.1:8080/status")
					if tmpRes == "ok" {
						cfg.Proxy.Status = 1
						dao.SetConfig(cfg)
						go until.CleanAutoCacheAll() // 清理缓存
						return dto.ReturnJsonDto{Code: 0, Msg: "启动成功，容器无法访问中转地址 " + scheme + "://" + pAddr + ":" + port + " ,若使用的IPv6地址请访问" + scheme + "://" + pAddr + ":" + port + "/status  返回ok即可忽略该提示", Type: "danger"}
					}
					return startError(cfg, errors.New("中转地址 "+scheme+"://"+pAddr+":"+port+" 无法访问,请重新配置地址或端口"))
				}
			} else {
				go until.CleanAutoCacheAll()
				return startError(cfg, errors.New(res.Msg))
			}
		}
	} else {
		cfg.Proxy.Status = 0

		dao.SetConfig(cfg)
		dao.WS.SendWS(dao.Request{Action: "stopProxy"})
		go until.CleanAutoCacheAll()
		return dto.ReturnJsonDto{Code: 1, Msg: "停止成功", Type: "success"}
	}

}

func startError(cfg *dto.Config, err error) dto.ReturnJsonDto {
	cfg.Proxy.Status = 0

	dao.SetConfig(cfg)
	dao.WS.SendWS(dao.Request{Action: "stopProxy"})
	return dto.ReturnJsonDto{Code: 2, Msg: "启动失败: " + err.Error(), Type: "danger"}
}

func ResEng() dto.ReturnJsonDto {
	if dao.WS.RestartLic() {
		return dto.ReturnJsonDto{Code: 1, Msg: "重启成功", Type: "success"}
	}
	return dto.ReturnJsonDto{Code: 0, Msg: "重启失败", Type: "danger"}
}

func AutoRes(params url.Values) dto.ReturnJsonDto {
	autoRes := params.Get("autoRes")
	cfg := dao.GetConfig()
	if autoRes == "1" || autoRes == "true" || autoRes == "on" {
		cfg.Resolution.Auto = 1
	} else {
		cfg.Resolution.Auto = 0
	}
	dao.SetConfig(cfg)
	return dto.ReturnJsonDto{Code: 1, Msg: "设置成功", Type: "success"}
}

func DisCh(params url.Values) dto.ReturnJsonDto {
	disCh := params.Get("disCh")
	cfg := dao.GetConfig()
	if disCh == "1" || disCh == "true" || disCh == "on" {
		cfg.Resolution.DisCh = 1
	} else {
		cfg.Resolution.DisCh = 0
	}
	dao.SetConfig(cfg)
	return dto.ReturnJsonDto{Code: 1, Msg: "设置成功", Type: "success"}
}

func EpgFuzz(params url.Values) dto.ReturnJsonDto {
	epgFuzz := params.Get("epgFuzz")
	cfg := dao.GetConfig()
	if epgFuzz == "1" || epgFuzz == "true" || epgFuzz == "on" {
		cfg.Epg.Fuzz = 1
	} else {
		cfg.Epg.Fuzz = 0
	}
	dao.SetConfig(cfg)
	return dto.ReturnJsonDto{Code: 1, Msg: "设置成功", Type: "success"}
}

func AggStatus(params url.Values) dto.ReturnJsonDto {
	aggStatus := params.Get("aggStatus")
	cfg := dao.GetConfig()
	if aggStatus == "1" || aggStatus == "true" || aggStatus == "on" {
		cfg.Aggregation.Status = 1
	} else {
		cfg.Aggregation.Status = 0
	}
	dao.SetConfig(cfg)
	return dto.ReturnJsonDto{Code: 1, Msg: "设置成功", Type: "success"}
}

func Dispay(params url.Values) dto.ReturnJsonDto {
	dispay := params.Get("dispay")
	cfg := dao.GetConfig()
	if dispay == "1" || dispay == "true" || dispay == "on" {
		_, err := until.CheckLicVer("v1.5.19")
		if err != nil {
			return dto.ReturnJsonDto{Code: 0, Msg: err.Error(), Type: "danger"}
		}
		cfg.System.DisPay = 1
	} else {
		cfg.System.DisPay = 0
	}
	dao.SetConfig(cfg)
	return dto.ReturnJsonDto{Code: 5, Msg: "设置成功,刷新页面生效", Type: "success"}
}

func ShortURL(params url.Values) dto.ReturnJsonDto {
	shortURL := params.Get("shortURL")
	cfg := dao.GetConfig()
	if shortURL == "1" || shortURL == "true" || shortURL == "on" {
		_, err := until.CheckLicVer("v1.5.19")
		if err != nil {
			return dto.ReturnJsonDto{Code: 0, Msg: err.Error(), Type: "danger"}
		}
		cfg.System.ShortURL = 1
	} else {
		cfg.System.ShortURL = 0
	}
	dao.SetConfig(cfg)
	return dto.ReturnJsonDto{Code: 1, Msg: "设置成功", Type: "success"}
}

func StartPHP(params url.Values) dto.ReturnJsonDto {
	startPHP := params.Get("startPHP")
	cfg := dao.GetConfig()
	if startPHP == "1" || startPHP == "true" || startPHP == "on" {
		_, err := dao.WS.SendWS(dao.Request{Action: "startPHP"})
		if err != nil {
			cfg.PHPWeb = 0
			dao.SetConfig(cfg)
			return dto.ReturnJsonDto{Code: 0, Msg: err.Error(), Type: "danger"}
		}
		cfg.PHPWeb = 1
		dao.SetConfig(cfg)
	} else {
		_, err := dao.WS.SendWS(dao.Request{Action: "stopPHP"})
		if err != nil {
			cfg.PHPWeb = 0
			dao.SetConfig(cfg)
			return dto.ReturnJsonDto{Code: 0, Msg: err.Error(), Type: "danger"}
		}
		cfg.PHPWeb = 0
		dao.SetConfig(cfg)
	}
	return dto.ReturnJsonDto{Code: 1, Msg: "设置成功", Type: "success"}
}
