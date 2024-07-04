source "command" "gh" {
  command = "gh repo list --json nameWithOwner"
}

source "command" "gh_w_template" {
  command = "gh repo list --json nameWithOwner"

  template {
    name = "default"
    value = "{{.nameWithOwner}}"
  }

  template {
    name = "name"
    value = "{{.name}}"
  }
}

source "file" "static" {
  path = "/data.json"
}

source "file" "static_w_template" {
  path = "/data.json"

  template {
    name = "default"
    value = "{{.nameWithOwner}}"
  }

  template {
    name = "name"
    value = "{{.name}}"
  }
}
