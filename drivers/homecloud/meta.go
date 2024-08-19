package homecloud

import (
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/op"
)

// https://homecloud.komect.com/
type Addition struct {
	//Account       string `json:"account" required:"true"`
	Authorization string `json:"authorization" type:"text" required:"true"`
	driver.RootID
	Type    string `json:"type" type:"select" options:"personal,family,personal_new" default:"personal"`
	CloudID string `json:"cloud_id"`
	UserID  string `json:"userId"  required:"true"`
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
