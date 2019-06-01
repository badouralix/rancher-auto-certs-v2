package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-acme/lego/log"
	"gopkg.in/yaml.v2"
)

// accountConfig is used to cache letsencrypt clients and users
// It is a subset of certConfig and we don't expect having many accountConfig
type accountConfig struct {
	CA      string
	Email   string
	Key     string
	KeyType string
}

type certConfig struct {
	AccountEmail       string `yaml:"account_email"`
	AccountKey         string `yaml:"account_key"`
	CA                 string
	Challenge          string
	CreateKeyIfMissing *bool `yaml:"create_key_if_missing"` // boolean pointer here to differentiate empty value from zero value
	Description        string
	Domains            []string
	KeyType            string `yaml:"key_type"`
	Name               string
	Provider           string `json:",omitempty" yaml:",omitempty"`
}

func (cc certConfig) getAccountConfig() accountConfig {
	return accountConfig{
		CA:      cc.CA,
		Email:   cc.AccountEmail,
		Key:     cc.AccountKey,
		KeyType: cc.KeyType,
	}
}

// We guarantee that a certConfig has a private key
// If it does not yet, create it here
func (cc certConfig) getPrivateKey() (crypto.PrivateKey, error) {
	// See https://stackoverflow.com/questions/21322182/how-to-store-ecdsa-private-key-in-go

	if _, err := os.Stat(cc.AccountKey); os.IsNotExist(err) && *cc.CreateKeyIfMissing {
		// Create a new key, and save it to disk
		// - cipher: ECDSA
		// - encoding: x509
		// - file format: pem
		log.Infof("config: Generating new key ( %s does not exist )", cc.AccountKey)

		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("config: %v", err)
		}

		x509Encoded, err := x509.MarshalECPrivateKey(privateKey)
		if err != nil {
			return nil, fmt.Errorf("config: %v", err)
		}

		pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})

		err = ioutil.WriteFile(cc.AccountKey, pemEncoded, 0400)
		if err != nil {
			// TODO: delete file here
			return nil, fmt.Errorf("config: %v", err)
		}

		return privateKey, nil
	}

	// Read private key from disk
	log.Infof("config: Reading key from %s", cc.AccountKey)

	b, err := ioutil.ReadFile(cc.AccountKey)
	if err != nil {
		return nil, fmt.Errorf("config: %v", err)
	}

	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("config: unable to read PEM data from %s", cc.AccountKey)
	}

	x509Encoded := block.Bytes

	privateKey, err := x509.ParseECPrivateKey(x509Encoded)
	if err != nil {
		return nil, fmt.Errorf("config: %v", err)
	}

	return privateKey, nil
}

type defaultConfig struct {
	AccountEmail       string `yaml:"account_email"`
	AccountKey         string `yaml:"account_key"`
	CA                 string
	Challenge          string
	CreateKeyIfMissing bool `yaml:"create_key_if_missing"`
	Description        string
	KeyType            string `yaml:"key_type"`
	Provider           string `json:",omitempty" yaml:",omitempty"`
}

type globalConfig struct {
	Default defaultConfig
	Certs   []certConfig
}

func loadConfig(path string) (config globalConfig, err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

// Replace empty values by default
func mergeDefaultConfig(cc certConfig, dc defaultConfig) certConfig {
	if cc.AccountEmail == "" {
		cc.AccountEmail = dc.AccountEmail
	}

	if cc.AccountKey == "" {
		cc.AccountKey = dc.AccountKey
	}

	if cc.CA == "" {
		cc.CA = dc.CA
	}

	if cc.Challenge == "" {
		cc.Challenge = dc.Challenge
	}

	if cc.CreateKeyIfMissing == nil {
		cc.CreateKeyIfMissing = &dc.CreateKeyIfMissing
	}

	if cc.Description == "" {
		// Instantiate default config format specifier
		cc.Description = fmt.Sprintf(dc.Description, cc.Name)
	}

	if cc.KeyType == "" {
		cc.KeyType = dc.KeyType
	}

	if cc.Provider == "" {
		cc.Provider = dc.Provider
	}

	return cc
}
