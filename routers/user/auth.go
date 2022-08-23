// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package user

import (
	"net/url"
	"strings"

	"github.com/macaron-contrib/captcha"

	"github.com/topitobravo/gogs/models"
	"github.com/topitobravo/gogs/modules/auth"
	"github.com/topitobravo/gogs/modules/base"
	"github.com/topitobravo/gogs/modules/log"
	"github.com/topitobravo/gogs/modules/mailer"
	"github.com/topitobravo/gogs/modules/middleware"
	"github.com/topitobravo/gogs/modules/setting"
)

const (
	SIGNIN          base.TplName = "user/auth/signin"
	SIGNUP          base.TplName = "user/auth/signup"
	ACTIVATE        base.TplName = "user/auth/activate"
	FORGOT_PASSWORD base.TplName = "user/auth/forgot_passwd"
	RESET_PASSWORD  base.TplName = "user/auth/reset_passwd"
)

func SignIn(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Tr("sign_in")

	if _, ok := ctx.Session.Get("socialId").(int64); ok {
		ctx.Data["IsSocialLogin"] = true
		ctx.HTML(200, SIGNIN)
		return
	}

	if setting.OauthService != nil {
		ctx.Data["OauthEnabled"] = true
		ctx.Data["OauthService"] = setting.OauthService
	}

	// Check auto-login.
	uname := ctx.GetCookie(setting.CookieUserName)
	if len(uname) == 0 {
		ctx.HTML(200, SIGNIN)
		return
	}

	isSucceed := false
	defer func() {
		if !isSucceed {
			log.Trace("auto-login cookie cleared: %s", uname)
			ctx.SetCookie(setting.CookieUserName, "", -1, setting.AppSubUrl)
			ctx.SetCookie(setting.CookieRememberName, "", -1, setting.AppSubUrl)
			return
		}
	}()

	u, err := models.GetUserByName(uname)
	if err != nil {
		if err != models.ErrUserNotExist {
			ctx.Handle(500, "GetUserByName", err)
		}
		return
	}

	if val, _ := ctx.GetSuperSecureCookie(
		base.EncodeMd5(u.Rands+u.Passwd), setting.CookieRememberName); val != u.Name {
		ctx.HTML(200, SIGNIN)
		return
	}

	isSucceed = true

	ctx.Session.Set("uid", u.Id)
	ctx.Session.Set("uname", u.Name)
	if redirectTo, _ := url.QueryUnescape(ctx.GetCookie("redirect_to")); len(redirectTo) > 0 {
		ctx.SetCookie("redirect_to", "", -1, setting.AppSubUrl)
		ctx.Redirect(redirectTo)
		return
	}

	ctx.Redirect(setting.AppSubUrl + "/")
}

func SignInPost(ctx *middleware.Context, form auth.SignInForm) {
	ctx.Data["Title"] = ctx.Tr("sign_in")

	sid, isOauth := ctx.Session.Get("socialId").(int64)
	if isOauth {
		ctx.Data["IsSocialLogin"] = true
	} else if setting.OauthService != nil {
		ctx.Data["OauthEnabled"] = true
		ctx.Data["OauthService"] = setting.OauthService
	}

	if ctx.HasError() {
		ctx.HTML(200, SIGNIN)
		return
	}

	u, err := models.UserSignIn(form.UserName, form.Password)
	if err != nil {
		if err == models.ErrUserNotExist {
			ctx.RenderWithErr(ctx.Tr("form.username_password_incorrect"), SIGNIN, &form)
		} else {
			ctx.Handle(500, "UserSignIn", err)
		}
		return
	}

	if form.Remember {
		days := 86400 * setting.LogInRememberDays
		ctx.SetCookie(setting.CookieUserName, u.Name, days, setting.AppSubUrl)
		ctx.SetSuperSecureCookie(base.EncodeMd5(u.Rands+u.Passwd),
			setting.CookieRememberName, u.Name, days, setting.AppSubUrl)
	}

	// Bind with social account.
	if isOauth {
		if err = models.BindUserOauth2(u.Id, sid); err != nil {
			if err == models.ErrOauth2RecordNotExist {
				ctx.Handle(404, "GetOauth2ById", err)
			} else {
				ctx.Handle(500, "GetOauth2ById", err)
			}
			return
		}
		ctx.Session.Delete("socialId")
		log.Trace("%s OAuth binded: %s -> %d", ctx.Req.RequestURI, form.UserName, sid)
	}

	ctx.Session.Set("uid", u.Id)
	ctx.Session.Set("uname", u.Name)
	if redirectTo, _ := url.QueryUnescape(ctx.GetCookie("redirect_to")); len(redirectTo) > 0 {
		ctx.SetCookie("redirect_to", "", -1, setting.AppSubUrl)
		ctx.Redirect(redirectTo)
		return
	}

	ctx.Redirect(setting.AppSubUrl + "/")
}

