package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/peterbourgon/ff"
	"golang.org/x/crypto/acme/autocert"

	"github.com/ml8/tinyr/service"
	"github.com/ml8/tinyr/service/db"
	"github.com/ml8/tinyr/service/healthz"
)

var (
	logger *slog.Logger
	fs     = flag.NewFlagSet("tinyr", flag.ExitOnError)

	// General flags
	_         = fs.String("config", "", "config file")
	hostname  = fs.String("hostname", "http://localhost", "Hostname")
	port      = fs.String("port", ":8080", "port to listen on")
	homeSrc   = fs.String("homePage", "home.html", "home page source")
	cacheTTL  = fs.Duration("cacheTTL", time.Minute*5, "ttl for caching entries")
	cacheSize = fs.Int("cacheSize", 1024, "size of url cache")

	// TLS flags
	certDir = fs.String("certDir", "", "directory for certificate caching")
	domain  = fs.String("domain", "", "domain for TLS")
	useTLS  = fs.Bool("tls", false, "use TLS")

	// Database flags
	pebblePath  = fs.String("pebblePath", "", "path to PebbleDB directory")
	cqlHosts    = fs.String("cqlHosts", "", "comma-separated list of cql hosts")
	cqlKeyspace = fs.String("cqlKeyspace", "tinyr", "keyspace for cql")
	connStr     = fs.String("connStr", "", "sql connection string")
	sqlDriver   = fs.String("sqlDriver", "mysql", "sql database driver")

	// Auth flags
	clientID     = fs.String("clientID", "", "OIDC client ID")
	clientSecret = fs.String("clientSecret", "", "OIDC client secret")
	cookieKey    = fs.String("cookieKey", "", "Cookie key")
	issuer       = fs.String("issuer", "", "OIDC issuer")
	scopes       = fs.String("scopes", "openid,profile", "Comma-separated list of scopes")
	jwtKey       = fs.String("jwtKey", "", "JWT key")
	jwtTimeout   = fs.Duration("jwtTimeout", 30*24*time.Hour, "JWT timeout")

	p string
)

func dbConfig(logger *slog.Logger) (cfg db.Config) {
	// TODO: add flag for method.
	cfg.Logger = logger
	cfg.Type = db.InMemory

	if *pebblePath != "" {
		cfg.Type = db.Pebble
		cfg.Pebble.Path = *pebblePath
	}

	if *cqlHosts != "" {
		cfg.Type = db.CQL
		cfg.CQL.Hosts = strings.Split(*cqlHosts, ",")
		cfg.CQL.Keyspace = *cqlKeyspace
	}

	if *connStr != "" {
		cfg.Type = db.SQL
		cfg.SQL.ConnString = *connStr
		cfg.SQL.Driver = *sqlDriver
	}

	return
}

func main() {
	ff.Parse(fs, os.Args[1:],
		ff.WithEnvVarPrefix("TINYR"),
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser))

	logger = slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		}))

	p = *port
	if !strings.HasPrefix(p, ":") {
		p = ":" + p
	}

	// Only for testing...
	if *cookieKey == "" {
		*cookieKey = uuid.New().String()[0:16]
	}

	// Only for testing...
	if *jwtKey == "" {
		*jwtKey = *cookieKey
	}

	h := &home{fetchIndex(*homeSrc)}
	mux := http.NewServeMux()
	mux.Handle("/", h)
	mux.Handle("/healthz", healthz.Handler())

	config := service.Config{}
	config.Logger = logger
	config.DB = db.New(dbConfig(config.Logger))
	config.ShortURLPrefix = ""

	config.ClientID = *clientID
	config.ClientSecret = *clientSecret
	config.Key = []byte(*cookieKey)
	config.JWTKey = []byte(*jwtKey)
	config.JWTTimeout = *jwtTimeout
	config.Issuer = *issuer
	config.Scopes = strings.Split(*scopes, ",")
	config.BaseURL = *hostname
	config.CallbackURL = "/auth"
	config.LoginURL = "/login"
	config.CacheSize = *cacheSize
	config.CacheTTL = *cacheTTL

	service.Init(mux, config)

	if *useTLS {
		serveTLS(mux)
	} else {
		serve(mux)
	}
}

type home struct {
	src string
}

func (h *home) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, h.src)
}

func fetchIndex(path string) string {
	f, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("Could not read %v: %v", path, err))
	}
	return string(f)
}

func serveTLS(mux *http.ServeMux) {
	u, _ := url.Parse(*hostname)
	allowedHost := u.Host
	certManager := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		HostPolicy: func(ctx context.Context, host string) error {
			if host == allowedHost {
				return nil
			}
			return fmt.Errorf("invalid host %v", host)
		},
		Cache: autocert.DirCache(*certDir),
	}
	server := http.Server{
		Addr:    ":443",
		Handler: mux,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}
	logger.Info(fmt.Sprintf("Serving on %v", server.Addr))
	go http.ListenAndServe(":80", certManager.HTTPHandler(nil))
	panic(server.ListenAndServeTLS("", ""))
}

func serve(mux *http.ServeMux) {
	logger.Info(fmt.Sprintf("Serving on %v", p))
	panic(http.ListenAndServe(p, mux))
}
