package main

import (
	"net"
	"sync"

	"github.com/cdreier/golang-snippets/snippets"
	"github.com/fasibio/funk_agent/geoipdbupdater"
	"github.com/fasibio/funk_agent/logger"
	"github.com/oschwald/geoip2-golang"
)

var geoIPDatabase *geoip2.Reader
var mutex sync.Mutex

func InitGeoIP() GeoReader {
	snippets.EnsureDir("./tmpassets")
	snippets.EnsureDir("./tmpassets/geoip")
	updateInfo := make(chan string, 2)
	geoipdbupdater.NewGEOIPUpdateTicker(updateInfo)

	go func() {
		for newDBPath := range updateInfo {
			mutex.Lock()
			tmpGeoIPDatabase, err := geoip2.Open(newDBPath)
			if err != nil {
				logger.Get().Error("cannot open new database file: ", err.Error(), ", no updated Database is loaded!")
			} else {
				geoIPDatabase = tmpGeoIPDatabase
				geoipdbupdater.CleanUpOldMaxmindDBs(newDBPath, geoipdbupdater.DefaultGeoIPurlParams)
				logger.Get().Info("Updated newer geoIP db table")
			}
			mutex.Unlock()
		}
	}()
	return &GeoDataReader{}
}

type GeoReader interface {
	GetGeoDataByIP(IPaddress string) (*geoip2.City, error)
}

type GeoDataReader struct {
}

func (g *GeoDataReader) GetGeoDataByIP(IPaddress string) (*geoip2.City, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return geoIPDatabase.City(net.ParseIP(IPaddress))
}
