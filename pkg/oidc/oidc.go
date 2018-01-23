package oidc

import (
	"context"
	"fmt"
	"log"
	"net/http"

	oidc "github.com/coreos/go-oidc"
	"github.com/labstack/echo"
	"github.com/pwillie/oidc-kubeconfig-helper/pkg/k8s"
	"golang.org/x/oauth2"
)

type (
	OidcConfig struct {
		Provider     *oidc.Provider
		ClientID     string
		ClientSecret string
		CallbackURL  string
	}
)

var (
	state = "nonprod" //TODO:
)

func NewOidcConfig(clientID, clientSecret, callbackURL, providerURL *string) *OidcConfig {
	config := &OidcConfig{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		CallbackURL:  *callbackURL,
	}
	var err error
	fmt.Printf("Initialising OIDC discovery endpoint: %v\n", *providerURL)
	config.Provider, err = oidc.NewProvider(context.Background(), *providerURL)
	if err != nil {
		log.Fatalf("Unable to initialise provider: %v", err)
	}
	return config
}

func (cfg OidcConfig) SigninHandler(c echo.Context) error {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     cfg.Provider.Endpoint(),
		RedirectURL:  cfg.CallbackURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
	}
	return c.Redirect(http.StatusFound, oauthConfig.AuthCodeURL(state))
}

func (cfg OidcConfig) CallbackHandler(c echo.Context) error {
	if c.QueryParam("state") != state {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid state")
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     cfg.Provider.Endpoint(),
		RedirectURL:  cfg.CallbackURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
	}
	oauth2Token, err := oauthConfig.Exchange(context.Background(), c.QueryParam("code"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to exchange token: "+err.Error())
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "No id_token field in oauth2 token.")
	}
	idTokenVerifier := cfg.Provider.Verifier(
		&oidc.Config{ClientID: cfg.ClientID, SupportedSigningAlgs: []string{"RS256"}},
	)
	idToken, err := idTokenVerifier.Verify(context.Background(), rawIDToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify ID Token: "+err.Error())
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse claims: "+err.Error())
	}
	output, err := k8s.GenerateUserKubeconfig(
		claims.Email,
		idToken.Issuer,
		cfg.ClientID,
		cfg.ClientSecret,
		rawIDToken,
		oauth2Token.RefreshToken,
	)

	return c.String(http.StatusOK, string(output))
}
