package main

import (
	"crypto"
	"fmt"
	"os"
	"sync"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/log"
	"github.com/go-acme/lego/v4/providers/dns"
	"github.com/go-acme/lego/v4/providers/http/webroot"
	"github.com/go-acme/lego/v4/registration"
)

// ACMEUser a user or account type that implements registration.User
type ACMEUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

// GetEmail returns the email address of the acme user
func (u *ACMEUser) GetEmail() string {
	return u.Email
}

// GetRegistration returns the registration resource of the acme user
func (u *ACMEUser) GetRegistration() *registration.Resource {
	return u.Registration
}

// GetPrivateKey returns the private key of the acme user
func (u *ACMEUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// ACMEManager is a cache for acme clients and their users
type ACMEManager struct {
	clients     map[string]*lego.Client
	environment sync.Mutex
}

func newACMEManager() (*ACMEManager, error) {
	return &ACMEManager{
		clients: make(map[string]*lego.Client),
	}, nil
}

// GenCertificate generates one certificate for a given config
func (am *ACMEManager) GenCertificate(cc *certConfig) (*certificate.Resource, error) {
	client, err := am.GetClient(cc)
	if err != nil {
		return nil, err
	}

	request := certificate.ObtainRequest{
		Domains: cc.Domains,
		Bundle:  true,
	}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return nil, err
	}

	// Each certificate comes back with the cert bytes, the bytes of the client's
	// private key, and a certificate URL

	return certificates, nil
}

// GetClient returns an acme client for a gieven config
// The client is created if it does not exist yet in the manager cache
func (am *ACMEManager) GetClient(cc *certConfig) (*lego.Client, error) {
	// Try to retrieve client from cache
	if client, ok := am.clients[cc.Name]; ok {
		return client, nil
	}

	// If not ok, no client has been generated for this cert yet

	// Create user: new accounts need an email and private key to start
	privateKey, err := cc.getPrivateKey()
	if err != nil {
		return nil, err
	}

	user := ACMEUser{
		Email: cc.AccountEmail,
		key:   privateKey,
	}

	// Create config for user
	config := lego.NewConfig(&user)

	// The CA URL can be overridden for development purpose
	config.CADirURL = cc.CA

	// The default key type is RSA2048, but we provide the ability to override it
	switch cc.KeyType {
	case "EC256":
		config.Certificate.KeyType = certcrypto.EC256
	case "EC384":
		config.Certificate.KeyType = certcrypto.EC384
	case "RSA2048":
		config.Certificate.KeyType = certcrypto.RSA2048
	case "RSA4096":
		config.Certificate.KeyType = certcrypto.RSA4096
	case "RSA8192":
		config.Certificate.KeyType = certcrypto.RSA8192
	default:
		return nil, fmt.Errorf("letsencrypt: unsupported key type %s", cc.KeyType)
	}

	// A client facilitates communication with the CA server.
	client, err := lego.NewClient(config)
	if err != nil {
		// lego.NewClient errors are not formatted
		return nil, fmt.Errorf("letsencrypt: %v", err)
	}

	// New users will need to register
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, err
	}
	user.Registration = reg

	// Register challenges

	// Lock mutex as we are going to update the environment
	am.environment.Lock()
	defer am.environment.Unlock()

	// Dispatch supported challenges to dedicated methods
	switch cc.Challenge {
	case "dns-01":
		// Yolo patch the environment with config credentials
		for envvar, value := range cc.Env {
			err := os.Setenv(envvar, value)
			defer os.Unsetenv(envvar)
			if err != nil {
				return nil, err
			}
		}

		// Build provider
		provider, err := dns.NewDNSChallengeProviderByName(cc.Provider)
		if err != nil {
			return nil, err
		}

		// Register provider
		err = client.Challenge.SetDNS01Provider(provider)
		if err != nil {
			return nil, err
		}
	case "http-01":
		// Build provider with some hardcoded parameters
		provider, err := webroot.NewHTTPProvider("/media/acme-challenge")
		if err != nil {
			return nil, err
		}

		// Register provider
		err = client.Challenge.SetHTTP01Provider(provider)
		if err != nil {
			return nil, err
		}
	default:
		// Return an error if no dispatch occurred
		return nil, fmt.Errorf("letsencrypt: challenge %s is not supported", cc.Challenge)
	}

	// Client is cached
	log.Infof("letsencrypt: A new client has been created for %s", cc.Name)
	am.clients[cc.Name] = client

	// Client is returned
	return client, nil
}
