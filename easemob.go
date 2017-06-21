package easemob

import (
	"fmt"
	"net/http"

	"github.com/go-resty/resty"
)

// EM ...
type EM struct {
	clientID     string
	clientSecret string
	baseURL      string
	token        string
}

// New ...
func New(clientID, clientSecret, baseURL string) *EM {
	em := &EM{clientID, clientSecret, baseURL, ``}
	em.init()
	return em
}

func (em *EM) init() {
	resty.SetDebug(true)
	resty.AddRetryCondition(func(r *resty.Response) (bool, error) {
		return (r.StatusCode() == http.StatusTooManyRequests || r.StatusCode() == http.StatusServiceUnavailable), nil
	})
	resty.OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
		r.SetAuthToken(em.token)
		r.SetHeader("Accept", "application/json")
		r.SetHeader(`Content-Type`, `application/json`)
		r.SetResult(map[string]interface{}{})
		r.SetError(map[string]interface{}{})
		return nil
	})

	// 更新token
	// go em.refreshToken()
}

// RegisterSignelUser ...
func (em *EM) RegisterSignelUser(username, password string) error {
	resp, err := em.excute(resty.R().SetBody(map[string]string{
		`username`: username,
		`password`: password,
	}), resty.MethodPost, em.url(`/users`))
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf(`em error: %v`, resp.Error())
	}
	return nil
}

// DeleteUser ...
func (em *EM) DeleteUser(username string) error {
	resp, err := em.excute(resty.R(), resty.MethodDelete, em.url(fmt.Sprintf(`/users/%s`, username)))
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf(`em error: %v`, resp.Error())
	}
	return nil
}

// SendCMDMsg 忽略服务端返回的发送结果
func (em *EM) SendCMDMsg(targets []string, action string) error {
	_, err := em.excute(resty.R().SetBody(map[string]interface{}{
		`target_type`: `users`,
		`target`:      targets,
		`msg`: map[string]string{
			`type`:   `cmd`,
			`action`: action,
		},
	}), resty.MethodPost, em.url(`/messages`))

	return err
}

func (em *EM) excute(request *resty.Request, method, url string) (*resty.Response, error) {
	resp, err := request.Execute(method, url)
	if err != nil {
		return resp, err
	}

	if resp.StatusCode() == http.StatusUnauthorized { // 需要更新token
		if em.refreshToken() {
			return em.excute(request, method, url)
		}
	}
	return resp, err
}

func (em *EM) refreshToken() bool {
	resp, _ := resty.New().SetDebug(true).R().
		SetBody(map[string]string{
			`grant_type`:    `client_credentials`,
			`client_id`:     em.clientID,
			`client_secret`: em.clientSecret,
		}).
		SetResult(map[string]interface{}{}).
		Post(em.url(`/token`))
	info := *resp.Result().(*map[string]interface{})
	if token, ok := info[`access_token`].(string); ok {
		em.token = token
		return true
	}
	return false
}

func (em *EM) url(endpoint string) string {
	return em.baseURL + endpoint
}
