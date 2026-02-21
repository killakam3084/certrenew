package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	defaultConfigPath  = "/etc/certrenew/config.json"
	certbotImage       = "certbot/dns-route53"
	defaultVerifyHost  = ""
	defaultVerifyPort  = "443"
)

// Config holds all configuration for certrenew.
type Config struct {
	LetsEncryptDir  string `json:"letsencrypt_dir"`
	CertName        string `json:"cert_name"`
	NginxContainer  string `json:"nginx_container"`
	Domain          string `json:"domain"`
	AWSAccessKeyID  string `json:"aws_access_key_id"`
	AWSSecretKey    string `json:"aws_secret_access_key"`
}

func main() {
	configPath := flag.String("config", defaultConfigPath, "Path to config file")
	dryRun := flag.Bool("dry-run", false, "Print actions without executing")
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime)
	log.SetPrefix("[certrenew] ")

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := validate(cfg); err != nil {
		log.Fatalf("config validation failed: %v", err)
	}

	if *dryRun {
		printDryRun(cfg)
		return
	}

	if err := runCertbot(cfg); err != nil {
		log.Fatalf("certbot failed: %v", err)
	}

	if err := restartNginx(cfg); err != nil {
		log.Fatalf("nginx restart failed: %v", err)
	}

	if cfg.Domain != "" {
		if err := verifyCert(cfg.Domain, defaultVerifyPort); err != nil {
			log.Printf("WARNING: cert verification failed: %v", err)
		}
	}

	log.Println("done!")
}

func loadConfig(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving config path: %w", err)
	}

	f, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("opening config file %s: %w", absPath, err)
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.LetsEncryptDir == "" {
		return fmt.Errorf("letsencrypt_dir is required")
	}
	if cfg.CertName == "" {
		return fmt.Errorf("cert_name is required")
	}
	if cfg.NginxContainer == "" {
		return fmt.Errorf("nginx_container is required")
	}
	if cfg.AWSAccessKeyID == "" {
		return fmt.Errorf("aws_access_key_id is required")
	}
	if cfg.AWSSecretKey == "" {
		return fmt.Errorf("aws_secret_access_key is required")
	}
	return nil
}

func runCertbot(cfg *Config) error {
	log.Printf("running certbot renewal for %s", cfg.CertName)

	args := []string{
		"run", "--rm",
		"-e", fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", cfg.AWSAccessKeyID),
		"-e", fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", cfg.AWSSecretKey),
		"-v", fmt.Sprintf("%s:/etc/letsencrypt", cfg.LetsEncryptDir),
		certbotImage,
		"renew",
		"--cert-name", cfg.CertName,
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("certbot container: %w", err)
	}

	log.Printf("certbot renewal succeeded for %s", cfg.CertName)
	return nil
}

func restartNginx(cfg *Config) error {
	log.Printf("restarting nginx container: %s", cfg.NginxContainer)

	cmd := exec.Command("docker", "restart", cfg.NginxContainer)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restarting %s: %w", cfg.NginxContainer, err)
	}

	log.Printf("nginx container restarted, waiting for it to come up...")
	time.Sleep(3 * time.Second)
	return nil
}

func verifyCert(domain, port string) error {
	addr := net.JoinHostPort(domain, port)
	log.Printf("verifying certificate at %s", addr)

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: domain,
	})
	if err != nil {
		return fmt.Errorf("tls dial %s: %w", addr, err)
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return fmt.Errorf("no certificates returned")
	}

	leaf := certs[0]
	log.Printf("certificate verified:")
	log.Printf("  subject:    %s", leaf.Subject.CommonName)
	log.Printf("  issuer:     %s", leaf.Issuer.CommonName)
	log.Printf("  not before: %s", leaf.NotBefore.Format(time.RFC1123))
	log.Printf("  not after:  %s", leaf.NotAfter.Format(time.RFC1123))
	log.Printf("  expires in: %s", time.Until(leaf.NotAfter).Round(time.Hour))

	return nil
}

func printDryRun(cfg *Config) {
	fmt.Println("[dry-run] would execute the following steps:")
	fmt.Println()
	fmt.Printf("  1. docker run --rm \\\n")
	fmt.Printf("       -e AWS_ACCESS_KEY_ID=****** \\\n")
	fmt.Printf("       -e AWS_SECRET_ACCESS_KEY=****** \\\n")
	fmt.Printf("       -v %s:/etc/letsencrypt \\\n", cfg.LetsEncryptDir)
	fmt.Printf("       %s renew --cert-name %s\n", certbotImage, cfg.CertName)
	fmt.Println()
	fmt.Printf("  2. docker restart %s\n", cfg.NginxContainer)
	fmt.Println()
	if cfg.Domain != "" {
		fmt.Printf("  3. verify TLS cert at %s:%s\n", cfg.Domain, defaultVerifyPort)
	}
}
