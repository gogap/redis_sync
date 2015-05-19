package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/gogap/errors"
	"github.com/hoisie/redis"
)

type PushData struct {
	Key   string
	Field string
	Value string
}

var (
	viewDetails = false

	app = cli.NewApp()

	conf syncConfig
)

func main() {
	app.Commands = []cli.Command{
		commandPush(cmdPush),
		commandPull(cmdPull),
		commandCommit(cmdCommit),
		commandInit(cmdInit),
		commandStatus(cmdStatus),
		commandDiff(cmdDiff),
	}

	app.Run(os.Args)

}

func cmdStatus(c *cli.Context) {
	var err error

	defer func() {
		if err != nil {
			exitError(err)
		}
	}()

	if !checkIsSyncDir() {
		err = ERR_THE_CWD_IS_NOT_SYNC_DIR.New()
		return
	}

	repo := GitRepo{}

	if e := repo.Status(); e != nil {
		err = ERR_GET_REPO_STATUS_FAILED.New(errors.Params{"err": e})
		return
	} else {
		fmt.Println(string(repo.Output))
	}
}

func cmdDiff(c *cli.Context) {
	var err error

	defer func() {
		if err != nil {
			exitError(err)
		}
	}()

	if !checkIsSyncDir() {
		err = ERR_THE_CWD_IS_NOT_SYNC_DIR.New()
		return
	}

	repo := GitRepo{}

	if e := repo.Diff(); e != nil {
		err = ERR_GET_REPO_DIFF_FAILED.New(errors.Params{"err": e})
		return
	} else {
		fmt.Println(string(repo.Output))
	}
}

