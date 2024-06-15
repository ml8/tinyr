package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ml8/tinyr/service/cache"
	"github.com/ml8/tinyr/service/db"
	"github.com/ml8/tinyr/service/healthz"
	"github.com/ml8/tinyr/service/util"
)

type cacheEntry struct {
	Long      string
	Timestamp time.Time
}

type instance struct {
	db     db.Interface
	cache  cache.KVCache[cacheEntry]
	ttl    time.Duration
	logger *slog.Logger
}

var svc instance

var reserved map[string]bool

type Config struct {
	AuthConfig
	ShortURLPrefix string
	DB             db.Interface
	CacheTTL       time.Duration
	CacheSize      int
}

func Init(mux *http.ServeMux, config Config) {
	// register routes
	mux.HandleFunc(fmt.Sprintf("%s/create", config.ShortURLPrefix), createHandler)
	mux.HandleFunc(fmt.Sprintf("%s/delete", config.ShortURLPrefix), deleteHandler)
	mux.HandleFunc(fmt.Sprintf("%s/{short}", config.ShortURLPrefix), goHandler)
	reserved = map[string]bool{
		"create": true,
		"delete": true,
	}

	var c cache.KVCache[cacheEntry] = nil
	if config.CacheSize > 0 {
		config.Logger.Info("caching enabled", "size", config.CacheSize, "ttl", config.CacheTTL)
		c = cache.New[cacheEntry](config.CacheSize)
	}
	svc = instance{db: config.DB, cache: c, logger: config.Logger}
	healthz.Register(&svc)

	initAuth(mux, config)

	svc.logger.Info("service config", "config", config)
}

func (s *instance) getWithCache(short string) (long string, err error) {
	if s.cache != nil {
		var entry cacheEntry
		entry, err = s.cache.Get(short)
		if err == nil {
			// in cache; valid?
			if entry.Timestamp.Add(s.ttl).Before(time.Now()) {
				// entry valid; exit early.
				long = entry.Long
				s.logger.Info("cache hit", "short", short, "long", entry.Long)
				return
			}
			// entry is expired
			s.logger.Info("ttl expired", "short", short)
			s.cache.Invalidate(short)
		}
	}
	// not in cache or cache invalid. query.
	s.logger.Info("cache miss", "short", short)
	data, err := svc.db.Shorts().Get(short)
	long = data.Long
	return
}

func (s *instance) invalidateAndReplace(short string, long string) {
	if s.cache == nil {
		return
	}
	s.invalidate(short)
	s.logger.Info("cache replace", "short", short, "long", long)
	s.cache.Put(short, cacheEntry{Long: long, Timestamp: time.Now()})
}

func (s *instance) invalidate(short string) {
	if s.cache == nil {
		return
	}
	s.logger.Info("cache invalidation", "short", short)
	s.cache.Invalidate(short)
}

func goHandler(w http.ResponseWriter, r *http.Request) {
	short := r.PathValue("short")
	svc.logger.Info("Request for", "short", short, "host", util.GetIP(r))

	long, err := svc.getWithCache(short)
	if err != nil {
		svc.logger.Warn("no url found", "short", short, "err", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	http.Redirect(w, r, long, http.StatusTemporaryRedirect)
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	uid, err := UserFrom(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	req := &CreateRequest{}
	if err := Parse(r, &req); err != nil {
		util.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	svc.logger.Info("Create", "short", req.Short, "long", req.Long)
	req.Long = httpify(req.Long)
	if !ValidShort(req.Short) {
		svc.logger.Info("Invalid short", "short", req.Short)
		util.ErrorResponse(w, http.StatusBadRequest, "Short urls must be simple strings")
		return
	} else if err = ValidUrl(req.Long); err != nil {
		svc.logger.Info("Invalid long", "long", req.Long)
		util.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	} else if ok := reserved[req.Short]; ok {
		svc.logger.Info("Reserved short", "short", req.Short)
		util.ErrorResponse(w, http.StatusBadRequest, util.InvalidValueError(req.Short).Error())
		return
	}

	if err := svc.db.Shorts().Put(db.ShortData{Short: req.Short, Long: req.Long, Owner: uid}); err != nil {
		svc.logger.Warn("Error storing", "short", req.Short, "error", err)
		// TODO: response codes
		util.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	svc.invalidateAndReplace(req.Short, req.Long)
	svc.logger.Info("Created", "short", req.Short, "long", req.Long, "owner", uid)
	w.WriteHeader(http.StatusOK)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	uid, err := UserFrom(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	req := &DeleteRequest{}
	if err := Parse(r, &req); err != nil {
		util.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	svc.logger.Info("Delete", "short", req.Short)
	if !ValidShort(req.Short) {
		util.ErrorResponse(w, http.StatusBadRequest, "Short urls must be simple strings")
		return
	}

	entry := db.ShortData{Short: req.Short, Owner: uid}

	if err := svc.db.Shorts().Delete(entry); err != nil {
		// TODO: response codes
		svc.logger.Info("Error deleting", "short", req.Short, "error", err)
		util.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	svc.invalidate(req.Short)
	svc.logger.Info("Deleted", "short", req.Short)
	w.WriteHeader(http.StatusOK)
	return
}

func (s *instance) Healthz(ctx context.Context) error {
	// TODO
	return nil
}
