package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/orange-cloudfoundry/promfetcher/models"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type GorouterConfig struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type BackendConfig struct {
	ClientAuthCertificate tls.Certificate
	MaxConns              int64 `yaml:"max_conns"`
	TLSPem                `yaml:",inline"` // embed to get cert_chain and private_key for client authentication
}

var defaultGorouterConfig = GorouterConfig{
	Host: "localhost",
	Port: 8080,
	User: "",
	Pass: "",
}

type BrokerConfig struct {
	BrokerServiceID string `yaml:"broker_service_id"`
	BrokerPlanID    string `yaml:"broker_plan_id"`
	User            string `yaml:"user"`
	Pass            string `yaml:"pass"`
}

var defaultBrokerConfig = BrokerConfig{
	BrokerPlanID:    "e2900be3-709b-419e-b63b-de3aabcd9e15",
	BrokerServiceID: "75bcebab-cc25-4ef6-89dc-a91b953919f1",
	User:            "user",
	Pass:            "password",
}

type Log struct {
	Level   string `yaml:"level"`
	NoColor bool   `yaml:"no_color"`
	InJson  bool   `yaml:"in_json"`
}

func (c *Log) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Log
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}
	log.SetFormatter(&log.TextFormatter{
		DisableColors: c.NoColor,
	})
	if c.Level != "" {
		lvl, err := log.ParseLevel(c.Level)
		if err != nil {
			return err
		}
		log.SetLevel(lvl)
	}
	if c.InJson {
		log.SetFormatter(&log.JSONFormatter{})
	}

	return nil
}

type TLSPem struct {
	CertChain  string `yaml:"cert_chain"`
	PrivateKey string `yaml:"private_key"`
}

type Config struct {
	Gorouter          GorouterConfig  `yaml:"gorouter,omitempty"`
	Logging           Log             `yaml:"logging,omitempty"`
	Port              uint16          `yaml:"port,omitempty"`
	EnableSSL         bool            `yaml:"enable_ssl,omitempty"`
	SSLCertificate    tls.Certificate `yaml:"-"`
	TLSPEM            TLSPem          `yaml:"tls_pem,omitempty"`
	CACerts           string          `yaml:"ca_certs,omitempty"`
	CAPool            *x509.CertPool  `yaml:"-"`
	SkipSSLValidation bool            `yaml:"skip_ssl_validation,omitempty"`

	Backends BackendConfig `yaml:"backends,omitempty"`

	Broker BrokerConfig `yaml:"broker,omitempty"`

	DisableKeepAlives   bool `yaml:"disable_keep_alives"`
	MaxIdleConns        int  `yaml:"max_idle_conns,omitempty"`
	MaxIdleConnsPerHost int  `yaml:"max_idle_conns_per_host,omitempty"`

	DbConn                string   `yaml:"db_conn"`
	SQLCnxMaxIdle         int      `yaml:"sql_cnx_max_idle"`
	SQLCnxMaxOpen         int      `yaml:"sql_cnx_max_open"`
	SQLCnxMaxLife         string   `yaml:"sql_cnx_max_life"`
	NotExitWhenConnFailed bool     `yaml:"not_exit_when_conn_failed"`
	DB                    *gorm.DB `yaml:"-"`

	BaseURL string `yaml:"base_url"`
}

var defaultConfig = Config{
	Gorouter:            defaultGorouterConfig,
	Logging:             Log{},
	Port:                8085,
	DisableKeepAlives:   true,
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 2,
	SQLCnxMaxIdle:       5,
	SQLCnxMaxOpen:       10,
	SQLCnxMaxLife:       "1h",
	Broker:              defaultBrokerConfig,
	BaseURL:             "http://localhost:8085",
}

func DefaultConfig() (*Config, error) {
	c := defaultConfig
	return &c, nil
}

func (c *Config) Process() error {
	c.BaseURL = strings.TrimSuffix(c.BaseURL, "/")
	if c.Backends.CertChain != "" && c.Backends.PrivateKey != "" {
		certificate, err := tls.X509KeyPair([]byte(c.Backends.CertChain), []byte(c.Backends.PrivateKey))
		if err != nil {
			errMsg := fmt.Sprintf("Error loading key pair: %s", err.Error())
			return fmt.Errorf(errMsg)
		}
		c.Backends.ClientAuthCertificate = certificate
	}

	if c.EnableSSL {
		if c.TLSPEM.PrivateKey == "" || c.TLSPEM.CertChain == "" {
			return fmt.Errorf("Error parsing PEM blocks of router.tls_pem, missing cert or key.")
		}

		certificate, err := tls.X509KeyPair([]byte(c.TLSPEM.CertChain), []byte(c.TLSPEM.PrivateKey))
		if err != nil {
			errMsg := fmt.Sprintf("Error loading key pair: %s", err.Error())
			return fmt.Errorf(errMsg)
		}
		c.SSLCertificate = certificate
	}

	if err := c.buildCertPool(); err != nil {
		return err
	}
	if err := c.gormDB(); err != nil {
		return fmt.Errorf("Error on creating db connexion: %s", err.Error())
	}
	return nil
}

func (c *Config) gormDB() error {
	if c.DbConn == "" {
		return nil
	}
	u, err := url.Parse(c.DbConn)
	if err != nil {
		return err
	}
	user := ""
	if u.User != nil {
		user = u.User.Username()
		password, ok := u.User.Password()
		if ok {
			user += ":" + password
		}
	}
	switch u.Scheme {
	case "mysql":
		fallthrough
	case "mariadb":
		if user != "" {
			user += "@"
		}
		connStr := fmt.Sprintf("%stcp(%s)%s%s", user, u.Host, u.Path, u.RawQuery)
		c.DB, err = gorm.Open("mysql", connStr)
		if err != nil {
			return err
		}
	case "sqlite":
		path := strings.TrimPrefix(u.Path, "/")
		c.DB, err = gorm.Open("sqlite3", path)
		if err != nil {
			return err
		}
	case "postgres":
		c.DB, err = gorm.Open("postgres", u.String())
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("sgbd not found")
	}
	dur, err := time.ParseDuration(c.SQLCnxMaxLife)
	if err != nil {
		return err
	}
	c.DB.DB().SetMaxIdleConns(c.SQLCnxMaxIdle)
	c.DB.DB().SetMaxOpenConns(c.SQLCnxMaxOpen)
	c.DB.DB().SetConnMaxLifetime(dur)
	c.DB.AutoMigrate(&models.AppEndpoint{})
	return nil
}

func (c *Config) buildCertPool() error {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return err
	}

	if c.CACerts != "" {
		if ok := certPool.AppendCertsFromPEM([]byte(c.CACerts)); !ok {
			return fmt.Errorf("Error while adding CACerts to gorouter's cert pool: \n%s\n", c.CACerts)
		}
	}
	c.CAPool = certPool
	return nil
}

func (c *Config) Initialize(configYAML []byte) error {
	c.Gorouter = GorouterConfig{}
	return yaml.Unmarshal(configYAML, &c)
}

func InitConfigFromFile(path string) (*Config, error) {
	c, err := DefaultConfig()
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = c.Initialize(b)
	if err != nil {
		return nil, err
	}

	err = c.Process()
	if err != nil {
		return nil, err
	}

	return c, nil
}
