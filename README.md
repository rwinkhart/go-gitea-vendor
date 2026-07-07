# go-gitea-vendor (ggv)
Want to vendor all your public Go projects hosted on your personal Gitea instance?
Don't want to bloat your git history and exhaust your storage?
Use `go-gitea-vendor` to maintain separate, shallow-clone repos that hold only the latest commit and vendored dependencies for each of your Go projects!

## Setup
In your gitea `app.ini`, add the following under `[repository]`:
```
ENABLE_PUSH_CREATE_USER = true
ENABLE_PUSH_CREATE_ORG = true
```

Optionally, create a new organization in your Gitea instance to store your `ggv` vendor repos.

> [!WARNING] 
>It is recommended to create a user that only has access to that one organization and use said new user to generate the below API key.

Create a new API key under `User -> Settings -> Applications -> Generate New Token`.

This key needs `repository` "Read and Write" access. Everything else can be left to "No Access".

## Usage
`echo 'EXAMPLE_API_TOKEN' | ggv https://gitea.example.com myOutputOrganization`

I recommend setting up a chron job or some form of automation for this. My deployment involves running this on my Gitea mirror (this server mirrors my main Gitea instance as well as my GitHub) weekly using a chron job, then my main Gitea instance mirrors the output repos from this tool.

> [!WARNING]
>This tool assumes your default branch is what you want to vendor.
