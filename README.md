Redis Sync
=========

An redis data manager with git version control

![Demo](https://github.com/gogap/redis_sync/blob/master/demo/redis_sync_demo.gif)

## What is redis_sync

redis sync is a tool for maintenance redis data, we may use redis `*.rdb` to backup or restore redis data, and we also could use mysql to import, but it was too heavy to use while we use redis to store some config, it was hard to maintenance, and we did not known what changed between every changes, we may use `phpRedisAdmin` and other tools for manage data, but it was so hard to build and deploy, and also did not have version control.

redis sync worked with `git` , and the data format is `json`,  so we could use command `git diff versionA...versionB` to known every changes, and we could use command `ls` to get `key` list, because the key is folder name. 

so, we could manage keys and data by sublime or other editor, just drag the data root folder to editor, and we could easily to modify data.


## Install

Require:
- Golang Installed

make sure you append the `GOPATH/bin` to `PATH` as following:
```bash
export PATH=$PATH:$GOPATH/bin
```

```bash
> go get -u github.com/gogap/redis_sync
> go install github.com/gogap/redis_sync
```

## Usage
### commands
```bash
> redis_sync

NAME:
   redis_sync - A new cli application

USAGE:
   redis_sync [global options] command [command options] [arguments...]

VERSION:
   0.0.0

AUTHOR(S):


COMMANDS:
   push		Push local config's to redis
   pull		Pull config's from redis
   commit	Record changes to the repository
   init		Init current dir for sync data
   status	Show the working tree status
   diff		Show changes between commits, commit and working tree, etc
   help, h	Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h		show help
   --version, -v	print the version
```

### inital redis data work dir

```bash
> redis_sync init
```

the current dir will got `redis_sync.conf`,  `.redis_sync` dir and a `token` file will generated, it contain a sync `token`, while we `push` or `pull` redis data, it will validate the `token`, if not match, it will stop sync data for avoid data pollution, and 

#### config
```json
{
    "redis": {
        "address": "127.0.0.1:6379",
        "db": 0,
        "auth": ""
    },
    "value_types": [
        {
            "key": "HKEY",
            "field": "Field",
            "type": "string"
        },
        {
            "key": "KEY",
            "type": "int"
        }
    ]
}
```
we need configure the redis `address`, `db` and `auth` info, so well could sync with the redis server, the `value_types` is used for data value define, because of redis's data always a string type, while we storage the data into file, we need known what the value's type actually is, and convert it to json object type. 

#### add key-value data (string)

add a `data` file at data dir's root as follow:

`./data`
```
{
	"key1":"abc",
	"key2":"efg"
}
```

```bash
> redis_sync commit -m "add key1 and key2"
> redis_sync push
```

we could check data by `redis-cli`

```bash
> redis-cli -h 127.0.0.1 -n 0 get key1
"abc"
> redis-cli -h 127.0.0.1 -n 0 get key2
"efg"
```

#### add key-field-value data (hash)

- create a folder that named with your key
- add a `data` file to that folder

e.g.: the key is `hello`

```bash
> mkdir hello
> touch ./hello/data
```

`./hello/data`
```
{
	"world":"gogap"
}
```

```bash
> redis_sync commit -m "add hash data of hello:world"
> redis_sync push
```

we could check data by `redis-cli`

```bash
> redis-cli -h 127.0.0.1 -n 0 hget hello world
"gogap"
```

