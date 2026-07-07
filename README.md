# go-gitea-vendor
Want to vendor all your Go projects on your personal Gitea instance?
Don't want to bloat your git history?
Use `go-gitea-vendor` to maintain a separate repo that is only the latest vendoring!

## Usage
`echo 'EXAMPLE_API_TOKEN' | go-gitea-vendor https://gitea.example.com`

I recommend setting up a chron job or some form of automation for this.
