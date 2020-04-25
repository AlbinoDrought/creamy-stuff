# Creamy Stuff

<a href="https://hub.docker.com/r/albinodrought/creamy-stuff">
<img alt="albinodrought/creamy-stuff Docker Pulls" src="https://img.shields.io/docker/pulls/albinodrought/creamy-stuff">
</a>
<a href="https://github.com/AlbinoDrought/creamy-stuff/blob/master/LICENSE"><img alt="AGPL-3.0 License" src="https://img.shields.io/github/license/AlbinoDrought/creamy-stuff"></a>

Generate shareable links to local files.

## Features

- Share public or password-protected links to files or folders
- Track link downloads
- Automatically disable links after an amount of time
- Automatically disable links after an amount of downloads

## Usage

Right now there are no configuration options.

### With Docker

```sh
docker run --rm -it \
    -v $(pwd)/foo/bar:/data \
    albinodrought/creamy-stuff
```

### Without Docker

```sh
./creamy-stuff
```

## Building

### With Docker

```sh
make image
```

### Without Docker

```sh
make build
```
