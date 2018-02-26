package caddywarp

import (
	"fmt"
	"log"
	"net"
	"path/filepath"
	"time"

	"github.com/cloudflare/cloudflare-warp/validation"
	"github.com/cloudflare/cloudflare-warp/warp"
	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/sirupsen/logrus"
)

func init() {
	caddy.RegisterPlugin("warp", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	cfg := httpserver.GetConfig(c)

	var count int
	for c.Next() {
		count++
		if count > 1 {
			return c.Err("The warp directive can only be specified once per site")
		}
	}

	// ensure there is a name to warp
	if cfg.Addr.Host == "" {
		return c.Errf("missing hostname to warp")
	}

	// get Cloudflare credential
	credentialPath, err := findOrObtainCredential()
	if err != nil {
		return c.Errf("Error getting Cloudflare credential: %v", err)
	}

	// tunnel makes outbound TLS connection;
	// no TLS connection needed for local listener
	cfg.TLS.Manual = true

	// always have an exit strategy -- this is how we'll cleanly shutdown
	connectedSignal := make(chan struct{})
	shutdownChan := make(chan struct{})
	c.OnShutdown(func() error {
		close(shutdownChan)
		return nil
	})

	// start the tunnel when the server starts; because we will know then
	// what the listener hostname and port are.
	c.OnStartup(func() error {
		// get the host and port we should connect the tunnel to
		// (be aware of catch-all or default values that might be empty)
		proxyToHost := cfg.ListenHost
		if proxyToHost == "" {
			proxyToHost = "localhost"
		}
		proxyToPort := cfg.Addr.Port
		if proxyToPort == "" {
			proxyToPort = httpserver.Port
		}
		if proxyToPort == "" {
			proxyToPort = httpserver.DefaultPort
		}

		// ensure our URL to connect the tunnel to is valid
		tentativeURL := "http://" + net.JoinHostPort(proxyToHost, proxyToPort)
		validURL, err := validation.ValidateUrl(tentativeURL)
		if err != nil {
			return fmt.Errorf("error validating URL : %v", tentativeURL)
		}

		// sanity check: ensure there is still a name to warp
		if cfg.Addr.Host == "" {
			return fmt.Errorf("missing hostname to warp")
		}

		// TODO: metrics?
		// metricsListener, err := listeners.Listen("tcp", c.String("metrics"))
		// if err != nil {
		// 	Log.WithError(err).Fatal("Error opening metrics server listener")
		// }
		// go func() {
		// 	errC <- metrics.ServeMetrics(metricsListener, shutdownChan)
		// 	wg.Done()
		// }()

		// Start the server
		go func() {
			// TODO: Expose configuration options for most of these parameters
			err := warp.StartServer(warp.ServerConfig{
				Hostname:  cfg.Addr.Host,
				ServerURL: validURL,
				// Tags:       tags,
				OriginCert: credentialPath,

				ConnectedChan: connectedSignal,
				ShutdownChan:  shutdownChan,

				Timeout:   30 * time.Second, //c.Duration("proxy-connect-timeout"),
				KeepAlive: 30 * time.Second, //c.Duration("proxy-tcp-keepalive"),
				// DualStack: !c.Bool("proxy-no-happy-eyeballs"),

				MaxIdleConns:        100,              // c.Int("proxy-keepalive-connections"),
				IdleConnTimeout:     90 * time.Second, // c.Duration("proxy-keepalive-timeout"),
				TLSHandshakeTimeout: 10 * time.Second, //c.Duration("proxy-tls-timeout"),

				// EdgeAddrs:         edgeAddrs,
				Retries:           5,               //   c.Uint("retries"),
				HeartbeatInterval: 5 * time.Second, //c.Duration("heartbeat-interval"),
				MaxHeartbeats:     5,               //c.Uint64("heartbeat-count"),
				// LBPool:            c.String("lb-pool"),
				HAConnections: 4, //c.Int("ha-connections"),
				// MetricsUpdateFreq: c.Duration("metrics-update-freq"),
				// IsAutoupdated:     c.Bool("is-autoupdated"),
				// TLSConfig:       tlsconfig.CreateTunnelConfig(c, edge),
				ReportedVersion: caddy.AppName + "/" + caddy.AppVersion,
				// ProtoLogger:     logrus.New(),
				Logger: logrus.New(), // TODO: See if we can replace with the standard logger
			})
			if err != nil {
				log.Printf("[ERROR] Warp tunnel: %v", err)
			}
		}()

		fmt.Println("Warp tunnel is being created. Please allow up to a few minutes for all edge nodes to activate.")
		return nil
	})

	return nil
}

// findOrObtainCredential looks for and prefers a credential certificate
// in Caddy's asset directory. If not there, it checks the official Warp
// client default location. If not there, it tries to obtain one by
// guiding the user through login, and downloading the certificate, which
// will be placed in the Caddy asset directory. It returns the path to the
// credential and an error, if any.
func findOrObtainCredential() (string, error) {
	// Find credential certificate; we will prefer one in the .caddy folder, but
	// we can also use one provisioned by the official Warp client if it exists.
	hasCaddyCredential, err := warp.HasExistingCertificate(caddy.AssetsPath(), credentialCertFilename)
	if err != nil {
		return "", err
	}
	hasOfficialClientCredential, err := warp.HasExistingCertificate(warp.DefaultConfigDir, warp.DefaultCredentialFilename)
	if err != nil {
		return "", err
	}

	if !hasCaddyCredential && !hasOfficialClientCredential {
		err := warp.Login(caddy.AssetsPath(), credentialCertFilename, "")
		if err != nil {
			return "", fmt.Errorf("getting credential from Cloudflare: %v", err)
		}
		hasCaddyCredential = true
	}

	var credentialPath string
	if hasCaddyCredential {
		credentialPath = filepath.Join(caddy.AssetsPath(), credentialCertFilename)
	} else if hasOfficialClientCredential {
		credentialPath = filepath.Join(warp.DefaultConfigDir, warp.DefaultCredentialFilename)
	} else {
		return "", fmt.Errorf("no Cloudflare credential available")
	}

	return credentialPath, nil
}

// The filename of the Cloudflare credential certificate
const credentialCertFilename = "cloudflare-warp.pem"
