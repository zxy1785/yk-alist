package homecloud

import (
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/op"
)

// https://homecloud.komect.com/
type Addition struct {
	//Account       string `json:"account" required:"true"`
	RefreshToken string `json:"refresh_token" required:"true"`
	driver.RootID
	GroupID string `json:"groupId" required:"true"`
}

var config = driver.Config{
	Name:      "homecloud",
	LocalSort: true,
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &HomeCloud{}
	})
}
