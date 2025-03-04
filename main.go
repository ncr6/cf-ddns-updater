package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/bobesa/go-domain-util/domainutil"
	"github.com/cloudflare/cloudflare-go"
	"github.com/rdegges/go-ipify"
	"github.com/zchee/go-xdgbasedir"
)

type Config struct {
	FQDN     []string
	APIToken string
}

var config Config
var appName = "cf-ddns-updater"
var updateErrors = 0

func updateDNSRecord(fqdnArr []string, recordType string, ip string, checkMode bool) error {
	api, err := cloudflare.NewWithAPIToken(config.APIToken)
	if err != nil {
		updateErrors++
	}

	for _, fqdn := range fqdnArr {
		log.Println("Processing FQDN:", fqdn)

		domain := domainutil.Domain(fqdn)
		zoneID, err := api.ZoneIDByName(domain)
		if err != nil {
			log.Println(err)
			updateErrors++
		}

		fqdnRecord := cloudflare.DNSRecord{Name: fqdn}
		records, err := api.DNSRecords(context.Background(), zoneID, fqdnRecord)
		if err != nil {
			log.Println(err)
			updateErrors++
		}

		for _, r := range records {
			if r.Type == recordType {
				if r.Content == ip {
					log.Printf("%s points to current IP address, no change is needed.", r.Name)
					continue
				}
				log.Printf("%s points to %s, the record will be updated.", r.Name, r.Content)

				if checkMode {
					log.Println("Check mode is active, no changes will be made.")
				} else {
					log.Println("Setting", fqdn, "=>", ip, "...")
					r.Content = ip
					err := api.UpdateDNSRecord(context.Background(), zoneID, r.ID, r)
					if err != nil {
						log.Println(err)
						updateErrors++
					}
					log.Println("Success!")
				}
			}
		}
	}

	log.Println(updateErrors, "errors occurred.")

	if updateErrors > 0 {
		return cloudflare.Error{}
	} else {
		return nil
	}
}

func loadConfig(fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(&config)
}

func main() {
	configFile := filepath.FromSlash(xdgbasedir.ConfigHome() + "/" + appName + "/config.json")
	cf := flag.String("c", configFile, "Config file")
	check_mode := flag.Bool("n", false, "Check mode (dry run)")
	flag.Parse()
	err := loadConfig(*cf)
	if err != nil {
		log.Fatal(err)
	}

	ipv4, err := ipify.GetIp()
	if err != nil {
		log.Fatal("Couldn’t determine the current IP address:", err)
	} else {
		log.Println("Current IP address is:", ipv4)
	}

	err = updateDNSRecord(config.FQDN, "A", ipv4, *check_mode)
	if err != nil {
		log.Println("Unable to update all DNS records.")
	}
}
