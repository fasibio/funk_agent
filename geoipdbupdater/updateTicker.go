package geoipdbupdater

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fasibio/funk_agent/logger"
	"github.com/mholt/archiver"
)

type geoIPurlParams struct {
	AssetPath     string
	MaxmindFolder string
}

var DefaultGeoIPurlParams = geoIPurlParams{
	AssetPath:     "./tmpassets/geoip",
	MaxmindFolder: "geoip",
}

const DownloadURL = "https://geolite.maxmind.com/download/geoip/database/GeoLite2-City.tar.gz"

func NewGEOIPUpdateTicker(geoIPReadyToUpdatePath chan string) error {
	defaultPath := path.Join(DefaultGeoIPurlParams.AssetPath, "GeoLite2-City.mmdb")
	stats, err := os.Stat(defaultPath)
	if err == nil {
		if stats.ModTime().After(time.Now().Add(24 * time.Hour)) {
			logger.Get().Debug("GeoIpUpdateTicker: File is older than one days so update")
			if updatedPath, err := downloadAndExtract(); err == nil {
				geoIPReadyToUpdatePath <- updatedPath
			} else {
				logger.Get().Error(err)
				return err
			}
		} else {
			geoIPReadyToUpdatePath <- defaultPath
		}
	} else {
		if updatedPath, err := downloadAndExtract(); err == nil {
			geoIPReadyToUpdatePath <- updatedPath
		} else {
			logger.Get().Error(err)
			return err
		}
	}

	ticker := time.NewTicker(27 * time.Hour)
	go func() {
		for range ticker.C {
			logger.Get().Infow("Start Downloading Maxmind DB")
			if updatedPath, err := downloadAndExtract(); err == nil {
				geoIPReadyToUpdatePath <- updatedPath
			} else {
				logger.Get().Error(err)
			}
		}
	}()
	return nil
}

func extractFilename(contentDisposition string) string {
	dlFilename := strings.Split(contentDisposition, "filename=")[1]
	dlFilename = strings.Split(dlFilename, ".tar.gz")[0]
	return dlFilename
}

func CleanUpOldMaxmindDBs(activeDBPath string, params geoIPurlParams) {
	activeDBFilename := filepath.Base(activeDBPath)
	filepath.Walk(params.AssetPath, func(path string, info os.FileInfo, err error) error {
		filename := filepath.Base(path)
		if strings.HasSuffix(path, ".dat") && filename != activeDBFilename {
			os.Remove(path)
		}
		return nil
	})
}

func downloadNewGeoIPInfos() (string, error) {
	resp, err := http.Get(DownloadURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	dlFilename := extractFilename(resp.Header.Get("Content-Disposition"))
	dlPath := "/tmp/" + dlFilename + ".tar.gz"

	out, err := os.Create(dlPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return dlPath, err
}

func downloadAndExtract() (string, error) {
	dlPath, err := downloadNewGeoIPInfos()
	// var err error
	// dlPath := "/tmp/GeoIP-132eu_20190319.tar.gz"
	if err != nil {
		return "", err
	}

	return extractDownloadedGeoIPDB(dlPath, DefaultGeoIPurlParams)
}

func extractDownloadedGeoIPDB(dlPath string, params geoIPurlParams) (string, error) {
	var destinationFilename = params.AssetPath + "/GeoLite2-City.mmdb"

	tz := archiver.NewTarGz()
	err := tz.Walk(dlPath, func(f archiver.File) error {
		if strings.HasSuffix(f.Name(), ".mmdb") {

			logger.Get().Debug("found database file in archive: ", f.Name())

			targetFile, err := os.OpenFile(destinationFilename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			defer targetFile.Close()
			defer f.Close()
			_, err = io.Copy(targetFile, f)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return destinationFilename, nil
}