func cmdPush(c *cli.Context) {
	var err error

	defer func() {
		if err != nil {
			exitError(err)
		}
	}()

	if !checkIsSyncDir() {
		err = ERR_THE_CWD_IS_NOT_SYNC_DIR.New()
		return
	}

	viewDetails = c.Bool("v")

	configFile := c.String("config")

	if err = initalConfig(configFile); err != nil {
		return
	}

	errorContinue := c.Bool("contine")
	overWrite := c.Bool("overwrite")

	client := redis.Client{
		Addr:        conf.Redis.Address,
		Db:          conf.Redis.Db,
		Password:    conf.Redis.Auth,
		MaxPoolSize: 3,
	}

	workDir := ""

	if workDir, err = os.Getwd(); err != nil {
		err = ERR_GET_CWD_FAILED.New(errors.Params{"err": err})
		return
	}

	repo := GitRepo{}

	if !repo.IsClean() {
		err = ERR_COMMIT_CURRENT_WORKDIR_NOT_CLEAN.New()
		return
	}

	pushCache := []PushData{}

	fnWalk := func(path string, info os.FileInfo, e error) (err error) {
		if info.Name() != "data" {
			return
		}

		if info.IsDir() &&
			strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		datafile, _ := filepath.Rel(workDir, path)

		if datafile == "." {
			return
		}

		datafileDir := filepath.Dir(datafile)

		var data []byte
		if data, err = ioutil.ReadFile(datafile); err != nil {
			err = ERR_READ_DATAFILE_ERROR.New(errors.Params{"fileName": datafile, "err": err})
			return
		}

		dataKV := map[string]interface{}{}

		if e := json.Unmarshal(data, &dataKV); e != nil {
			err = ERR_PARSE_DATAFILE_ERROR.New(errors.Params{"fileName": datafile, "err": err})
			return
		}

		if datafileDir == "." {
			//SET
			for k, v := range dataKV {
				if strV, e := serializeObject(v); e != nil {
					err = ERR_COULD_NOT_CONV_VAL_TO_STRING.New(errors.Params{"key": k, "err": e})
					return
				} else {
					pushCache = append(pushCache, PushData{Key: k, Field: "", Value: strV})
				}

			}
		} else {
			//HSET
			for k, v := range dataKV {
				if strV, e := serializeObject(v); e != nil {
					err = ERR_COULD_NOT_CONV_VAL_TO_STRING.New(errors.Params{"key": k, "err": e})
					return
				} else {
					pushCache = append(pushCache, PushData{Key: datafileDir, Field: k, Value: strV})
				}
			}
		}
		return
	}

	if err = filepath.Walk(workDir, fnWalk); err != nil {
		return
	}

	total := len(pushCache)
	ignore := 0
	pushed := 0

	consoleReader := bufio.NewReader(os.Stdin)
	for _, data := range pushCache {
		exceptType := "string"
		if data.Field != "" {
			exceptType = "hash"
		}

		keyTypeMatchd := false
		actualKeyType := "none"

		if keyType, e := client.Type(data.Key); e != nil {
			err = ERR_GET_REDIS_KEY_TYPE_FAILED.New(errors.Params{"key": data.Key, "err": e})
			return
		} else {
			actualKeyType = keyType
		}

		keyTypeMatchd = exceptType == actualKeyType

		if !keyTypeMatchd && !overWrite && actualKeyType != "none" {
			fmt.Printf("The key: '%s' already exist, but the type is not '%s', do you want overwrite [y/N]: ", data.Key, exceptType)
			if line, e := consoleReader.ReadByte(); e != nil {
				err = ERR_READE_USER_INPUT_ERROR.New()
				return
			} else if line == 'n' || line == 'N' {
				continue
			} else if line == 'y' || line == 'Y' {
				if _, e := client.Del(data.Key); e != nil {
					err = ERR_DELETE_REDIS_KEY_FAILED.New(errors.Params{"key": data.Key, "err": e})
					return
				}
			} else {
				continue
			}
		}

		if keyTypeMatchd {
			if exceptType == "string" {
				if originV, e := client.Get(data.Key); e != nil {
					if !errorContinue {
						err = ERR_GET_REDIS_VALUE_ERROR.New(errors.Params{"key": data.Key, "err": e})
						return
					}
				} else if string(originV) == data.Value {
					if viewDetails {
						fmt.Printf("[IGNORE] key: '%s' already have value of '%s'\n", data.Key, data.Value)
					}
					ignore += 1
					continue
				} else if !overWrite {
					fmt.Printf("The key: '%s' already exist, and current value is '%s', do you want overwrite it to '%s' [y/N]: ", data.Key, string(originV), data.Value)
					if line, e := consoleReader.ReadByte(); e != nil {
						err = ERR_READE_USER_INPUT_ERROR.New()
						return
					} else if line == 'n' || line == 'N' {
						continue
					} else if line == 'y' || line == 'Y' {

					} else {
						continue
					}
				}
			} else {
				if exist, e := client.Hexists(data.Key, data.Field); e != nil {
					if !errorContinue {
						err = ERR_HGET_KEY_STATUS_ERROR.New(errors.Params{"key": data.Key, "field": data.Field, "err": e})
						return
					}
				} else if exist {
					if originV, e := client.Hget(data.Key, data.Field); e != nil {
						if !errorContinue {
							err = ERR_HGET_REDIS_VALUE_ERROR.New(errors.Params{"key": data.Key, "field": data.Field, "err": e})
							return
						}
					} else if string(originV) == data.Value {
						if viewDetails {
							fmt.Printf("[IGNORE] key: '%s', field: '%s', already have value of '%s'\n", data.Key, data.Field, data.Value)
						}
						ignore += 1
						continue
					} else if !overWrite {
						fmt.Printf("The key: '%s', field: '%s', already exist, and current value is '%s', do you want overwrite it to '%s' [y/N]: ", data.Key, data.Field, string(originV), data.Value)
						if line, e := consoleReader.ReadByte(); e != nil {
							err = ERR_READE_USER_INPUT_ERROR.New()
							return
						} else if line == 'n' || line == 'N' {
							continue
						} else if line == 'y' || line == 'Y' {

						} else {
							continue
						}
					}
				}
			}
		}

		if exceptType == "string" {
			if e := client.Set(data.Key, []byte(data.Value)); e != nil {
				err = ERR_SET_REDIS_DATA_ERROR.New(errors.Params{"key": data.Key, "value": data.Value, "err": e})
				return
			}

			pushed += 1
			if viewDetails {
				fmt.Printf("[SET]\t '%s' '%v' \n", data.Key, data.Value)
			}
		} else {
			if _, e := client.Hset(data.Key, data.Field, []byte(data.Value)); e != nil {
				err = ERR_HSET_REDIS_DATA_ERROR.New(errors.Params{"key": data.Key, "field": data.Field, "value": data.Value, "err": e})
				return
			}

			pushed += 1
			if viewDetails {
				fmt.Printf("[HSET]\t '%s' '%s' '%v' \n", data.Key, data.Field, data.Value)
			}
		}
	}
	fmt.Printf("ignored: %d, pushed: %d, total: %d\n", ignore, pushed, total)
}

