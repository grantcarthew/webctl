# Interaction Commands

Commands modify browser state or simulate user actions.

## click

```
webctl click "#submit"
webctl click ".btn-primary"
webctl click "button[type=submit]"
webctl click "[data-testid=login-btn]"
```

## type

```
webctl type "#username" "user@example.com"
webctl type "#password" "secret"
webctl type "#search" "query" --key Enter
webctl type "#email" "new@email.com" --clear
webctl type "#field1" "value" --key Tab
```

## key

```
webctl key Enter
webctl key Tab
webctl key Escape
webctl key ArrowDown
webctl key ArrowUp
webctl key Backspace
webctl key a --ctrl
webctl key a --meta
webctl key z --ctrl --shift
```

## select

```
webctl select "#country" "AU"
webctl select "select[name=language]" "en"
webctl select ".size-picker" "large"
webctl select "[data-testid=region]" "asia"
```

## scroll

```
webctl scroll "#footer"
webctl scroll ".next-section"
webctl scroll --to 0,0
webctl scroll --to 0,500
webctl scroll --by 0,100
webctl scroll --by 0,-100
```

## focus

```
webctl focus "#username"
webctl focus "input[type=text]"
webctl focus ".search-input"
```
