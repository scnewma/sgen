source "command" "gh" {
  command = "gh repo list --json nameWithOwner"
}

source "command" "gh_w_template" {
  default_template = "{{.nameWithOwner}}"
  command = "gh repo list --json nameWithOwner"
}

source "file" "static" {
  path = "/data.json"
}

source "file" "static_w_template" {
  default_template = "{{.nameWithOwner}}"
  path = "/data.json"
}