func SignOut(ctx *middleware.Context) {
	ctx.Session.Delete("uid")
	ctx.Session.Delete("uname")
	ctx.Session.Delete("socialId")
	ctx.Session.Delete("socialName")
	ctx.Session.Delete("socialEmail")
	ctx.SetCookie(setting.CookieUserName, "", -1, setting.AppSubUrl)
	ctx.SetCookie(setting.CookieRememberName, "", -1, setting.AppSubUrl)
	ctx.Redirect(setting.AppSubUrl + "/")
}

func oauthSignUp(ctx *middleware.Context, sid int64) {
	ctx.Data["Title"] = ctx.Tr("sign_up")

	if _, err := models.GetOauth2ById(sid); err != nil {
		if err == models.ErrOauth2RecordNotExist {
			ctx.Handle(404, "GetOauth2ById", err)
		} else {
			ctx.Handle(500, "GetOauth2ById", err)
		}
		return
	}

	ctx.Data["IsSocialLogin"] = true
	ctx.Data["uname"] = strings.Replace(ctx.Session.Get("socialName").(string), " ", "", -1)
	ctx.Data["email"] = ctx.Session.Get("socialEmail")
	log.Trace("social ID: %v", ctx.Session.Get("socialId"))
	ctx.HTML(200, SIGNUP)
}

func SignUp(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Tr("sign_up")

	if setting.Service.DisableRegistration {
		ctx.Data["DisableRegistration"] = true
		ctx.HTML(200, SIGNUP)
		return
	}

	if sid, ok := ctx.Session.Get("socialId").(int64); ok {
		oauthSignUp(ctx, sid)
		return
	}

	ctx.HTML(200, SIGNUP)
}

func SignUpPost(ctx *middleware.Context, cpt *captcha.Captcha, form auth.RegisterForm) {
	ctx.Data["Title"] = ctx.Tr("sign_up")

	if setting.Service.DisableRegistration {
		ctx.Error(403)
		return
	}

	isOauth := false
	sid, isOauth := ctx.Session.Get("socialId").(int64)
	if isOauth {
		ctx.Data["IsSocialLogin"] = true
	}

	// May redirect from home page.
	if ctx.Query("from") == "home" {
		// Clear input error box.
		ctx.Data["Err_UserName"] = false
		ctx.Data["Err_Email"] = false

		// Make the best guess.
		uname := ctx.Query("uname")
		i := strings.Index(uname, "@")
		if i > -1 {
			ctx.Data["email"] = uname
			ctx.Data["uname"] = uname[:i]
		} else {
			ctx.Data["uname"] = uname
		}
		ctx.Data["password"] = ctx.Query("password")
		ctx.HTML(200, SIGNUP)
		return
	}

	if ctx.HasError() {
		ctx.HTML(200, SIGNUP)
		return
	}

	if !cpt.VerifyReq(ctx.Req) {
		ctx.Data["Err_Captcha"] = true
		ctx.RenderWithErr(ctx.Tr("form.captcha_incorrect"), SIGNUP, &form)
		return
	} else if form.Password != form.Retype {
		ctx.Data["Err_Password"] = true
		ctx.RenderWithErr(ctx.Tr("form.password_not_match"), SIGNUP, &form)
		return
	}

	u := &models.User{
		Name:     form.UserName,
		Email:    form.Email,
		Passwd:   form.Password,
		IsActive: !setting.Service.RegisterEmailConfirm || isOauth,
	}

	if err := models.CreateUser(u); err != nil {
		switch err {
		case models.ErrUserAlreadyExist:
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("form.username_been_taken"), SIGNUP, &form)
		case models.ErrEmailAlreadyUsed:
			ctx.Data["Err_Email"] = true
			ctx.RenderWithErr(ctx.Tr("form.email_been_used"), SIGNUP, &form)
		case models.ErrUserNameIllegal:
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("form.illegal_username"), SIGNUP, &form)
		default:
			ctx.Handle(500, "CreateUser", err)
		}
		return
	}
	log.Trace("Account created: %s", u.Name)

	// Bind social account.
	if isOauth {
		if err := models.BindUserOauth2(u.Id, sid); err != nil {
			ctx.Handle(500, "BindUserOauth2", err)
			return
		}
		ctx.Session.Delete("socialId")
		log.Trace("%s OAuth binded: %s -> %d", ctx.Req.RequestURI, form.UserName, sid)
	}

	// Send confirmation e-mail, no need for social account.
	if !isOauth && setting.Service.RegisterEmailConfirm && u.Id > 1 {
		mailer.SendRegisterMail(ctx.Render, u)
		ctx.Data["IsSendRegisterMail"] = true
		ctx.Data["Email"] = u.Email
		ctx.Data["Hours"] = setting.Service.ActiveCodeLives / 60
		ctx.HTML(200, ACTIVATE)

		if err := ctx.Cache.Put("MailResendLimit_"+u.LowerName, u.LowerName, 180); err != nil {
			log.Error(4, "Set cache(MailResendLimit) fail: %v", err)
		}
		return
	}

	ctx.Redirect(setting.AppSubUrl + "/user/login")
}

