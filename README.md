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
| `Page`                 | Creates a new page |
| `--`                   | Inserts a horizontal rule and resets columns |

## Editing

The `/edit` page allows updating the entire bookmark file.
Each category heading on the index page now also includes a small
`edit` link that opens `/editCategory`. This page shows only the selected
category text and saves changes back to your bookmarks without touching
other sections. Edits check the file's SHA so you'll get an error if it
changed while you were editing.

![img.png](media/img.png)

![img_1.png](media/img_1.png)

![img_2.png](media/img_2.png)

# How to setup for yourself

You can run this yourself. There is a docker version available under my github packages. There are also precompiled versions
under the releases section of this git repo: https://github.com/arran4/StartHere/releases

You will require 3 environment arguments:

| Arg | Value                                                                                            |
| --- |--------------------------------------------------------------------------------------------------|
| `OAUTH2_CLIENT_ID` | The Client ID generated from setting up Oauth2 on github: https://github.com/settings/developers |
| `OAUTH2_SECRET` | Secret ID  generated from setting up Oauth2 on github: https://github.com/settings/developers |
| `EXTERNAL_URL` | The fully qualified URL that it is to accept connections from. Ie `http://localhost:8080`        |

## Oauth2 setup

Visit: https://github.com/settings/developers

Create an application, call it what ever you like. Set the Callback URL what ever you put in `EXTERNAL_URL` and add: 
`/oauth2Callback` to the end, ie if you entered: `http://localhost:8080` it should be: `http://localhost:8080/oauth2Callback`

Upload `logo.png` for the logo.

Generate a secret key and use it for the environment variables with the Client Id.

