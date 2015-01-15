package admin

import (
	"net/http"

	"gateway/config"
	aphttp "gateway/http"
	sql "gateway/sql"

	"github.com/gorilla/mux"
)

func subrouter(router *mux.Router, config config.ProxyAdmin) *mux.Router {
	adminRoute := router.NewRoute()
	if config.Host != "" {
		adminRoute = adminRoute.Host(config.Host)
	}
	if config.PathPrefix != "" {
		adminRoute = adminRoute.PathPrefix(config.PathPrefix)
	}
	return adminRoute.Subrouter()
}

// Setup sets up the session and adds admin routes.
func Setup(router *mux.Router, db *sql.DB, conf config.ProxyAdmin) {
	setupSessions(conf)

	var admin aphttp.Router
	admin = aphttp.NewAccessLoggingRouter(config.Admin, subrouter(router, conf))

	// siteAdmin is additionally protected for the site owner
	siteAdmin := aphttp.NewHTTPBasicRouter(conf.Username, conf.Password, conf.Realm, admin)
	RouteAccounts(siteAdmin, db)
	RouteAccountUsers(siteAdmin, db)

	// sessions are unprotected to allow users to authenticate
	RouteSessions(admin, db)

	admin.Handle("/{path:.*}", http.HandlerFunc(adminStaticFileHandler))
}
