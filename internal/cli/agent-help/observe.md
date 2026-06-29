# Observation Commands

Commands extract data from browser. Default output: stdout.

## html

```
webctl html
webctl html --select "#main"
webctl html --find "login"
webctl html --select "form" --find "password"
webctl html --raw
webctl html save
webctl html save ./page.html
webctl html save ./output/
```

## markdown

```
webctl markdown
webctl md
webctl markdown --select "#main"
webctl markdown --find "install"
webctl markdown save
webctl markdown save ./page.md
webctl markdown save ./output/
```

## css

```
webctl css
webctl css --select "h1"
webctl css --find "background"
webctl css --raw
webctl css save
webctl css save ./styles.css
webctl css save ./output/
webctl css computed "#header"
webctl css computed ".button"
webctl css get "#header" "background-color"
webctl css inline "[style]"
webctl css matched "#main"
```

## console

```
webctl console
webctl console --type error
webctl console --type warn
webctl console --find "undefined"
webctl console --head 10
webctl console --tail 20
webctl console --range 5-15
webctl console --raw
webctl console save
webctl console save ./logs.json
webctl console save ./output/
```

## network

```
webctl network
webctl network --status 4xx
webctl network --status 5xx
webctl network --status 200
webctl network --method POST
webctl network --type xhr
webctl network --type fetch
webctl network --url "api"
webctl network --mime "application/json"
webctl network --min-duration 1s
webctl network --min-size 1000
webctl network --failed
webctl network --headers
webctl network --find "error"
webctl network --head 10
webctl network --tail 20
webctl network --range 5-15
webctl network --raw
webctl network save
webctl network save ./requests.json
webctl network save ./output/
```

Request and response bodies. Each entry carries the outgoing request body as
requestBody and the response payload as responseBody when the request sent one
(POST, PUT, PATCH, and any request with a payload). GET requests have no request
body. Typical bodies (JSON, form-encoded) arrive inline; bodies larger than the
inline cap are fetched separately and still appear in output. Both bodies are
bounded by --max-body-size; a truncated request body sets requestBodyTruncated and
a truncated response body sets responseBodyTruncated. A binary response body is
saved to a file whose path appears as responseBodyPath. --find matches the request
body as well as the URL and response body, so a request can be located by its
payload.

NOTE: Multipart uploads are captured partially by design. Chrome supplies the form
fields and boundaries but omits the uploaded file contents, so requestBody holds the
partial body, not the files. requestBody is empty only when no data was sent.

Text view fields. The default text line is METHOD URL STATUS DURATIONms followed by
the resource type and the human-readable response size when present. A failed request
renders a FAILED token plus its reason instead of a status. Headers are omitted from
the default text view to keep it compact; pass --headers to print request and response
headers as indented lines in text mode. JSON output always carries the full entry,
including headers, mimeType, and statusText.

Transport and origin detail. A cached response is tagged on the main line with its
origin: (disk), (service-worker), or (prefetch). When captured, indented lines follow
each entry. A remote: line shows the contacted endpoint and negotiated protocol, the
connection id as conn:N (shared ids reveal HTTP/2 multiplexing and keep-alive reuse),
and a non-secure security state (insecure, neutral, unknown) when present; a secure
state is omitted. A timing: line shows per-phase latency in milliseconds (dns, connect,
tls, send, wait), dropping phases under half a millisecond. An initiator: line names
what triggered the request as type url:line for parser and script initiators; the
locationless other initiator is omitted. Every field is always present in --json.

## cookies

```
webctl cookies
webctl cookies --domain ".example.com"
webctl cookies --name "session"
webctl cookies --find "auth"
webctl cookies --raw
webctl cookies save
webctl cookies save ./cookies.json
webctl cookies save ./output/
webctl cookies set session abc123
webctl cookies set auth xyz --secure --httponly
webctl cookies delete session
```

## screenshot

```
webctl screenshot save
webctl screenshot save ./page.png
webctl screenshot save ./output/
webctl screenshot save --full-page
```

## eval

```
webctl eval "document.title"
webctl eval "window.location.href"
webctl eval "document.querySelector('#main').textContent"
webctl eval "JSON.stringify(window.appState)"
```
