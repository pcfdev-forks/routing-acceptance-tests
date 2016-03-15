package helpers

import (
	"encoding/json"
	"os"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/nu7hatch/gouuid"
)

type RoutingConfig struct {
	helpers.Config
	Addresses     []string     `json:"addresses"`
	SystemDomain  string       `json:"system_domain"`
	RoutingApiUrl string       `json:"routing_api_url"`
	OAuth         *OAuthConfig `json:"oauth"`
}

type OAuthConfig struct {
	TokenEndpoint            string `json:"token_endpoint"`
	ClientName               string `json:"client_name"`
	ClientSecret             string `json:"client_secret"`
	Port                     int    `json:"port"`
	SkipOAuthTLSVerification bool   `json:"skip_oauth_tls_verification"`
}

func LoadConfig() RoutingConfig {
	loadedConfig := loadConfigJsonFromPath()
	loadedConfig.Config = helpers.LoadConfig()

	if loadedConfig.OAuth == nil {
		panic("missing configuration oauth")
	}

	if len(loadedConfig.Addresses) == 0 {
		panic("missing configuration 'addresses'")
	}

	if loadedConfig.RoutingApiUrl == "" && loadedConfig.SystemDomain == "" {
		panic("Need to set either routing_api_url or system_domain")
	}
	if loadedConfig.RoutingApiUrl == "" {
		loadedConfig.RoutingApiUrl = loadedConfig.Protocol() + "api." + loadedConfig.SystemDomain
	}

	return loadedConfig
}

func loadConfigJsonFromPath() RoutingConfig {
	var config RoutingConfig

	path := configPath()

	configFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&config)
	if err != nil {
		panic(err)
	}

	return config
}

func configPath() string {
	path := os.Getenv("CONFIG")
	if path == "" {
		panic("Must set $CONFIG to point to an integration config .json file.")
	}

	return path
}

func (c RoutingConfig) Protocol() string {
	if c.UseHttp {
		return "http://"
	} else {
		return "https://"
	}
}

func RandomName() string {
	guid, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	return guid.String()
}
