package ipdb

import (
	"fmt"
	"net"
)

func GetCity(input net.IP) string {
	db, err := NewCity("")
	if err != nil {
		fmt.Println(err)
		return ""
	}
	cityInfo, err := db.FindInfo(input.String(), "CN")
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return cityInfo.RegionName + cityInfo.CityName + cityInfo.IspDomain + cityInfo.IDC + cityInfo.BaseStation
}
