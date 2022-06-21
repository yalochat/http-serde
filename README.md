# HTTP Request Serializer/Deserializer

Golang package to serialize and deserialize http requests.

## Serialize an HTTP Request

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/yalochat/http-serde"
)

func main() {
    req, err := http.NewRequest(http.MethodGet, "your.url", nil)
    if err != nil {
        // handle error
    }
    serializer := http_serde.New()
    bytes, err := serializer.Serialize(req)
    if err != nil {
        // handle error
    }
    fmt.Println(string(bytes))
}
```

## Deserialize an HTTP Request

```go
package main

import (
	"io/ioutil"

	"github.com/yalochat/http-serde"
)

func main() {
    bytes, err := ioutil.ReadFile("stored_request.txt")
    if err != nil {
        // handle error
    }
    deserializer := http_serde.New()
    req, err := deserializer.Deserialize(bytes)
    if err != nil {
        // handle error
    }
    // do something with req
}
```