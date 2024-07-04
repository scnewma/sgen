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

`sgen` requires a configuration file to exist in `$HOME/.config/sgen/config.hcl`.

### Source Blocks

`sgen` requires you to configure `source` blocks in order to know how to load
data to generate output.

#### Syntax

```
source "command" "gh" {
    command = "gh repo list --json nameWithOwner"

    template {
        name = "default"
        value = "{{.nameWithOwner}}"
    }
}
```

A `source` block requires a type (i.e. `"command"`) with a name (i.e. `"gh"`).
The name of the source is used as input on the command line in order to select
which sources should be consulted in order to generate output. All sources must
have unique names.

The data format of every source MUST be an array of objects, i.e. `list(map)`.
Unless otherwise specified by the specific source type, this is expected to be
in JSON format. To be super explicit, this is valid output of a source:

```
[
    { "name": "Bob" },
    { "name": "Alice" }
]
```

#### Source Types

##### Common Properties

All sources can specify a repeatable `template` block. This block allows you to
define commonly used templates in your configuration file and reference them by
name on the command line. Specifying a template named `default` will override
the default output for the source when `--template` is not specified. The
template is in Go Template syntax.

Example:

```
template {
    name = "default"
    value = "{{.name}}"
}
```

##### source "command"

Execute an external command in order to load data. The command's stdout will be
used as the data source.

Example:

```
source "command" "gh" {
    command = "gh repo list --json nameWithOwner"
}
```

Properties:

* `command` - The full command to execute. By default the command is executed
  directly, but you can access shell features by prefixing the command with
  `!` (i.e. `!gh repo list --json name | jq '.name'`)

##### source "file"

The `file` source loads data from an existing file on disk. Currently `json`
and `yaml` formats are automatically detected (via file ext).

Example:

```
source "file" "gh" {
    path = "/data.json"
}
```

Properties:

* `path` - Full path to the file on disk to load. Must end in one of the
  following extensions: `.json`, `.yaml`, `.yml`.

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
source "command" "gh" {
  command = "gh repo list hashicorp --limit 5000 --json nameWithOwner,url,sshUrl"

  template {
    name = "default"
    value = "{{.nameWithOwner}}"
  }
}
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
