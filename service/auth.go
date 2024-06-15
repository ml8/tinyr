package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/zitadel/logging"
	"github.com/zitadel/oidc/v3/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/ml8/tinyr/service/db"
	"github.com/ml8/tinyr/service/util"
)

// TODO: For local testing, create a fake auth layer.
// TODO: Separate auth from short service.

type AuthConfig struct {
	ClientID     string
	ClientSecret string
	Key          []byte
	JWTKey       []byte
	JWTTimeout   time.Duration
	Issuer       string
	Scopes       []string
	BaseURL      string
	CallbackURL  string
	LoginURL     string
	Logger       *slog.Logger
}

var authcfg AuthConfig

const authTemplate = `
<html>
<head><title>Login OK</title></head>
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/4.7.0/css/font-awesome.min.css">
<script>
function copy_token() {
  var tok = document.getElementById("token").innerHTML;
	navigator.clipboard.writeText(tok);
}
</script>
<body>
<p>Paste the following token where requested:</p>
<code id="token">%v</code> <i class="fa fa-clipboard" onclick="copy_token()"/>
</body>
</html>
`

func initAuth(mux *http.ServeMux, config Config) {
	authcfg = config.AuthConfig
	authcfg.Logger.Info("Config", "authcfg", authcfg)

	cookieHandler := httphelper.NewCookieHandler(authcfg.Key, authcfg.Key, httphelper.WithUnsecure())
	client := &http.Client{Timeout: time.Minute}

	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(30 * time.Second)),
		rp.WithHTTPClient(client),
		rp.WithLogger(authcfg.Logger),
	}

	redirect := fmt.Sprintf("%s%s", authcfg.BaseURL, authcfg.CallbackURL)
	ctx := logging.ToContext(context.TODO(), authcfg.Logger)
	provider, err := rp.NewRelyingPartyOIDC(ctx, authcfg.Issuer, authcfg.ClientID, authcfg.ClientSecret, redirect, authcfg.Scopes, options...)
	util.OkOrDie(err)
	urlOptions := []rp.URLParamOpt{
		rp.WithPromptURLParam(""),
	}
	mux.Handle(authcfg.LoginURL, rp.AuthURLHandler(
		func() string { return "" },
		provider,
		urlOptions...,
	))
	mux.Handle(authcfg.CallbackURL, rp.CodeExchangeHandler(rp.UserinfoCallback(responseHandler), provider))
}

func responseHandler(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], state string, rp rp.RelyingParty, info *oidc.UserInfo) {
	svc.logger.Debug("OIDC response", "info", info)
	user := svc.db.Users().LookupOrCreate(db.UserData{Name: info.Name, Email: info.Email})
	tok, err := createToken(user.Id)
	svc.logger.Info("Login", "uid", user.Id)
	if err != nil {
		svc.logger.Error("Could not create token", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "token", Value: tok, Path: "/"})
	msg := fmt.Sprintf(authTemplate, tok)
	w.Write([]byte(msg))
}

func UserFrom(r *http.Request) (uid uint64, err error) {
	ok := false
	uid, ok = verifyRequest(r)
	svc.logger.Info("Auth info", "uid", uid, "ok", ok)
	if !ok {
		err = util.InvalidTokenError
	}
	return
}
