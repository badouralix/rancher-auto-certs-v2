package main

import (
	"time"

	"github.com/go-acme/lego/log"
	rancher "github.com/rancher/go-rancher/client"
)

func main() {
	// Setup let's encrypt manager once
	lm, err := newLEManager()
	if err != nil {
		log.Fatal(err)
	}

	// Setup rancher manager once
	cm, err := newCertificateManagerFromEnvvar()
	if err != nil {
		log.Fatal(err)
	}

	// Run repetitively
	ticker := time.NewTicker(24 * time.Second)

	// See https://stackoverflow.com/questions/32705582/how-to-get-time-tick-to-tick-immediately
	for ; true; <-ticker.C {
		// TODO: rewrite this to handle errors properly
		go run(lm, cm)
	}
}

func run(lm *letsencryptManager, cm *certificateManager) error {
	// Load config from file
	log.Infof("main: Reading config")
	config, err := loadConfig("config/config.yml")
	if err != nil {
		log.Warnf("%s", err)
		return err
	}

	// Read existing rancher certificates
	log.Infof("main: Reading existing rancher certificates")
	existingCerts, err := cm.listRancherCerts()
	if err != nil {
		log.Warnf("%s", err)
		return err
	}

	// Read cert by cert
	for _, c := range config.Certs {
		// Build cert config
		cc := mergeDefaultConfig(c, config.Default)

		// Look for certificates that do not exist yet or that will expire in less than 30 days
		// TODO: iterate with Next
		if _, ok := existingCerts[cc.Name]; ok {
			t, err := time.Parse("Mon Jan 02 15:04:05 MST 2006", existingCerts[cc.Name].ExpiresAt)

			if err != nil {
				log.Warnf("main: Failed to parse expire date, renewing %s", cc.Name)
			} else if t.After(time.Now().AddDate(0, 0, 30)) {
				log.Infof("main: Skipping %s", cc.Name)
				continue
			} else {
				log.Infof("main: Renewing %s", cc.Name)
			}
		} else {
			log.Infof("main: Creating %s", cc.Name)
		}

		// Request new certificate
		legoCertificate, err := lm.GenCertificate(&cc)
		if err != nil {
			log.Warnf("%s", err)
			continue
		}

		// Build rancher certificate
		rancherCertificate := &rancher.Certificate{
			Name:        cc.Name,
			Description: cc.Description,
			Cert:        string(legoCertificate.Certificate),
			CertChain:   string(legoCertificate.IssuerCertificate),
			Key:         string(legoCertificate.PrivateKey),
		}

		// Upload certificate
		log.Infof("main: Uploading %s", cc.Name)
		if cert, ok := existingCerts[cc.Name]; ok {
			_, err := cm.updateRancherCert(cert, rancherCertificate)
			if err != nil {
				log.Warnf("%s", err)
				continue
			}
		} else {
			_, err := cm.createRancherCert(rancherCertificate)
			if err != nil {
				log.Warnf("%s", err)
				continue
			}
		}
	}

	// Wrap up
	log.Infof("main: I'm done")
	return nil
}
