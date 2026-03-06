package html

import (
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/until"

	"github.com/gin-gonic/gin"
)

func License(c *gin.Context) {
	username, ok := until.GetAuthName(c)
	if !ok {
		c.JSON(200, dto.NewAdminRedirectDto())
		return
	}
	var pageData = dto.AdminLicenseDto{
		LoginUser: username,
		Title:     "进阶功能",
	}

	cfg := dao.GetConfig()
	pageData.Proxy = cfg.Proxy.Status

	if pageData.Proxy == 1 {
		pageData.Aggregation = cfg.Aggregation.Status
	} else {
		pageData.Aggregation = 0
	}

	pageData.ProxyAddr = cfg.Proxy.PAddr
	pageData.Scheme = cfg.Proxy.Scheme
	pageData.Port = cfg.Proxy.Port

	pageData.AutoRes = cfg.Resolution.Auto
	pageData.DisCh = cfg.Resolution.DisCh
	pageData.EpgFuzz = cfg.Epg.Fuzz
	pageData.ShortURL = cfg.System.ShortURL
	pageData.DisPay = cfg.System.DisPay

	c.HTML(200, "admin_license.html", pageData)
}
