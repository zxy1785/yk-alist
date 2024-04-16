package model

type Bark struct {
	BarkPush  string `json:"barkPush"`
	BarkIcon  string `json:"barkIcon,omitempty"`  // 可选字段
	BarkSound string `json:"barkSound,omitempty"` // 可选字段
	BarkGroup string `json:"barkGroup,omitempty"` // 可选字段
	BarkLevel string `json:"barkLevel,omitempty"` // 可选字段
	BarkUrl   string `json:"barkUrl,omitempty"`   // 可选字段
}

type Webhook struct {
	WebhookUrl         string `json:"webhookUrl"`
	WebhookBody        string `json:"webhookBody,omitempty"`    // 可选字段
	WebhookHeaders     string `json:"webhookHeaders,omitempty"` // 可选字段
	WebhookMethod      string `json:"webhookMethod"`            // 可选字段
	WebhookContentType string `json:"webhookContentType"`       // 可选字段
}
