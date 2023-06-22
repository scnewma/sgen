// #Config describes the root document of the configuration file.
#Config: {
    sources: [...#Source]
}

// #Source defines how to retrieve data
#Source: {
    // every source must have a name, and each source must have a unique name.
    // you reference the name of a source when generating data, i.e.
    //
    //      sgen mysource
    name: string

    // most of the time you will be providing a template on the command line,
    // but if you don't you can specify default_template here and that template
    // will be used instead.
    // if neither default_template nor a `--template` is defined then sgen will
    // fallback to rendering all data as json
    default_template?: string
} & (#FileSource | #CommandSource)

// #FileSource is a source that loads data from a file on disk. The following
// data types are automatically detected (via file ext) and loaded:
// * json
// * yaml
//
// The format of the data file is described in #Data.
#FileSource: {
    type: "file"
    file: {
        path: string
    }
    ...
}

// #CommandSource is a source that executes a command in order to load data.
// The command's STDOUT will be parsed according to #Data.
#CommandSource: {
    type: "command"
    // commands are executed directly by default, but you can execute the
    // command in a shell by prefixing the command with a '!'
    //
    //      # THIS DOES NOT WORK
    //      gh repo list --json | jq '.name'
    //
    //      # DO THIS
    //      !gh repo list --json | jq '.name'
    command: string
    ...
}

// #Data defines the return format which all sources must return. Unless
// otherwise specified by the particular source, it is assumed that all sources
// return JSON.
// i.e. list(map)
#Data: [...{
    ...
}]
