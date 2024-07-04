source "file" "names-no-default-file" {
  path = "${sgen.directory}/names.json"
}

source "command" "names-no-default-command" {
  command = "cat ${sgen.directory}/names.json"
}

source "file" "names-file" {
  path = "${sgen.directory}/names.json"

  template {
    name = "default"
    value = "{{.name | upper}}"
  }

  template {
    name = "bulleted"
    value = "* {{.name}}"
  }
}

source "command" "names-command" {
  command = "cat ${sgen.directory}/names.json"

  template {
    name = "default"
    value = "{{.name | upper}}"
  }

  template {
    name = "bulleted"
    value = "* {{.name}}"
  }
}
