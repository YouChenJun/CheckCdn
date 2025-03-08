// @Author Chen_dark
// @Date 2024/9/18 16:37:00
// @Desc
package cmd

import (
	"github.com/YouChenJun/CheckCdn/client"
	"github.com/YouChenJun/CheckCdn/config"
)

/**
 * @author ChenDark
 * @description //TODO
 * @date 16:38 2024/9/18
 * @param
 * @return
 **/
type CdnCheck struct {
	client *client.Client
}

func New(config *config.BATconfig) CdnCheck {
	return CdnCheck{client: client.New(config)}
}

func (c *CdnCheck) Check(ip string) config.Result {
	result := c.client.Check(ip)
	return result
}
