# git-now-playing

`git-now-playing` is an attempt to bring some of the panache of the early
aughts' AIM away messages to your git commits, by including what you're
currently listening to when you write a git commit.

There are two ways to do this. By default, the binary just writes properly
formatted track info to standard output, making it easy to integrate into the
`prepare-commit-msg` hook in git. If the `--daemon` flag is passed before the
config file, however, it runs as a background process and checks in with
Spotify and/or Plex every ten seconds to find out what you're listening to, and
updating a specified file with that information. You can then use this file as
your git commit template.

## Install

There aren't binary releases yet, sorry. I'm lazy and this was a 12 hour
nerdsnipe gone awry. So you're gonna need to run `go build` or `go install`
yourself. `go install paddy.dev/git-now-playing` should, in theory, do it. Good
luck!

## Configure

You must use a configuration file when running `git-now-playing`. You also need
to use [Vault](https://vaultproject.io). Good luck!

Configuration is done mostly through an HCL file:

```hcl
vault {
  address = "https://my.vault.server:8200/"
  mount_path = "kv_v2_secrets_engine_path"
}

# this block is optional
# if not set, output when running as a daemon gets written to
# $HOME/.config/gitmessage
output {
  path = "/path/where/file/containing/now/playing/track/should/be/written.txt"
}

# this block is optional
# also, you can specify it as many times as you want
# git-now-playing will ask all of them what you're playing
plex {
  server = "https://my.plex.server:32400/"
  
  # this bit is optional and will default to "plex"
  # it's the key inside the kv v2 vault in vault that contains
  # your plex token
  vault_path = "plex"

  # this bit is optional. It's the names of the Plex users that should be
  # considered "you" if multiple people use this Plex server.
  users = ["me", "otherme"]
}

# this block is optional
# if it's excluded, spotify won't be checked
spotify {
  # this attribute is optional, and will default to "spotify"
  # it's the key inside the kv v2 vault in vault that contains your spotify
  # token, client ID, and client secret
  vault_path = "spotify"

  # this attribute is optional, and will default to "0.0.0.0:8765". It's the IP
  # and port you want the web server git-now-playing needs to temporarily stand
  # up to receive the Spotify authorization callback on to listen on
  auth_callback_addr = "0.0.0.0:8765"
}
```

You'll also need to set up Vault. As mentioned, you're gonna want a [kv secrets
engine (v2)](https://www.vaultproject.io/api/secret/kv/kv-v2) in Vault set up
to hold the sensitive credentials, because writing secrets to config files
makes me anxious.

If you want to grant a Vault token that has the least possible access (you
should!) you can use this policy:

```hcl
path "git-now-playing/data/*" {
  capabilities = ["read", "patch"]
}
```

You're going to want to set up the following credentials:

(Note: for all these examples, we're using `git-now-playing` as the
`mount_path` of your Vault secrets engine. Replace it with whatever you're
using. We're also using the default `vault_path`s for each service. Replace
them with whatever you're using if you're not using the defaults.)

### Spotify

```sh
$ vault kv put git-now-playing/spotify client_id=$SPOTIFY_CLIENT_ID client_secret=$SPOTIFY_CLIENT_SECRET redirect_url=$SPOTIFY_REDIRECT_URL
```

You're gonna need to set up a Spotify app to do this bit. You can do that
[here](https://developer.spotify.com/dashboard/applications).
`$SPOTIFY_CLIENT_ID` and `$SPOTIFY_CLIENT_ID` are on the main application page
after you create it. For `$SPOTIFY_REDIRECT_URL`, you're going to need to add a
callback URL to your app. You gotta click "Edit Settings" to do this. Make the
callback URL `http://localhost:8765/` or whatever will reach the
`auth_callback_addr` configured in the `spotify` block of your HCL config.

The first time you run `git-now-playing`, it's gonna ask you to authorize your
application. Click the link and it'll take care of the rest. After this
happens, the token will be stored in Vault, and _should_ manage itself. Sorta.
Though now that I think about it, I don't think I ever wrote the bit where it
updates the token after it gets refreshed. Whoooops. Probably should do that.

### Plex

This bit is easier. You just need to get your hands on [a Plex
token](https://support.plex.tv/articles/204059436-finding-an-authentication-token-x-plex-token/)
and then run

```sh
$ vault kv put git-now-playing/plex token=$PLEX_TOKEN
```

## Running

### Git Hook

Set up a git `prepare-commit-msg` hook that will run this program and prepend
its standard output to whatever the contents of the commit file are. A sample
is found below. You need to pass in a `VAULT_TOKEN` environment variable with
the Vault token that `git-now-playing` can use to read and update secrets with.

#### Setting Up Your Commit Hook

To set up a git hook for this, modify the `.git/hooks/prepare-commit-msg` file
to look something like this:

```bash
#!/bin/bash
COMMIT_MSG_FILE=$1
NOW_PLAYING=$(VAULT_TOKEN="{vault token here}" /path/to/git-now-playing /path/to/git-now-playing.hcl)
COMMIT_CONTENTS=$(cat $COMMIT_MSG_FILE)

if [[ ! $COMMIT_CONTENTS =~ .*"${NOW_PLAYING}".* ]]; then
	if [[ $COMMIT_CONTENTS =~ ^\n.* ]]; then
		echo "${NOW_PLAYING}${COMMIT_CONTENTS}" > $COMMIT_MSG_FILE
	else
		echo "${COMMIT_CONTENTS}${NOW_PLAYING}" > $COMMIT_MSG_FILE
	fi
fi
```

If you want to use this hook in all of your repositories and you're using git
2.9 or later, you can run `git config --global core.hooksPath
/path/to/central/hook/location` and then put that `prepare-commit-msg` file in
that directory, and it will be used for all your git repos.

### Daemon Mode

Run this program as a background process. I use systemd. This bit is left as an
exercise for the reader for now. You need to pass in a `VAULT_TOKEN`
environment variable with the Vault token that `git-now-playing` can use to
read and update secrets with. So running it looks like this:

```sh
$ VAULT_TOKEN={vault token here} /path/to/git-now-playing --daemon /path/to/config.hcl
```

#### Setting Up Your Commit Template

To set up the output as the default template for your git commit messages, run:

```sh
$ git config --global commit.template ~/.config/gitmessage
```

If you used the `output` block in your HCL config, use the path there instead
of `~/.config/gitmessage`.

## License

This software is licensed under the MIT license, and I'd like to direct your
attention to the bits about no liability or warranty. This software is a bad
idea that I ran with and I think you probably shouldn't use it because it's
deeply cursed.
