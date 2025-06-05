# gobookmarks

![logo.png](logo.png)

The purpose of the site is to display a list of links for you to see every time you open your browser. I have tried to 
move as much of the work into the app as possible with minimal effort but you will need to use github occasionally.

![img_4.png](media/img_4.png)

This project is a converstion of a project: [goa4web-bookmarks](https://github.com/arran4/goa4web-bookmarks) to remove
the SQL and replace it with Github. Which itself is extract from [goa4web](https://github.com/arran4/goa4web), which is
a Go port of a C++ site I made in 2003. Ported using ChatGPT: [a4web](https://github.com/arran4/a4web). It's all been
minimally modified and as close to the original as I could get but with the changes I required. I made modifications to
this because [StartHere](https://github.com/arran4/StartHere) my SPA version using modern tech failed because of Github
Oauth2 restrictions on SPA sites. You can read more about this here: https://arranubels.substack.com/p/quicklinks

# How to use

1. Create a (private or public doesn't matter) repo in github under your user name called: "MyBookmarks"
2. Create 1 file in it called `bookmarks.txt` Put the following content (or anything you want really):
```text
Category: Search
http://www.google.com.au Google
Category: Wikies
http://en.wikipedia.org/wiki/Main_Page Wikipedia
http://mathworld.wolfram.com/ Math World
http://gentoo-wiki.com/Main_Page Gentoo-wiki
```
Ie:
![img_3.png](media/img_3.png)
3. Goto the URL this app is deployed at, your private instance or: https://bookmarks.arran.net.au
4. Enjoy

## File format

It's a basic file format. Every command must be on it's own line empty lines are ignored.

| Code                   | Meaning                                                      |
|------------------------|--------------------------------------------------------------|
| `Category: <category>` | Will create a category title.                                |
| `<Link>`               | Will create a link to `<Link>` with the display name `<Link>` |
| `<Link> <Name>`        | Will create a link to `<Link>` with the display name `<Name>` |
| `Column`               | Will create a column                                         |

![img.png](media/img.png)

![img_1.png](media/img_1.png)

![img_2.png](media/img_2.png)

# How to setup for yourself

You can run this yourself. There is a docker version available under my github packages. There are also precompiled versions
under the releases section of this git repo: https://github.com/arran4/StartHere/releases

You will require 3 environment arguments:

| Arg | Value                                                                                            |
| --- |--------------------------------------------------------------------------------------------------|
| `OAUTH2_CLIENT_ID` | The Client ID from registering an OAuth application with your provider |
| `OAUTH2_SECRET` | Secret value from the same OAuth application |
| `EXTERNAL_URL` | The fully qualified URL that it is to accept connections from. Ie `http://localhost:8080`        |
| `GBM_PROVIDER` | Either `github` or `gitlab` to select the backend. Must be set |
| `GBM_GITLAB_BASE_URL` | Base API URL for GitLab when using a self-hosted instance |
| `GBM_NAMESPACE` | Optional suffix used when generating the bookmarks repository name |
| `GBM_COMMIT_EMAIL` | Email address used for git commits |

## OAuth2 setup

Create an OAuth application with the provider you intend to use.

### GitHub

1. Visit <https://github.com/settings/developers> and choose **New OAuth App**.
2. Set the callback URL to your `EXTERNAL_URL` followed by `/oauth2Callback` (for
   example `http://localhost:8080/oauth2Callback`).
3. Record the generated **Client ID** and **Client Secret** and use them as the
   `OAUTH2_CLIENT_ID` and `OAUTH2_SECRET` environment variables.

### GitLab

1. Open `<your gitlab host>/-/profile/applications` (for GitLab.com use
   <https://gitlab.com/-/profile/applications>).
2. Enter a name, use the same callback URL as above, and enable the `api` scope.
3. Save the application and use the Application ID and Secret as
   `OAUTH2_CLIENT_ID` and `OAUTH2_SECRET`.

When you log in through the web interface gobookmarks uses this application to
acquire an OAuth2 token. You can also reuse the application to obtain tokens for
command-line access if needed.

### Selecting the git provider

Select the backend with `GBM_PROVIDER` or the `--provider` flag. Supported
values are `github` and `gitlab`. If you use a self-hosted GitLab instance,
set `GBM_GITLAB_BASE_URL` to the base API endpoint
(e.g. `https://gitlab.example.com/api/v4`).

### Build tags and version

Provider implementations live behind build tags. Use `go build` with
`-tags exclude_github` or `exclude_gitlab` to omit a backend.
Run the binary with `-version` to print compiled capabilities and exit.