func Activate(ctx *middleware.Context) {
	code := ctx.Query("code")
	if len(code) == 0 {
		ctx.Data["IsActivatePage"] = true
		if ctx.User.IsActive {
			ctx.Error(404)
			return
		}
		// Resend confirmation e-mail.
		if setting.Service.RegisterEmailConfirm {
			if ctx.Cache.IsExist("MailResendLimit_" + ctx.User.LowerName) {
				ctx.Data["ResendLimited"] = true
			} else {
				ctx.Data["Hours"] = setting.Service.ActiveCodeLives / 60
				mailer.SendActiveMail(ctx.Render, ctx.User)

				if err := ctx.Cache.Put("MailResendLimit_"+ctx.User.LowerName, ctx.User.LowerName, 180); err != nil {
					log.Error(4, "Set cache(MailResendLimit) fail: %v", err)
				}
			}
		} else {
			ctx.Data["ServiceNotEnabled"] = true
		}
		ctx.HTML(200, ACTIVATE)
		return
	}

	// Verify code.
	if user := models.VerifyUserActiveCode(code); user != nil {
		user.IsActive = true
		user.Rands = models.GetUserSalt()
		if err := models.UpdateUser(user); err != nil {
			if err == models.ErrUserNotExist {
				ctx.Error(404)
			} else {
				ctx.Handle(500, "UpdateUser", err)
			}
			return
		}

		log.Trace("User activated: %s", user.Name)

		ctx.Session.Set("uid", user.Id)
		ctx.Session.Set("uname", user.Name)
		ctx.Redirect(setting.AppSubUrl + "/")
		return
	}

	ctx.Data["IsActivateFailed"] = true
	ctx.HTML(200, ACTIVATE)
}

func ForgotPasswd(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Tr("auth.forgot_password")

	if setting.MailService == nil {
		ctx.Data["IsResetDisable"] = true
		ctx.HTML(200, FORGOT_PASSWORD)
		return
	}

	ctx.Data["IsResetRequest"] = true
	ctx.HTML(200, FORGOT_PASSWORD)
}

func ForgotPasswdPost(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Tr("auth.forgot_password")

	if setting.MailService == nil {
		ctx.Handle(403, "user.ForgotPasswdPost", nil)
		return
	}
	ctx.Data["IsResetRequest"] = true

	email := ctx.Query("email")
	u, err := models.GetUserByEmail(email)
	if err != nil {
		if err == models.ErrUserNotExist {
			ctx.Data["Err_Email"] = true
			ctx.RenderWithErr(ctx.Tr("auth.email_not_associate"), FORGOT_PASSWORD, nil)
		} else {
			ctx.Handle(500, "user.ResetPasswd(check existence)", err)
		}
		return
	}

	if ctx.Cache.IsExist("MailResendLimit_" + u.LowerName) {
		ctx.Data["ResendLimited"] = true
		ctx.HTML(200, FORGOT_PASSWORD)
		return
	}

	mailer.SendResetPasswdMail(ctx.Render, u)
	if err = ctx.Cache.Put("MailResendLimit_"+u.LowerName, u.LowerName, 180); err != nil {
		log.Error(4, "Set cache(MailResendLimit) fail: %v", err)
	}

	ctx.Data["Email"] = email
	ctx.Data["Hours"] = setting.Service.ActiveCodeLives / 60
	ctx.Data["IsResetSent"] = true
	ctx.HTML(200, FORGOT_PASSWORD)
}

func ResetPasswd(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Tr("auth.reset_password")

	code := ctx.Query("code")
	if len(code) == 0 {
		ctx.Error(404)
		return
	}
	ctx.Data["Code"] = code
	ctx.Data["IsResetForm"] = true
	ctx.HTML(200, RESET_PASSWORD)
}

func ResetPasswdPost(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Tr("auth.reset_password")

	code := ctx.Query("code")
	if len(code) == 0 {
		ctx.Error(404)
		return
	}
	ctx.Data["Code"] = code

	if u := models.VerifyUserActiveCode(code); u != nil {
		// Validate password length.
		passwd := ctx.Query("password")
		if len(passwd) < 6 {
			ctx.Data["IsResetForm"] = true
			ctx.Data["Err_Password"] = true
			ctx.RenderWithErr(ctx.Tr("auth.password_too_short"), RESET_PASSWORD, nil)
			return
		}

		u.Passwd = passwd
		u.Rands = models.GetUserSalt()
		u.Salt = models.GetUserSalt()
		u.EncodePasswd()
		if err := models.UpdateUser(u); err != nil {
			ctx.Handle(500, "UpdateUser", err)
			return
		}

		log.Trace("User password reset: %s", u.Name)
		ctx.Redirect(setting.AppSubUrl + "/user/login")
		return
	}

	ctx.Data["IsResetFailed"] = true
	ctx.HTML(200, RESET_PASSWORD)
}
