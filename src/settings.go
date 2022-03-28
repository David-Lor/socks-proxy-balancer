package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	EnvServerHost    = "SOCKS_SERVER_HOST"
	EnvServerPort    = "SOCKS_SERVER_PORT"
	EnvConnectPrefix = "SOCKS_CONNECT"
)

type HostPort struct {
	Host string
	Port int64
}

func (host *HostPort) ToAddr() string {
	return fmt.Sprintf("%s:%d", host.Host, host.Port)
}

type Settings struct {
	HostPort
	ProxiesConnect []HostPort
}

func getAllEnvironmentVariables() map[string]string {
	environment := make(map[string]string)
	for _, e := range os.Environ() {
		i := strings.Index(e, "=")
		if i >= 0 {
			environment[e[:i]] = e[i+1:]
		}
	}

	return environment
}

func parsePort(value string) (int64, error) {
	port, err := strconv.ParseInt(value, 10, 64)
	if err == nil {
		if port < 0 || port > 65535 {
			err = errors.New("invalid port")
		}
	}
	return port, err
}

func formatInvalidError(key string, value string) error {
	return fmt.Errorf("invalid %s: '%s'", key, value)
}

func getConnects(allEnv map[string]string) ([]HostPort, []error) {
	var connectEnvs []string
	for k, v := range allEnv {
		if strings.HasPrefix(k, EnvConnectPrefix) {
			connectEnvs = append(connectEnvs, v)
		}
	}

	var connects []HostPort
	var errs []error
	for _, superEnv := range connectEnvs {
		superEnv = strings.ReplaceAll(superEnv, " ", "")
		subEnvs := strings.Split(superEnv, ",")
		superEnvOk := true

		for _, env := range subEnvs {
			ok := func() bool {
				envChunks := strings.Split(env, ":")
				if len(envChunks) != 2 {
					return false
				}

				host := envChunks[0]
				port, err := parsePort(envChunks[1])
				if host == "" || port == 0 || err != nil {
					return false
				}

				connects = append(connects, HostPort{host, port})
				return true
			}()

			if !ok {
				superEnvOk = false
				break
			}
		}

		if !superEnvOk {
			errs = append(errs, formatInvalidError(EnvConnectPrefix, superEnv))
		}
	}

	return connects, errs
}

func getMainSettings(allEnv map[string]string) (*Settings, []error) {
	var errs []error

	serverHost := allEnv[EnvServerHost]
	if serverHost == "" {
		errs = append(errs, formatInvalidError(EnvServerHost, serverHost))
	}

	serverPortRaw := allEnv[EnvServerPort]
	serverPort, serverPortErr := parsePort(serverPortRaw)
	if serverPort == 0 || serverPortErr != nil {
		errs = append(errs, formatInvalidError(EnvServerPort, serverPortRaw))
	}

	settings := Settings{HostPort: HostPort{serverHost, serverPort}}
	return &settings, errs
}

func LoadSettings() (*Settings, []error) {
	allEnv := getAllEnvironmentVariables()

	settings, errors1 := getMainSettings(allEnv)
	connects, errors2 := getConnects(allEnv)
	settings.ProxiesConnect = connects
	if len(settings.ProxiesConnect) == 0 {
		errors2 = append(errors2, errors.New("no proxies to connect defined"))
	}

	errs := append(errors1, errors2...)
	return settings, errs
}
