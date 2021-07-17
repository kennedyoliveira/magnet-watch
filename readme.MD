# Magnet Watch
Simple utility for monitoring magnet torrents, adding them to transmission and renaming it.

## Building
You can build it from source using the following command:

```shell
make build
```

It will generate a `magnet-watch` binary file in the `dist` directory.

## Configurations

```
Usage of dist/magnet-watch:
  -debug
        Log more information for debug purposes
  -directory string
        Directory to watch, will watch the current directory if not provided (default ".")
  -fileNamePattern string
        The pattern to match file names, by default it look for files with extension ".magnet" (default ".*\\.magnet$")
  -help
        Print the help message
  -process-files-on-start
        Process the files when the application starts (default true)
  -transmission-api-path string
        The path of the transmission API (default "/transmission/rpc")
  -transmission-password string
        The password to authenticate in transmission API
  -transmission-url string
        The URL of transmission (default "http://localhost:9091")
  -transmission-username string
        The username to authenticate in transmission API
```

## Init script
Check if there is a script for your system in `init-scripts` folder, but this app is not a well behaved daemon so you will need
something to start it as a child, create pid etc.