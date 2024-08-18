package PikPakProxy

import (
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/op"
)

type Addition struct {
	driver.RootID
	Username         string `json:"username" required:"true"`
	Password         string `json:"password" required:"true"`
	ClientID         string `json:"client_id" required:"true" default:"YNxT9w7GMdWvEOKa"`
	ClientSecret     string `json:"client_secret" required:"true" default:"dbw2OtmVEeuUvIptb1Coyg"`
	RefreshToken     string `json:"refresh_token" required:"true" default:""`
	CaptchaToken     string `json:"captcha_token" default:""`
	DisableMediaLink bool   `json:"disable_media_link"`
	//是否使用代理
	UseProxy bool `json:"use_proxy"`
	//下代理地址
	ProxyUrl string `json:"proxy_url" default:""`
}

var config = driver.Config{
	Name:        "PikPakProxy",
	LocalSort:   true,
	DefaultRoot: "",
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &PikPakProxy{}
	})
}
