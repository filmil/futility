
## Handling repository

### Running tests

```
bazel test //...
```

The project must test without errors.

### Building the project

```
bazel build //...
```

### Updating the module registry

```
bazel mod tidy
```

### Run the go compiler directly

Use this command to run the hermetic go compiler, instead of the usual `go`
command.

Replace `<args>` with the list of arguments to give the go compiler.

```
bazel run @rules_go//go -- <args>
```

### Update the bazel BUILD files


```
bazel run //:gazelle
```

### Handle a newly added dependency

This is a little bit involved because we draw dependencies from the `go.mod` file,
meaning we need to run go to update that first. We then run bazel tidy process
to tidy up MODULE.bazel. We can then run gazelle so that gazelle will update the
BUILD files.

```
bazel run @rules_go//go -- mod tidy -v
bazel mod tidy
bazel run //:gazelle
```

