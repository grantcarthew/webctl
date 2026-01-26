# Wait Strategies

Ready command synchronization modes.

## Page Load Mode

```
webctl ready
webctl ready --timeout 30s
```

## Selector Mode

```
webctl ready "#dashboard"
webctl ready ".content-loaded"
webctl ready "[data-loaded=true]"
webctl ready "button.submit:enabled"
```

## Network Idle Mode

```
webctl ready --network-idle
webctl ready --network-idle --timeout 120s
```

## Eval Mode

```
webctl ready --eval "document.readyState === 'complete'"
webctl ready --eval "window.appReady === true"
webctl ready --eval "document.querySelector('.error') === null"
webctl ready --eval "window.app && window.app.initialized"
```

## Chaining Waits

```
webctl ready
webctl ready --network-idle
webctl ready --eval "window.dataLoaded"
```

## Common Patterns

```
webctl navigate example.com
webctl ready

webctl click ".nav-dashboard"
webctl ready "#dashboard-content"

webctl click "#submit"
webctl ready ".success-message"

webctl click "#load-data"
webctl ready --network-idle

webctl scroll "#load-more"
webctl ready --network-idle
webctl ready ".new-items"
```
