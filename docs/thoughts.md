# Thoughts

This document is a space for brainstorming, capturing ideas, asking questions, and exploring possibilities. Use it freely for any thoughts that don't yet warrant a formal Design Record. When an idea matures into a concrete decision, create a DR in `design/design-records/`.

- What about adding a `webctl serve` command that runs a web server? Support hot reloading on file change? 127.0.0.1 by default but have the option of serving to the network?

- Can all commands work by just matching the minimal characters? nav should navigate etc.
- Can we have the navigate command automatically add the http(s) to the address? See the ../snag project for an example of this.
- Increase the page navigation timeout to 60 seconds.
- The `html` command does not pretty print the html in the file. SHould it? I don't remember.
- After navigate, the REPL terminal still shows the old address/title.
