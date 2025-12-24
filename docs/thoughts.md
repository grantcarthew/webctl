# Thoughts

This document is a space for brainstorming, capturing ideas, asking questions, and exploring possibilities. Use it freely for any thoughts that don't yet warrant a formal Design Record. When an idea matures into a concrete decision, create a DR in `design/design-records/`.

- What about adding a `webctl serve` command that runs a web server? Support hot reloading on file change? 127.0.0.1 by default but have the option of serving to the network?

- Can all commands work by just matching the minimal characters? nav should navigate etc.
- Can we have the navigate command automatically add the http(s) to the address? See the ../snag project for an example of this.
- Increase the page navigation timeout to 60 seconds.
- The `html` command does not pretty print the html in the file. SHould it? I don't remember.
- After navigate, the REPL terminal still shows the old address/title.
- We should have a `find` command to find an element on the page.
- The default timeout needs to be extended to 60 seconds or more (abc.net.au times out at 30 sec).
- What about color for the cookies?
- With output commands, we should check if the page is ready, if not, stderr a message saying "waiting for ready" or the like.
- tesla.com -> html failed `Error: failed to get window: request timed out: context deadline exceeded`, then a reload hung. The site has a "Select region" dialog.
- What about CSS?
