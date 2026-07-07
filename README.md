# go-gitea-vendor
Want to vendor all your public Go projects hosted on your personal Gitea instance?
Don't want to bloat your git history and exhaust your storage?
Use `go-gitea-vendor` to maintain separate, shallow-clone repos that hold only the latest commit and vendored dependencies for each of your Go projects!

## Usage
`echo 'EXAMPLE_API_TOKEN' | go-gitea-vendor https://gitea.example.com <username> [output organization]`

I recommend setting up a chron job or some form of automation for this. My deployment involves running this on my Gitea mirror (this server mirrors my main Gitea instance as well as my GitHub) weekly using a chron job, then my main Gitea instance mirrors the output repos from this tool.
