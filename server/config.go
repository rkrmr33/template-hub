package server

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/rkrmr33/pkg/log"
	"github.com/rkrmr33/template-hub/assets"
	"github.com/rkrmr33/template-hub/common"
	"github.com/rkrmr33/template-hub/util"
	serverutil "github.com/rkrmr33/template-hub/util/server"
)

const (
	MaxGRPCMessageSize = 1 << 20 // 100mb
	ConnectionTimeout  = time.Second * 100
)

var (
	DefaultListenAddr = fmt.Sprintf(":%d", common.ServerPort)

	DefaultBackoffConfig = wait.Backoff{
		Steps:    5,
		Duration: 500 * time.Millisecond,
		Factor:   1.0,
		Jitter:   0.1,
	}
)

type (
	Config struct {
		insecure bool
		lisAddr  string
		rootPath string
		hostname string

		tlsCertPath string
		tlsKeyPath  string
	}
)

func AddFlags(flags *pflag.FlagSet) *Config {
	c := &Config{}

	util.Must(viper.BindEnv("root-path", "TEMPLATE_HUB_ROOT_PATH"))
	util.Must(viper.BindEnv("hostname", "TEMPLATE_HUB_HOSTNAME"))

	flags.StringVarP(&c.lisAddr, "listen", "l", DefaultListenAddr, "Server listen address")
	flags.StringVar(&c.rootPath, "root-path", viper.GetString("root-path"), "The server root path")
	flags.BoolVarP(&c.insecure, "insecure", "k", false, "If set to false, will not use tls")
	flags.StringVar(&c.hostname, "hostname", viper.GetString("hostname"), "Server hostname")
	flags.StringVar(&c.tlsCertPath, "tls-cert", "", "TLS cert file path")
	flags.StringVar(&c.tlsKeyPath, "tls-key", "", "TLS key file path")

	return c
}

func (c *Config) Build() (Server, error) {
	var (
		err error
		s   server
	)

	s.lisAddr = DefaultListenAddr
	if c.lisAddr != "" {
		s.lisAddr = c.lisAddr
	}

	if !c.insecure {
		s.tlsConf, s.tlsCert, err = c.getTLSConf()
		if err != nil {
			return nil, fmt.Errorf("failed to get tls config: %w", err)
		}
	}

	if s.envMock, err = ioutil.ReadFile("/Users/roikramer/playground/snapshots/state.db"); err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	s.log = log.G().Named("server")
	s.rootPath = strings.TrimRight(strings.TrimLeft(c.rootPath, "/"), "/")
	s.staticAssetsFs = assets.StaticFS
	s.backoffConfig = DefaultBackoffConfig

	return &s, nil
}

func (c *Config) getTLSConf() (*tls.Config, []byte, error) {
	var (
		err     error
		cert    tls.Certificate
		tlsCert []byte
	)

	if c.tlsCertPath != "" && c.tlsKeyPath != "" {
		// load server certificates from file
		log.G().Debugf("loading server cert from: %s", c.tlsCertPath)
		log.G().Debugf("loading server key from: %s", c.tlsKeyPath)

		tlsCert, err = ioutil.ReadFile(c.tlsCertPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read tls cert: %w", err)
		}

		cert, err = tls.LoadX509KeyPair(c.tlsCertPath, c.tlsKeyPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load tls certificates from file: %w", err)
		}

		return &tls.Config{
			Certificates:       []tls.Certificate{cert},
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		}, tlsCert, nil
	}

	log.G().Debug("generating self-signed certificates")

	return serverutil.GenerateTLSConfig(c.hostname)
}
