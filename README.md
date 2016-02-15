# webcrawler

This is a parallel webcrawler written in go.

## Usage
The crawler takes two arguments, a first URL to crawl and the number of links find.

`webcrawler http://reddit.com 10000`

## Caveats

The webcrawler uses a buffered channel to push links back into the goroutines that find more links.  If that buffer is full links will go into an overflow slice.  If the overflow slice grows beyond two times the channel buffer size links will be forgotten.  In the real world you could throw this into a database or write to some other process to add in later, or process on a later run.

The web crawler doesn't check if a link has been visited before processing.  This could also be implemented in memory with a map or in a database with a time stamp so we could know whether to crawl it again or not.

The web crawler doesn't crawl relative links.  This could be implemented with not too much additional effort.
