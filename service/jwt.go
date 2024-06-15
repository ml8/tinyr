package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	"github.com/ml8/tinyr/service/util"
)

func createToken(uid uint64) (tok string, err error) {
	now := time.Now()
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": fmt.Sprintf("%d", uid),
		"iss": "tinyr",
		"aud": "user",
		"exp": now.Add(authcfg.JWTTimeout).Unix(),
		"iat": now.Unix(),
	})

	tok, err = claims.SignedString(authcfg.JWTKey)
	return
}

func verifyToken(tok string) (uid uint64, ok bool) {
	token, err := jwt.Parse(tok, func(token *jwt.Token) (interface{}, error) {
		return authcfg.JWTKey, nil
	})
	// We don't really check any claims...
	ok = err == nil && token.Valid
	svc.logger.Debug("Token parsed", "ok", ok, "err", err, "token.Valid", token.Valid)
	if ok {
		sub, err := token.Claims.GetSubject()
		util.OkOrDie(err)
		uid, err = strconv.ParseUint(sub, 10, 64)
		util.OkOrDie(err)
		svc.logger.Info("Token ok", "subject", sub, "uid", uid)
	}
	return
}

func verifyRequest(r *http.Request) (uid uint64, ok bool) {
	if hdr := r.Header.Get("Authorization"); strings.HasPrefix(hdr, "Bearer") {
		els := strings.Split(hdr, " ")
		if len(els) != 2 {
			return
		}
		uid, ok = verifyToken(els[1])
		svc.logger.Debug("Token found in header", "ok", ok, "uid", uid)
		return
	}
	// fall back to checking cookie.
	if tok, err := r.Cookie("token"); err == nil {
		uid, ok = verifyToken(tok.Value)
		svc.logger.Debug("Token found in cookie", "ok", ok, "uid", uid)
		return
	}
	return
}
