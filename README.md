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

Create a new organization in your Gitea instance to store your `ggv` vendor repos.
This is necessary not just to keep your Gitea clean, but also to tell `ggv` what repos to ignore as inputs
(you don't want to vendor your vendor repos!).

> [!WARNING] 
>It is recommended to create a user that only has access to that one organization and use said new user to generate the below API key.

Create a new API key under `User -> Settings -> Applications -> Generate New Token`.

This key needs `repository` "Read and Write" access. Everything else can be left to "No Access".

## Usage
`echo 'EXAMPLE_API_TOKEN' | ggv https://gitea.example.com myOutputOrganization`

I recommend setting up a cron job or some form of automation for this.

> [!WARNING]
>This tool assumes your default branch is what you want to vendor.
