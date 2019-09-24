# docker-registry
cli tool to manage [docker registry v2](https://docs.docker.com/registry/spec/api/).

```bash
$ docker-registry 
manage images in the docker registry v2. (https://docs.docker.com/registry/spec/api/)

Usage:
  docker-registry [command]

Available Commands:
  ls          List repositories or images in the registry.
  rm          Delete repository or images in the registry.
  manifest    Show manifest of image in the registry.
  blob        Display blob content of repository in the registry. Redirect should be used for binary object.
  help        Help about any command

Flags:
  -h, --help   help for docker-registry

Use "docker-registry [command] --help" for more information about a command.
```

