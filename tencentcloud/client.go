package tencentcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	common "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/regions"
	"github.com/vincenthcui/awesome-tencentcloud-go/tencentcloud/actions"
	"github.com/vincenthcui/awesome-tencentcloud-go/tencentcloud/sign"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultDomain        = "tencentcloudapi.com"
	defaultRequestClient = "awesome-tencentcloud-go"
	defaultMethod        = http.MethodPost
	defaultURI           = "/"
	defaultQuery         = ""
	schemaHttps          = "https"
	authorizeTpl         = "%s Credential=%s, SignedHeaders=%s, Signature=%s"

	eol        = "\n"         // end of line
	dateLayout = "2006-01-02" // ref: package time

	headerHost          = "Host"
	headerContentType   = "Content-Type"
	headerTCAction      = "X-TC-Action"
	headerTCVersion     = "X-TC-Version"
	headerTCTimestamp   = "X-TC-Timestamp"
	headerTCLanguage    = "X-TC-Language"
	headerTCRegion      = "X-TC-Region"
	headerRequestClient = "X-TC-RequestClient"

	contentTypeJson = "application/json"
	algorithmSHA256 = "TC3-HMAC-SHA256"
	scopeTC3Request = "tc3_request"
	languageZhCN    = "zh-CN"
)

func NewClient(opts ...Option) *Client {
	cli := &Client{
		client:    &http.Client{},
		language:  languageZhCN,
		region:    regions.Guangzhou,
		algorithm: algorithmSHA256,

		httpMethod: defaultMethod,
		httpURI:    defaultURI,
		httpQuery:  defaultQuery,
	}
	for idx := range opts {
		opts[idx](cli)
	}
	return cli
}

type Client struct {
	client *http.Client

	secretID  string
	secretKey string
	region    string
	language  string
	algorithm string

	httpMethod string
	httpURI    string
	httpQuery  string
}

func (c *Client) Send(ctx context.Context, action actions.Action, request interface{}, response interface{}) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("tencentcloud: marshal request failed: %+v", err)
	}
	u := url.URL{Scheme: schemaHttps, Host: action.Host(defaultDomain), Path: defaultURI, RawQuery: defaultQuery}
	httpRequest, err := http.NewRequest(defaultMethod, u.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}

	now := time.Now()
	headers := map[string]string{
		headerHost:        action.Host(defaultDomain),
		headerContentType: contentTypeJson,

		headerTCAction:      action.Action(),
		headerTCVersion:     action.Version(),
		headerTCTimestamp:   strconv.FormatInt(now.Unix(), 10),
		headerRequestClient: defaultRequestClient,
		headerTCLanguage:    c.language,
		headerTCRegion:      c.region,
	}

	headers["Authorization"] = c.authorize(action, headers, body, now)
	for k, v := range headers {
		httpRequest.Header[k] = []string{v}
	}

	httpResponse, err := c.client.Do(httpRequest)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()

	byts, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return err
	}

	terr := common.ErrorResponse{}
	err = json.Unmarshal(byts, &terr)
	if err != nil {
		return err
	}
	if terr.Response.Error.Code != "" {
		return &errors.TencentCloudSDKError{
			Code:      terr.Response.Error.Code,
			Message:   terr.Response.Error.Message,
			RequestId: terr.Response.RequestId,
		}
	}

	return json.Unmarshal(byts, response)
}

func (c *Client) authorize(action actions.Action, headers map[string]string, body []byte, now time.Time) string {
	date := now.UTC().Format(dateLayout)
	timestamp := strconv.FormatInt(now.Unix(), 10)
	scope := fmt.Sprintf("%s/%s/%s/%s", c.secretID, date, action.Service(), scopeTC3Request)
	signedHeaders, signedHeadersVal := sign.SignedHeaders(headers).PickOut(headerContentType, headerHost)

	payload := sign.SHA256Hex(body)
	payload = joinLines(c.httpMethod, c.httpURI, c.httpQuery, signedHeadersVal, signedHeaders, payload)
	payload = sign.SHA256Hex([]byte(payload))
	payload = joinLines(c.algorithm, timestamp, scope, payload)
	payload = sign.Sign(payload, c.secretKey, action.Service(), date)
	payload = fmt.Sprintf(authorizeTpl, c.algorithm, scope, signedHeaders, payload)
	return payload
}

func joinLines(lines ...string) string {
	return strings.Join(lines, eol)
}
