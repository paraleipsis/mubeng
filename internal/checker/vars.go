package checker

import (
	"fmt"
	"net/http"
	"time"
)

var (
	client *http.Client

	endpoint = "https://ipinfo.io/json"
	tgAPI    = "https://api.telegram.org"

	clientTimeout             = 5 * time.Second
	dialContextTimeout        = 5 * time.Second
	clientTLSHandshakeTimeout = 5 * time.Second
	clientRetryWaitTime       = 300 * time.Millisecond
	retryCount                = 3
	httpClientDebug           = true
)

var UnsuccessfulRequestError = fmt.Errorf("unsuccessful request")
var ProxyConnectError = fmt.Errorf("proxy connect error")
