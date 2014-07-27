# swift-cli #

swift-cli provides pipelined streaming access to OpenStack Swift

It is designed for high speed transfer of large objects into and out of OpenStack Swift. Streaming support allows for usage like:

```
  $ tar -czf - <my_dir/> | scli put <container>/<object>    
  $ scli get <container>/<object> | tar -zx
```

## Installation ##

swift-cli is written in Go and requires a Go installation. It can be installed with `go get` to download and compile it from source. To install the command-line tool, `scli`:

    $ go get github.com/fanatic/swift-cli
    $ go build -o scli github.com/fanatic/swift-cli

## Usage: ##

 ```
  version                   Print version information and quit
  ls [container[/object]]   list containers or objects
  put container[/object]    upload (put) an object
  get container[/object]    download (get) an object
  delete container[/object] delete an object
  help [command]            Help about any command

 Available Flags:
  -D, --debug=false: Enable debug mode
```

 Set Swift keys as environment Variables:

```
  $ export ST_AUTH=http://localhost:8080/auth/v1.0
  $ export ST_USER=test:tester
  $ export ST_KEY=testing

```

## Odds and Ends ##
This is a WIP that does one thing relatively well: managing dynamic large objects in swift.  Currently it only supports stdin and stdout.

Borrows upload logic from rlmcpherson's s3gof3r (https://github.com/rlmcpherson/s3gof3r)

Leverages the well-written golang swift library (github.com/ncw/swift)