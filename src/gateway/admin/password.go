package admin

import (
	"net/http"

	"gateway/config"
	aphttp "gateway/http"
	"gateway/mail"
	"gateway/model"
	apsql "gateway/sql"

	"github.com/gorilla/handlers"
)

type PasswordResetController struct {
	BaseController
}

type PasswordReset struct {
	Email string `json:"email"`
}

func RoutePasswordReset(controller *PasswordResetController, path string,
	router aphttp.Router, db *apsql.DB, conf config.ProxyAdmin) {

	routes := map[string]http.Handler{
		"POST": write(db, controller.Reset),
	}
	if conf.CORSEnabled {
		routes["OPTIONS"] = aphttp.CORSOptionsHandler([]string{"POST", "OPTIONS"})
	}

	router.Handle(path, handlers.MethodHandler(routes))
}

func (c *PasswordResetController) Reset(w http.ResponseWriter, r *http.Request, tx *apsql.Tx) aphttp.Error {
	request := struct {
		PasswordReset PasswordReset `json:"password_reset"`
	}{}
	if err := deserialize(&request, r.Body); err != nil {
		return err
	}

	user, err := model.FindUserByEmail(tx.DB, request.PasswordReset.Email)
	if err != nil {
		return nil
	}

	err = mail.SendResetEmail(c.SMTP, c.ProxyServer, c.conf, user, tx)
	if err != nil {
		return aphttp.NewError(err, http.StatusBadRequest)
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

type PasswordResetConfirmationController struct {
	BaseController
}

type PasswordResetConfirmation struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func RoutePasswordResetConfirmation(controller *PasswordResetConfirmationController, path string,
	router aphttp.Router, db *apsql.DB, conf config.ProxyAdmin) {

	routes := map[string]http.Handler{
		"POST": write(db, controller.Confirmation),
	}
	if conf.CORSEnabled {
		routes["OPTIONS"] = aphttp.CORSOptionsHandler([]string{"POST", "OPTIONS"})
	}

	router.Handle(path, handlers.MethodHandler(routes))
}

func (c *PasswordResetConfirmationController) Confirmation(w http.ResponseWriter, r *http.Request, tx *apsql.Tx) aphttp.Error {
	request := struct {
		PasswordResetConfirmation PasswordResetConfirmation `json:"password_reset_confirmation"`
	}{}
	if err := deserialize(&request, r.Body); err != nil {
		return err
	}
	user, err := model.ValidateUserToken(tx, request.PasswordResetConfirmation.Token)
	if err != nil {
		return aphttp.NewError(err, http.StatusBadRequest)
	}
	user.NewPassword = request.PasswordResetConfirmation.NewPassword
	err = user.Update(tx)
	if err != nil {
		return aphttp.NewError(err, http.StatusBadRequest)
	}

	w.WriteHeader(http.StatusOK)
	return nil
}