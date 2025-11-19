pbjq
===

This is a gojq fork which allows to filter protobuf serialized data.

See the original gojq docs here: https://github.com/itchyny/gojq.

See the jq manual (almost compatible api) here: https://jqlang.org/manual/.

Added features
---

- cli: `--argpb <name> <jsonspec>` / `--argpbfile <name> <file>`:

    Add protobuf message parser under `$<name>`.

    `<jsonspec>` should specify the parser options, e.g.:

    ```
    {
        "message": "mypackage.MyMessage",
        "proto_paths": ["mymsg.proto"],
        "import_paths": ["myimports/"]
    }
    ```

    For `--argpbfile` the paths can be relative to the spec file dir.

- func: `pb_decode(<parser>)`

    Will output the protojson version of the binary input parsed by a given parser.

Examples
---

```
$ cat pbspec.json

{
    "message": "mypackage.MyMessage",
    "proto_paths": ["mymsg.proto"],
    "import_paths": ["myimports/"]
}

$ cat ./mybinmsg.bin | pbjq -sR \
    --argpb myparser "$(cat pbspec.json)" \
    'pb_decode($myparser)'

<protojson data>

# OR base64 data:
$ cat ./mybinmsg.bin | base64 -w0 | pbjq -R \
    --argpb myparser "$(cat pbspec.json)" \
    '@base64d | pb_decode($myparser)'

<protojson data>
```

Dependencies
---
The tool requires protoc binary to be available.