func cmdCommit(c *cli.Context) {
	var err error

	defer func() {
		if err != nil {
			exitError(err)
		}
	}()

	if !checkIsSyncDir() {
		err = ERR_THE_CWD_IS_NOT_SYNC_DIR.New()
		return
	}

	message := c.String("m")

	if message == "" {
		err = ERR_COMMIT_MSG_NOT_INPUT.New()
		return
	}

	repo := GitRepo{}

	if e := repo.AddUntracked(); e != nil {
		err = ERR_ADD_UNTRACKED_FILES_TO_GIT_FAILED.New(errors.Params{"err": e})
		return
	}

	if e := repo.AddModified(); e != nil {
		err = ERR_ADD_MODIFIED_FILES_TO_GIT_FAILED.New(errors.Params{"err": e})
		return
	}

	if e := repo.Commit(message); e != nil {
		err = ERR_COMMIT_GIT_REPO_FAILED.New(errors.Params{"err": e})
		return
	}
}

func cmdPull(c *cli.Context) {
	var err error

	defer func() {
		if err != nil {
			exitError(err)
		}
	}()

	if !checkIsSyncDir() {
		err = ERR_THE_CWD_IS_NOT_SYNC_DIR.New()
		return
	}

	configFile := c.String("config")

	if err = initalConfig(configFile); err != nil {
		return
	}

	// client := redis.Client{
	// 	Addr:        conf.Redis.Address,
	// 	Db:          conf.Redis.Db,
	// 	Password:    conf.Redis.Auth,
	// 	MaxPoolSize: 3,
	// }
}

func cmdInit(c *cli.Context) {
	var err error

	defer func() {
		if err != nil {
			exitError(err)
		}
	}()

	cwd := ""
	if cwd, err = os.Getwd(); err != nil {
		err = ERR_GET_CWD_FAILED.New(errors.Params{"err": err})
		return
	}

	signfile := cwd + "/.redis_sync"

	syncConf := syncConfig{
		Redis: redisConfig{
			Address: "127.0.0.1:6379",
			Db:      0,
			Auth:    "",
		},
		ValueTypes: []valueType{
			valueType{
				Key:   "HKEY",
				Field: "Fild",
				Type:  "string",
			},
			valueType{
				Key:  "KEY",
				Type: "int32",
			},
		},
	}

	if e := os.Mkdir(signfile, 0644); e != nil {
		err = ERR_WRITE_INIT_CONF_ERROR.New(errors.Params{"err": e})
		return
	}

	strConf := ""

	if strConf, err = syncConf.Serialize(); err != nil {
		return
	}

	if e := ioutil.WriteFile("redis_sync.conf", []byte(strConf), 0644); e != nil {
		err = ERR_WRITE_INIT_CONF_ERROR.New(errors.Params{"err": e})
		return
	}

	repo := GitRepo{}

	if e := repo.Init(); e != nil {
		err = ERR_INIT_TO_GIT_REPO_FAILED.New(errors.Params{"err": e})
		return
	}

	if e := repo.AddUntracked(); e != nil {
		err = ERR_ADD_UNTRACKED_FILES_TO_GIT_FAILED.New(errors.Params{"err": e})
		return
	}

	if e := repo.Commit("data workdir initaled"); e != nil {
		err = ERR_COMMIT_GIT_REPO_FAILED.New(errors.Params{"err": e})
		return
	}
}

func checkIsSyncDir() bool {
	if _, e := os.Stat(".redis_sync"); e != nil {
		return false
	}
	return true
}

func initalConfig(configFile string) (err error) {
	if configFile == "" {
		configFile = "./redis_sync.conf"
	}

	if err = conf.Load(configFile); err != nil {
		return
	}

	return
}
