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
webctl network --find "error"
webctl network --head 10
webctl network --tail 20
webctl network --range 5-15
webctl network --raw
webctl network save
webctl network save ./requests.json
webctl network save ./output/
```

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
