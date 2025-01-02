package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/smtp"
	"os"
	"os/signal"
	"syscall"

	log "github.com/flashmob/go-guerrilla/log"
)

const (
	DefaultSTMPPort        = 465
	DefaultMaxEmailSize    = (10 << 23) // 83 MB
	DefaultLocalListenIP   = "0.0.0.0"
	DefaultLocalListenPort = 2525
	DefaultTimeoutSecs     = 300 // 5 minutes
)

// Logger is the global logger
var Logger log.Logger

type mailRelayConfig struct {
	SMTPServer        string   `json:"smtp_server"`
	SMTPPort          int      `json:"smtp_port"`
	SMTPStartTLS      bool     `json:"smtp_starttls"`
	SMTPLoginAuthType bool     `json:"smtp_login_auth_type"`
	SMTPUsername      string   `json:"smtp_username"`
	SMTPPassword      string   `json:"smtp_password"`
	SkipCertVerify    bool     `json:"smtp_skip_cert_verify"`
	MaxEmailSize      int64    `json:"smtp_max_email_size"`
	LocalListenIP     string   `json:"local_listen_ip"`
	LocalListenPort   int      `json:"local_listen_port"`
	AllowedHosts      []string `json:"allowed_hosts"`
	TimeoutSecs       int      `json:"timeout_secs"`
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func run() error {
	var configFile string
	var test bool
	var testsender string
	var testrcpt string
	var verbose bool
	flag.StringVar(&configFile, "config", "/etc/mailrelay.json", "specifies JSON config file")
	flag.BoolVar(&test, "test", false, "sends a test message to SMTP server")
	flag.StringVar(&testsender, "sender", "", "used with 'test' to specify sender email address")
	flag.StringVar(&testrcpt, "rcpt", "", "used with 'test' to specify recipient email address")
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.Parse()

	appConfig, err := loadConfig(configFile)
	if err != nil {
		flag.Usage()
		return fmt.Errorf("loading config: %w", err)
	}

	err = Start(appConfig, verbose)
	if err != nil {
		flag.Usage()
		return fmt.Errorf("starting server: %w", err)
	}

	logLevel := "info"
	if verbose {
		logLevel = "debug"
	}
	Logger, err = log.GetLogger("stdout", logLevel)
	if err != nil {
		return fmt.Errorf("creating logger: %w", err)
	}

	if test {
		err = sendTest(testsender, testrcpt, appConfig.LocalListenPort)
		if err != nil {
			return fmt.Errorf("sending test message: %w", err)
		}
		return nil
	}

	// Wait for SIGINT
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Block until a signal is received.
	<-c
	return nil
}

func loadConfig(path string) (*mailRelayConfig, error) {
	var cfg mailRelayConfig
	configDefaults(&cfg)

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	parser := json.NewDecoder(file)
	if err := parser.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func configDefaults(config *mailRelayConfig) {
	config.SMTPPort = DefaultSTMPPort
	config.SMTPStartTLS = false
	config.SMTPLoginAuthType = false
	config.MaxEmailSize = DefaultMaxEmailSize
	config.SkipCertVerify = false
	config.LocalListenIP = DefaultLocalListenIP
	config.LocalListenPort = DefaultLocalListenPort
	config.AllowedHosts = []string{"*"}
	config.TimeoutSecs = DefaultTimeoutSecs
}

// sendTest sends a test message to the SMTP server specified in mailrelay.json
func sendTest(sender string, rcpt string, port int) error {
	conn, err := smtp.Dial(fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return err
	}

	if err := conn.Mail(sender); err != nil {
		return err
	}
	if err := conn.Rcpt(rcpt); err != nil {
		return err
	}

	if err := writeBody(conn, sender); err != nil {
		return err
	}
	return conn.Quit()
}

func writeBody(conn *smtp.Client, sender string) error {
	wc, err := conn.Data()
	if err != nil {
		return err
	}
	defer wc.Close()
	_, err = fmt.Fprintf(wc, "From: %s\nSubject: Test message\n\nThis is a test email from mailrelay.\n", sender)
	return err
}
