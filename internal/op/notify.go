package op

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type SendNotifyPlatform struct{}

// 注意映射方法名必需大写要不然找不到
func (e SendNotifyPlatform) Bark(body string, title string, content string) (bool, error) {
	var bark model.Bark
	err := json.Unmarshal([]byte(body), &bark)
	if err != nil {
		log.Errorln("无法解析配置文件")
		return false, errors.Errorf("无法解析配置文件")
	}

	if len(bark.BarkPush) < 2 {
		log.Errorln("请正确设置BarkPush")
		return false, errors.Errorf("请正确设置BarkPush")
	}

	if !strings.HasPrefix(bark.BarkPush, "http") {
		bark.BarkPush = fmt.Sprintf("https://api.day.app/%s", bark.BarkPush)
	}
	urlValues := url.Values{}
	urlValues.Set("icon", bark.BarkIcon)
	urlValues.Set("sound", bark.BarkSound)
	urlValues.Set("group", bark.BarkGroup)
	urlValues.Set("level", bark.BarkLevel)
	urlValues.Set("url", bark.BarkUrl)
	url := fmt.Sprintf("%s/%s/%s?%s", bark.BarkPush, url.QueryEscape(title), url.QueryEscape(content), urlValues.Encode())

	resp, err := http.Get(url)
	if err != nil {
		log.Error("通知发送失败")
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else {
		return false, fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}
}

// 注意映射方法名必需大写要不然找不到
func (e SendNotifyPlatform) Gotify(body string, title string, content string) (bool, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(body), &m)
	gotifyUrl, gotifyToken, gotifyPriority := m["gotifyUrl"].(string), m["gotifyToken"].(string), m["gotifyPriority"].(string)

	surl := fmt.Sprintf("%s/message?token=%s", gotifyUrl, gotifyToken)
	data := url.Values{}
	data.Set("title", title)
	data.Set("message", content)
	data.Set("priority", fmt.Sprintf("%d", gotifyPriority))

	req, err := http.NewRequest("POST", surl, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if err != nil {
		log.Error("通知发送失败")
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else {
		return false, fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}
}

// 注意映射方法名必需大写要不然找不到
func (e SendNotifyPlatform) GoCqHttpBot(body string, title string, content string) (bool, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(body), &m)
	goCqHttpBotUrl, goCqHttpBotToken, goCqHttpBotQq := m["goCqHttpBotUrl"].(string), m["goCqHttpBotToken"].(string), m["goCqHttpBotQq"].(string)

	surl := fmt.Sprintf("%s?user_id=%s", goCqHttpBotUrl, goCqHttpBotQq)
	data := map[string]string{"message": fmt.Sprintf("%s\n%s", title, content)}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest("POST", surl, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+goCqHttpBotToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if err != nil {
		log.Error("通知发送失败")
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else {
		return false, fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}
}

// 注意映射方法名必需大写要不然找不到
func (e SendNotifyPlatform) ServerChan(body string, title string, content string) (bool, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(body), &m)
	serverChanKey := m["serverChanKey"].(string)

	surl := ""
	if len(serverChanKey) >= 3 && serverChanKey[:3] == "SCT" {
		surl = fmt.Sprintf("https://sctapi.ftqq.com/%s.send", serverChanKey)
	} else {
		surl = fmt.Sprintf("https://sc.ftqq.com/%s.send", serverChanKey)
	}

	data := url.Values{}
	data.Set("title", title)
	data.Set("desp", content)

	req, err := http.NewRequest("POST", surl, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if err != nil {
		log.Error("通知发送失败")
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else {
		return false, fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}
}

// 注意映射方法名必需大写要不然找不到
func (e SendNotifyPlatform) PushDeer(body string, title string, content string) (bool, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(body), &m)
	pushDeerKey, pushDeerUrl := m["pushDeerKey"].(string), m["pushDeerUrl"].(string)

	surl := pushDeerUrl
	if surl == "" {
		surl = "https://api2.pushdeer.com/message/push"
	}

	data := url.Values{}
	data.Set("pushkey", pushDeerKey)
	data.Set("text", title)
	data.Set("desp", content)
	data.Set("type", "markdown")

	req, err := http.NewRequest("POST", surl, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if err != nil {
		log.Error("通知发送失败")
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else {
		return false, fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}
}

// 注意映射方法名必需大写要不然找不到
func (e SendNotifyPlatform) TelegramBot(body string, title string, content string) (bool, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(body), &m)
	telegramBotToken, telegramBotUserId, telegramBotProxyHost, telegramBotProxyPort, telegramBotProxyAuth, telegramBotApiHost := m["telegramBotToken"].(string), m["telegramBotUserId"].(string), m["telegramBotProxyHost"].(string), m["telegramBotProxyPort"].(string), m["telegramBotProxyAuth"].(string), m["telegramBotApiHost"].(string)

	if telegramBotApiHost == "" {
		telegramBotApiHost = "https://api.telegram.org"
	}

	surl := fmt.Sprintf("%s/bot%s/sendMessage", telegramBotApiHost, telegramBotToken)

	var client *http.Client
	if telegramBotProxyHost != "" && telegramBotProxyPort != "" {
		proxyURL := fmt.Sprintf("http://%s:%s", telegramBotProxyHost, telegramBotProxyPort)
		if telegramBotProxyAuth != "" {
			proxyURL = fmt.Sprintf("http://%s@%s:%s", telegramBotProxyAuth, telegramBotProxyHost, telegramBotProxyPort)
		}

		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(proxyURL)
		}

		client = &http.Client{
			Transport: &http.Transport{
				Proxy: proxy,
			},
		}
	} else {
		client = http.DefaultClient
	}

	data := url.Values{}
	data.Set("chat_id", telegramBotUserId)
	data.Set("text", fmt.Sprintf("%s\n\n%s", title, content))
	data.Set("disable_web_page_preview", "true")

	req, err := http.NewRequest("POST", surl, strings.NewReader(data.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if err != nil {
		log.Error("通知发送失败")
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else {
		return false, fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}
}

// 注意映射方法名必需大写要不然找不到
func (e SendNotifyPlatform) WeWorkBot(body string, title string, content string) (bool, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(body), &m)
	weWorkBotKey, weWorkOrigin := m["weWorkBotKey"].(string), m["weWorkOrigin"].(string)

	if weWorkOrigin == "" {
		weWorkOrigin = "https://qyapi.weixin.qq.com"
	}

	surl := fmt.Sprintf("%s/cgi-bin/webhook/send?key=%s", weWorkOrigin, weWorkBotKey)

	bodyData := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": fmt.Sprintf("%s\n\n%s", title, content),
		},
	}
	data, err := json.Marshal(bodyData)
	if err != nil {
		return false, err
	}

	var client *http.Client
	client = http.DefaultClient

	req, err := http.NewRequest("POST", surl, strings.NewReader(string(data)))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if err != nil {
		log.Error("通知发送失败")
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else {
		return false, fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}
}

// 注意映射方法名必需大写要不然找不到
func (e SendNotifyPlatform) Webhook(body string, title string, content string) (bool, error) {
	var webhook model.Webhook
	err := json.Unmarshal([]byte(body), &webhook)
	webhookBodyString := string(webhook.WebhookBody)
	if err != nil {
		log.Errorln("无法解析配置文件")
		return false, errors.New("无法解析配置文件")
	}

	if !strings.Contains(webhook.WebhookUrl, "$title") && !strings.Contains(webhookBodyString, "$title") {
		return false, errors.New("URL 或者 Body 中必须包含 $title")
	}

	headers := make(map[string]string)
	if len(webhook.WebhookHeaders) > 2 {
		// 按换行符分割字符串
		headerLines := strings.Split(webhook.WebhookHeaders, "\n")
		// 遍历每一行
		for _, line := range headerLines {
			// 忽略空行
			if line == "" {
				continue
			}
			// 按冒号分割键值对
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				return false, fmt.Errorf("malformed header: %s", line)
			}
			// 去除键和值两端的空白字符
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// 将键值对添加到 map 中
			headers[key] = value
		}
	}
	targetBody := strings.ReplaceAll(strings.ReplaceAll(webhookBodyString, "$title", title), "$content", content)
	rbodys := make(map[string]string)
	if len(targetBody) > 2 {
		// 按换行符分割字符串
		headerLines := strings.Split(targetBody, "\n")
		// 遍历每一行
		for _, line := range headerLines {
			// 忽略空行
			if line == "" {
				continue
			}
			// 按冒号分割键值对
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				return false, fmt.Errorf("malformed header: %s", line)
			}
			// 去除键和值两端的空白字符
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// 将键值对添加到 map 中
			rbodys[key] = value
		}
	}

	var fbody *bytes.Buffer
	switch webhook.WebhookContentType {
	case "application/json":
		fbody, err = formatJSON(rbodys)
	case "multipart/form-data":
		fbody, err = formatMultipart(rbodys)
	case "application/x-www-form-urlencoded", "text/plain":
		fbody, err = formatURLForm(rbodys)
	default:
		fmt.Println("Unsupported content type")
		return false, nil
	}
	// if err != nil {
	// 	log.Errorln("WebhookBody解析失败")
	// 	return false, errors.New("WebhookBody解析失败")
	// }
	formatURL := strings.ReplaceAll(strings.ReplaceAll(webhook.WebhookUrl, "$title", url.QueryEscape(title)), "$content", url.QueryEscape(content))
	client := &http.Client{}
	req, err := http.NewRequest(webhook.WebhookMethod, formatURL, fbody)
	if err != nil {
		log.Errorln("Webhook创建请求失败")
		return false, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if err != nil {
		log.Error("通知发送失败")
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else {
		return false, fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}
}

var platform model.SettingItem

func Notify(title string, content string) {
	platform, err := GetSettingItemByKey(conf.NotifyPlatform)
	enable, err := GetSettingItemByKey(conf.NotifyEnabled)
	notifyBody, err := GetSettingItemByKey(conf.NotifyValue)

	if err != nil {
		log.Error("无法找到配置信息")
	}
	if enable.Value != "true" && enable.Value != "1" {
		log.Debug("未开启消息推送功能")
		return
	}

	if !conf.Conf.Notify {
		log.Debug("配置文件禁用通知")
		return
	}

	caser := cases.Title(language.English)
	methodName := caser.String(platform.Value)

	//注意映射方法名必需大写要不然找不到
	// 使用反射获取结构体实例的值
	v := reflect.ValueOf(SendNotifyPlatform{})
	// 检查是否成功获取结构体实例的值
	if v.IsValid() {
		log.Debug("成功获取结构体实例的值")
	} else {
		log.Debug("未能获取结构体实例的值")
		return
	}

	method := v.MethodByName(methodName)
	// 检查方法是否存在
	if !method.IsValid() {
		log.Debug("Method %s not found\n", methodName)
		return
	}
	args := []reflect.Value{reflect.ValueOf(notifyBody.Value), reflect.ValueOf(title), reflect.ValueOf(content)}
	// 调用方法

	method.Call(args)
}

// formatJSON 格式化为 JSON
func formatJSON(bodys map[string]string) (*bytes.Buffer, error) {
	jsonData, err := json.Marshal(bodys)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(jsonData), nil
}

// formatMultipart 格式化为 multipart/form-data
func formatMultipart(bodys map[string]string) (*bytes.Buffer, error) {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	for key, value := range bodys {
		err := writer.WriteField(key, value)
		if err != nil {
			return nil, err
		}
	}
	err := writer.Close()
	if err != nil {
		return nil, err
	}

	return &b, nil
}

// formatURLForm 格式化为 application/x-www-form-urlencoded
func formatURLForm(bodys map[string]string) (*bytes.Buffer, error) {
	values := url.Values{}
	for key, value := range bodys {
		values.Add(key, value)
	}
	formData := values.Encode()
	return bytes.NewBufferString(formData), nil
}
