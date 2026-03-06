package service

import (
	"go-iptv/assets"
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/until"
	"net/url"
	"strconv"
)

var BuildStatus int64 = 0

func SetMyTVAppInfo(params url.Values) dto.ReturnJsonDto {
	appServerUrl := params.Get("serverUrl")
	appVersion := params.Get("app_version")
	upBody := params.Get("up_body")

	if appVersion == "" || appServerUrl == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "参数错误", Type: "danger"}
	}

	appVersionInt, err := strconv.ParseInt(appVersion, 10, 64)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "版本号为纯数字", Type: "danger"}
	}
	appVersion = strconv.FormatInt(appVersionInt, 10)

	if appVersionInt <= 0 || appVersionInt > 999 {
		return dto.ReturnJsonDto{Code: 0, Msg: "版本号范围为1-999的纯数字", Type: "danger"}
	}

	cfg := dao.GetConfig()

	if cfg.MyTV.BaseVersion == "" {
		cfg.MyTV.BaseVersion = string(assets.MyTVApkVersion)
	}

	if cfg.MyTV.BaseVersion == string(assets.MyTVApkVersion) && cfg.MyTV.Version == appVersion {
		return dto.ReturnJsonDto{Code: 0, Msg: "版本号不能相同", Type: "danger"}
	}
	cfg.MyTV.BaseVersion = string(assets.MyTVApkVersion)

	cfg.MyTV.Version = appVersion

	if cfg.ServerUrl != appServerUrl {
		cfg.ServerUrl = appServerUrl
	}

	cfg.MyTV.Update = upBody

	dao.SetConfig(cfg)
	return dto.ReturnJsonDto{Code: 1, Msg: "设置成功", Type: "success"}
}

func GetMyTVBuildStatus() dto.ReturnJsonDto {
	cfg := dao.GetConfig()
	return dto.ReturnJsonDto{Code: 1, Msg: "APK编译完成", Type: "success", Data: map[string]interface{}{"size": until.GetFileSize("/config/app/清和IPTV-mytv.apk"), "version": cfg.MyTV.Version, "url": "/app/清和IPTV-mytv.apk", "name": "清和IPTV-mytv-1.2.0." + cfg.MyTV.Version + ".apk"}}
}

func MytvReleases() dto.MyTvDto {
	cfg := dao.GetConfig()
	return dto.MyTvDto{
		Version:     "1.2.0." + cfg.MyTV.Version,
		DownloadUrl: cfg.ServerUrl + "/app/清和IPTV-mytv.apk",
		UpdateMsg:   cfg.MyTV.Update,
	}
}
