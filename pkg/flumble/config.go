package flumble

import (
	"crypto/tls"
	"net"
	"strconv"
	"time"

	"layeh.com/gumble/gumble"
	flag "github.com/spf13/pflag"
)

type Config struct {
	Gumble          *gumble.Config
	MumbleAddr      string
	FlrigAddr       string
	MumbleTlsConfig tls.Config
	IgnoreUsername	string
	AudioTimeout    time.Duration
}

func BuildConfig() (*Config, error) {
	config := &Config{}
	config.Gumble = gumble.NewConfig()

	flag.StringVar(&config.MumbleAddr, "mumble-addr", "localhost:64738", "Mumble server address")
	flag.StringVar(&config.FlrigAddr, "flrig-addr", "localhost:12345", "FLrig xmlrpc address")
	flag.StringVar(&config.Gumble.Username, "username", "flumble-bot", "client username")
	flag.StringVar(&config.Gumble.Password, "password", "", "client password")
	flag.StringVar(&config.IgnoreUsername, "ignore-user", "Radiopi", "username of radio to ignore")
	flag.DurationVar(&config.AudioTimeout, "audio-timeout", 250*time.Millisecond, "timeout to consider stopped talking")

	insecure := flag.Bool("insecure", false, "skip server certificate verification")
	certificateFile := flag.String("certificate", "", "user certificate file (PEM)")
	keyFile := flag.String("key", "", "user certificate key file (PEM)")

	if !flag.Parsed() {
		flag.Parse()
	}

	host, port, err := net.SplitHostPort(config.MumbleAddr)
	// assume host with default port if cannot be parsed
	if err != nil {
		host = config.MumbleAddr
		port = strconv.Itoa(gumble.DefaultPort)
	}
	config.MumbleAddr = net.JoinHostPort(host, port)

	if *insecure {
		config.MumbleTlsConfig.InsecureSkipVerify = true
	}
	if *certificateFile != "" {
		if *keyFile == "" {
			keyFile = certificateFile
		}
		if certificate, err := tls.LoadX509KeyPair(*certificateFile, *keyFile); err != nil {
			return nil, err
		} else {
			config.MumbleTlsConfig.Certificates = append(config.MumbleTlsConfig.Certificates, certificate)
		}
	}

	return config, nil
}
