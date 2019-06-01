package main

import (
	"fmt"
	"time"

	"github.com/go-acme/lego/platform/config/env"
	rancher "github.com/rancher/go-rancher/client"
)

// Manager for Rancher certificates
type certificateManager struct {
	cache  map[string]*rancher.Certificate
	client rancher.CertificateOperations
}

func newCertificateManagerFromEnvvar() (*certificateManager, error) {
	// Read config from environment variables
	values, err := env.Get("CATTLE_URL", "CATTLE_ACCESS_KEY", "CATTLE_SECRET_KEY")
	if err != nil {
		return nil, fmt.Errorf("rancher: %v", err)
	}

	// Build rancher options
	rancherOpts := rancher.ClientOpts{
		Url:       values["CATTLE_URL"],
		AccessKey: values["CATTLE_ACCESS_KEY"],
		SecretKey: values["CATTLE_SECRET_KEY"],
		Timeout:   env.GetOrDefaultSecond("CATTLE_TIMEOUT", 60*time.Second),
	}

	// Build rancher client
	rancherClient, err := rancher.NewRancherClient(&rancherOpts)
	if err != nil {
		return nil, fmt.Errorf("rancher: %v", err)
	}

	// Build the certificate manager
	return &certificateManager{
		cache:  make(map[string]*rancher.Certificate),
		client: rancherClient.Certificate,
	}, nil
}

func (cm *certificateManager) clearLocalCertCache() {
	// Old cache will be garbage collected
	cm.cache = make(map[string]*rancher.Certificate)
}

func (cm *certificateManager) updateLocalCertCache() error {
	// TODO: iterate with Next
	certificateCollection, err := cm.client.List(&rancher.ListOpts{})
	if err != nil {
		return fmt.Errorf("rancher: %v", err)
	}

	for _, certificate := range certificateCollection.Data {
		cm.cache[certificate.Name] = &certificate
	}

	return nil
}

func (cm *certificateManager) updateRancherCert(existing *rancher.Certificate, updates interface{}) (*rancher.Certificate, error) {
	cert, err := cm.client.Update(existing, updates)
	if err != nil {
		return nil, fmt.Errorf("rancher: %v", err)
	}
	return cert, nil
}

func (cm *certificateManager) createRancherCert(new *rancher.Certificate) (*rancher.Certificate, error) {
	cert, err := cm.client.Create(new)
	if err != nil {
		return nil, fmt.Errorf("rancher: %v", err)
	}
	return cert, nil
}
