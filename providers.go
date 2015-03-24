package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/bitly/go-simplejson"
)

type Provider interface {
	LoginUrl() string
	RedeemUrl() string
	Scope() string
	GetEmailAddress(auth_response *simplejson.Json,
		access_token string) (string, error)
}

type ProviderData struct {
	loginUrl *url.URL
	redeemUrl *url.URL
	profileUrl *url.URL
	scope string
}

func NewProviderData(opts *Options) *ProviderData {
	return &ProviderData{
		loginUrl: opts.loginUrl,
		redeemUrl: opts.redeemUrl,
		profileUrl: opts.profileUrl,
		scope: opts.Scope}
}

func (p *ProviderData) LoginUrl() string { return p.loginUrl.String() }
func (p *ProviderData) RedeemUrl() string { return p.redeemUrl.String() }
func (p *ProviderData) Scope() string { return p.scope }

type GoogleProvider struct {
	*ProviderData
}

func NewGoogleProvider(opts *Options) *GoogleProvider {
	if opts.LoginUrl == "" {
		opts.loginUrl = &url.URL{Scheme: "https",
			Host: "accounts.google.com",
			Path: "/o/oauth2/auth"}
	}
	if opts.RedeemUrl == "" {
		opts.redeemUrl = &url.URL{Scheme: "https",
			Host: "accounts.google.com",
			Path: "/o/oauth2/token"}
	}
	if opts.Scope == "" {
		opts.Scope = "profile email"
	}
	return &GoogleProvider{ProviderData: NewProviderData(opts)}
}

func (s *GoogleProvider) GetEmailAddress(auth_response *simplejson.Json,
	unused_access_token string) (string, error) {
	idToken, err := auth_response.Get("id_token").String()
	if err != nil {
		return "", err
	}
	// id_token is a base64 encode ID token payload
	// https://developers.google.com/accounts/docs/OAuth2Login#obtainuserinfo
	jwt := strings.Split(idToken, ".")
	b, err := jwtDecodeSegment(jwt[1])
	if err != nil {
		return "", err
	}
	data, err := simplejson.NewJson(b)
	if err != nil {
		return "", err
	}
	email, err := data.Get("email").String()
	if err != nil {
		return "", err
	}
	return email, nil
}

type MyUsaProvider struct {
	*ProviderData
}

func NewMyUsaProvider(opts *Options) *MyUsaProvider {
	const myUsaHost string = "alpha.my.usa.gov"
	if opts.LoginUrl == "" {
		opts.loginUrl = &url.URL{Scheme: "https",
			Host: myUsaHost,
			Path: "/oauth/authorize"}
	}
	if opts.RedeemUrl == "" {
		opts.redeemUrl = &url.URL{Scheme: "https",
			Host: myUsaHost,
			Path: "/oauth/token"}
	}
	if opts.ProfileUrl == "" {
		opts.profileUrl = &url.URL{Scheme: "https",
			Host: myUsaHost,
			Path: "/api/v1/profile"}
	}
	if opts.Scope == "" {
		opts.Scope = "profile.email"
	}
	return &MyUsaProvider{ProviderData: NewProviderData(opts)}
}

func (s *MyUsaProvider) GetEmailAddress(auth_response *simplejson.Json,
	access_token string) (string, error) {
	req, err := http.NewRequest("GET",
		s.profileUrl.String()+"?access_token="+access_token, nil)
	if err != nil {
		log.Printf("failed building request %s", err)
		return "", err
	}
	json, err := apiRequest(req)
	if err != nil {
		log.Printf("failed making request %s", err)
		return "", err
	}
	return json.Get("email").String()
}

func NewProvider(opts *Options) Provider {
	switch opts.Provider {
	case "myusa":
		return NewMyUsaProvider(opts)
	default:
		return NewGoogleProvider(opts)
	}
}
