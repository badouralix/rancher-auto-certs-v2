package main

import (
	"time"

	"github.com/go-acme/lego/v4/log"
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
	ticker := time.NewTicker(24 * time.Hour)

	// See https://stackoverflow.com/questions/32705582/how-to-get-time-tick-to-tick-immediately
	for ; true; <-ticker.C {
		// TODO: rewrite this to handle errors properly
		runAll(lm, cm)
	}
}

func runAll(lm *letsencryptManager, cm *certificateManager) error {
	// Load config from file
	log.Infof("main: Reading config")
	config, err := loadConfig("config/config.yml")
	if err != nil {
		log.Warnf("%s", err)
		return err
	}

	// Read existing rancher certificates
	log.Infof("main: Reading existing rancher certificates")
	err = cm.updateLocalCertCache()
	if err != nil {
		log.Warnf("%s", err)
		return err
	}

	// Read cert by cert
	for _, c := range config.Certs {
		// Build cert config
		cc := mergeDefaultConfig(c, config.Default)

		// Cannot run this in a goroutine as cm.cache is not thread-safe
		runCert(lm, cm, cc)
	}

	// Clear rancher cache
	cm.clearLocalCertCache()

	// Wrap up
	log.Infof("main: I'm done")
	return nil
}

func runCert(lm *letsencryptManager, cm *certificateManager, cc certConfig) {
	// Look for certificates that do not exist yet or that will expire in less than 30 days
	if _, ok := cm.cache[cc.Name]; ok {
		t, err := time.Parse("Mon Jan 02 15:04:05 MST 2006", cm.cache[cc.Name].ExpiresAt)

		if err != nil {
			log.Warnf("main: Failed to parse expire date, renewing %s", cc.Name)
		} else if t.After(time.Now().AddDate(0, 0, 30)) {
			log.Infof("main: Skipping %s", cc.Name)
			return
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
		return
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
	if cert, ok := cm.cache[cc.Name]; ok {
		_, err := cm.updateRancherCert(cert, rancherCertificate)
		if err != nil {
			log.Warnf("%s", err)
			return
		}
	} else {
		_, err := cm.createRancherCert(rancherCertificate)
		if err != nil {
			log.Warnf("%s", err)
			return
		}
	}
}
