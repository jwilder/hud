package docker

import (
	"fmt"
	"strings"

	"os"
	"strconv"
	dockerapi "github.com/fsouza/go-dockerclient"
)

// based off of https://github.com/dotcloud/docker/blob/2a711d16e05b69328f2636f88f8eac035477f7e4/utils/utils.go
func parseHost(addr string) (string, string, error) {

	var (
		proto string
		host  string
		port  int
	)
	addr = strings.TrimSpace(addr)
	switch {
	case addr == "tcp://":
		return "", "", fmt.Errorf("Invalid bind address format: %s", addr)
	case strings.HasPrefix(addr, "unix://"):
		proto = "unix"
		addr = strings.TrimPrefix(addr, "unix://")
		if addr == "" {
			addr = "/var/run/docker.sock"
		}
	case strings.HasPrefix(addr, "tcp://"):
		proto = "tcp"
		addr = strings.TrimPrefix(addr, "tcp://")
	case strings.HasPrefix(addr, "fd://"):
		return "fd", addr, nil
	case addr == "":
		proto = "unix"
		addr = "/var/run/docker.sock"
	default:
		if strings.Contains(addr, "://") {
			return "", "", fmt.Errorf("Invalid bind address protocol: %s", addr)
		}
		proto = "tcp"
	}

	if proto != "unix" && strings.Contains(addr, ":") {
		hostParts := strings.Split(addr, ":")
		if len(hostParts) != 2 {
			return "", "", fmt.Errorf("Invalid bind address format: %s", addr)
		}
		if hostParts[0] != "" {
			host = hostParts[0]
		} else {
			host = "127.0.0.1"
		}

		if p, err := strconv.Atoi(hostParts[1]); err == nil && p != 0 {
			port = p
		} else {
			return "", "", fmt.Errorf("Invalid bind address format: %s", addr)
		}

	} else if proto == "tcp" && !strings.Contains(addr, ":") {
		return "", "", fmt.Errorf("Invalid bind address format: %s", addr)
	} else {
		host = addr
	}
	if proto == "unix" {
		return proto, host, nil

	}
	return proto, fmt.Sprintf("%s:%d", host, port), nil
}

func GetEndpoint() (string, error) {
	defaultEndpoint := "unix:///var/run/docker.sock"
	if os.Getenv("DOCKER_HOST") != "" {
		defaultEndpoint = os.Getenv("DOCKER_HOST")
	}

	//FIXME: jwilder: add endpoint flag
	/*	if endpoint != "" {
			defaultEndpoint = endpoint
		}
	*/
	_, _, err := parseHost(defaultEndpoint)
	if err != nil {
		return "", err
	}

	return defaultEndpoint, nil
}

func NewDockerClient(endpoint string) (*dockerapi.Client, error) {
	if strings.HasPrefix(endpoint, "unix:") {
		return dockerapi.NewClient(endpoint)
		//FIXME: jwilder: add tls support
		/*	} else if tlsVerify || tlsCert != "" || tlsKey != "" || tlsCaCert != "" {
			if tlsVerify {
				if tlsCaCert == "" {
					return nil, errors.New("TLS verification was requested, but no -tlscacert was provided")
				}
			}

			return dockerapi.NewTLSClient(endpoint, tlsCert, tlsKey, tlsCaCert)*/
	}
	return dockerapi.NewClient(endpoint)
}
