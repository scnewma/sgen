# SGen

`sgen` is a tool that allows you to source data and use it to generate
different outputs. It is intentionally simple, giving you just enough
capabilities to be able to integrate with other common command line tools to
create powerful workflows.

## Installation

```
go install github.com/scnewma/sgen@latest
```

## Configuration

`sgen` requires a configuration file to exist in `$HOME/.config/sgen/config.yaml`.

See [schema.cue](./schema.cue) for how to setup the configuration file.

## How I use it

I use `sgen` as a data source to add smart fuzzy search capabilities to
otherwise dumb commands without introducing latency waiting for external APIs.

For example, the list of GitHub repositories in your org don't change very
often, so you can easily cache the names (and metadata) for all of these
repositories. Then you can use that single datasource to power the following:
* fuzzy search/open a github repository
* fuzzy search for ssh url to git clone a repository
* open an external integrated service based on a github repository name (circleci, sourcegraph, etc.)

I drive these types of interactions through shell functions/aliases, and [alfred](https://www.alfredapp.com/).

### GitHub

Here is a real configuration and shell functions to utilize the configuration. Requires the [gh cli](https://github.com/cli/cli).

```
sources:
  - name: gh
    default_template: "{{ .nameWithOwner }}"
    type: command
    command: |
      gh repo list hashicorp --limit 5000 --json nameWithOwner,url,sshUrl
    
```

```
# "github clone": clone based on fuzzy searching the cached github repository
# list
function ghc {
    SELECTION="$(sgen gh --template '{{.nameWithOwner}} {{.sshUrl}}' \
        | fzf --with-nth=1)"
    [ -z "$SELECTION" ] && return
    NAME="$(echo "$SELECTION" | cut -d' ' -f1)"
    URL="$(echo "$SELECTION" | cut -d' ' -f2)"
    git clone "$URL" "$HOME/dev/$NAME"
}

# "github open": fuzzy find a github repository and open it in github.com
function gho {
    SELECTION="$(sgen gh --template '{{.nameWithOwner}}' | fzf)"
    [ -z "$SELECTION" ] && return
    gh repo view "$SELECTION" --web
}
```

I have this integrated in an Alfred workflow via the following script filter:

```
export PATH="$HOME/go/bin:$PATH"

sgen gh --template '{{.name}} {{(dict "title" .nameWithOwner "arg" .url "autocomplete" .nameWithOwner) | toJson}}' \
    | fzf --filter "$1" \
    | awk '{print $2}' \
    | jq --slurp '{items:.}'
```
