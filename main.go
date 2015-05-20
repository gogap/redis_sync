package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/gogap/errors"
	"github.com/hoisie/redis"
	"github.com/nu7hatch/gouuid"
)

type PushData struct {
	Key   string
	Field string
	Value string
}

const (
	_REDIS_SYNC_TOKEN_KEY = "__redis_sync_token"
)

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

	viewDetails = c.Bool("v")

	if !checkIsSyncDir() {
		err = ERR_THE_CWD_IS_NOT_SYNC_DIR.New()
		return
	}

	configFile := c.String("config")

	if err = initalConfig(configFile); err != nil {
		return
	}

	errorContinue := c.Bool("contine")
	overWrite := c.Bool("overwrite")

	redisToken := ""
	redisTokenExist := false

	token := c.String("token")
	if token == "" {
		token = getLocalSyncToken()
	}

	if redisToken, redisTokenExist, err = getRedisSyncToken(); err != nil {
		return
	}

	if redisTokenExist {
		if redisToken != token {
			err = ERR_SYNC_TOKEN_NOT_MATCH.New()
			return
		}
	} else if err = pushSyncToken(token); err != nil {
		return
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

				dockeyValType, keyValTypeExist := conf.KeyType(k)
				if keyValTypeExist {
					dataValType := getValType(v)
					if dataValType != dockeyValType {
						err = ERR_KEY_VAL_TYPE_NOT_MATCH_TO_CONF.New(
							errors.Params{
								"key":   k,
								"eType": dataValType,
								"type":  dockeyValType,
							},
						)
						return
					}
				}

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

				dockeyValType, keyValTypeExist := conf.HKeyType(datafileDir, k)
				if keyValTypeExist {
					dataValType := getValType(v)
					if dataValType != dockeyValType {
						err = ERR_HKEY_VAL_TYPE_NOT_MATCH_TO_CONF.New(
							errors.Params{
								"key":   datafileDir,
								"field": k,
								"eType": dataValType,
								"type":  dockeyValType,
							},
						)
						return
					}
				}

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

	client := redis.Client{
		Addr:        conf.Redis.Address,
		Db:          conf.Redis.Db,
		Password:    conf.Redis.Auth,
		MaxPoolSize: 3,
	}

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
				err = ERR_READ_USER_INPUT_ERROR.New()
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
						err = ERR_READ_USER_INPUT_ERROR.New()
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
							err = ERR_READ_USER_INPUT_ERROR.New()
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

	viewDetails = c.Bool("v")

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

	repo := GitRepo{}

	isStashed := false

	defer func() {
		if err != nil {
			if isStashed {
				repo.StashPop()
			}
			exitError(err)
		} else {
			if isStashed {
				repo.StashDrop()
			}
		}
	}()

	viewDetails = c.Bool("v")

	if !checkIsSyncDir() {
		err = ERR_THE_CWD_IS_NOT_SYNC_DIR.New()
		return
	}

	configFile := c.String("config")
	token := c.String("token")

	if token == "" {
		token = getLocalSyncToken()
	}

	if err = initalConfig(configFile); err != nil {
		return
	}

	redisToken := ""
	redisTokenExist := false

	if redisToken, redisTokenExist, err = getRedisSyncToken(); err != nil {
		return
	}

	if redisTokenExist {
		if redisToken != token {
			err = ERR_SYNC_TOKEN_NOT_MATCH.New()
			return
		}
	} else {
		if err = initSyncTokenOnNotExist(redisToken); err != nil {
			return
		}
	}

	var redisData, localData map[string][]PushData
	if redisData, err = getRedisData(); err != nil {
		return
	}

	if localData, err = getLocalData(); err != nil {
		return
	}

	needAddToLocal := []PushData{}

	for key, items := range redisData {
		if _, exist := localData[key]; !exist {
			needAddToLocal = append(needAddToLocal, items...)
		}
	}

	added := len(needAddToLocal)

	needDelToLocal := []PushData{}

	for key, items := range localData {
		if _, exist := redisData[key]; !exist {
			needDelToLocal = append(needDelToLocal, items...)
		}
	}

	deleted := len(needDelToLocal)

	valueChanged := []PushData{}

	for key, redisVals := range redisData {
		if vals, exist := localData[key]; exist {
			for _, redisItem := range redisVals {
				for _, localItem := range vals {
					if redisItem.Key == localItem.Key &&
						redisItem.Field == localItem.Field &&
						redisItem.Value != localItem.Value {
						valueChanged = append(valueChanged, redisItem)
					}
				}
			}
		}
	}

	updated := len(valueChanged)

	if !repo.IsClean() {
		if e := repo.StashSaveAll(); e != nil {
			err = ERR_STASH_CURRENT_DIR_FAILED.New(errors.Params{"err": e})
			return
		}
		isStashed = true

		repo.StashApply()
	}

	if err = addDataToLocal(needAddToLocal); err != nil {
		return
	}

	if err = removeLocalData(needDelToLocal); err != nil {
		return
	}

	if err = updateLocalData(valueChanged); err != nil {
		return
	}

	fmt.Printf("update: %d, delete: %d, add: %d\n", updated, deleted, added)
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

	signDir := cwd + "/.redis_sync"

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
				Type: "int",
			},
		},
	}

	if e := os.Mkdir(signDir, 0766); e != nil {
		err = ERR_WRITE_INIT_CONF_ERROR.New(errors.Params{"err": e})
		return
	}

	token := c.String("token")

	if token == "" {
		tokenUUID, _ := uuid.NewV4()
		token = strings.Replace(tokenUUID.String(), "-", "", -1)
	}

	if e := ioutil.WriteFile(signDir+"/token", []byte(token), 0644); e != nil {
		err = ERR_WRITE_SYNC_TOKEN_FAILED.New(errors.Params{"err": e})
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

func initSyncTokenOnNotExist(token string) (err error) {
	if _, e := os.Stat(".redis_sync/token"); e != nil {
		if os.IsNotExist(e) {
			if e := ioutil.WriteFile(".redis_sync/token", []byte(token), 0644); e != nil {
				err = ERR_WRITE_SYNC_TOKEN_FAILED.New(errors.Params{"err": e})
				return
			}
		}
	}
	return
}

func getLocalSyncToken() string {
	tk, _ := ioutil.ReadFile(".redis_sync/token")
	return string(tk)
}

func getRedisSyncToken() (token string, exist bool, err error) {
	client := redis.Client{
		Addr:        conf.Redis.Address,
		Db:          conf.Redis.Db,
		Password:    conf.Redis.Auth,
		MaxPoolSize: 3,
	}

	if exist, err = client.Exists(_REDIS_SYNC_TOKEN_KEY); err != nil {
		err = ERR_GET_REDIS_SYNC_TOKEN_FAILED.New(errors.Params{"err": err})
		return
	} else if !exist {
		return "", false, nil
	} else {
		if bToken, e := client.Get(_REDIS_SYNC_TOKEN_KEY); e != nil {
			err = ERR_GET_REDIS_SYNC_TOKEN_FAILED.New(errors.Params{"err": e})
			return
		} else {
			return string(bToken), true, nil
		}
	}
}

func pushSyncToken(token string) (err error) {
	client := redis.Client{
		Addr:        conf.Redis.Address,
		Db:          conf.Redis.Db,
		Password:    conf.Redis.Auth,
		MaxPoolSize: 3,
	}

	if exist, e := client.Exists(_REDIS_SYNC_TOKEN_KEY); e != nil {
		err = ERR_GET_REDIS_SYNC_TOKEN_FAILED.New(errors.Params{"err": e})
		return
	} else if !exist {
		if e := client.Set(_REDIS_SYNC_TOKEN_KEY, []byte(token)); e != nil {
			err = ERR_SYNC_TOKEN_TO_REDIS_FAILED.New(errors.Params{"err": e})
			return
		}
	} else {
		err = ERR_REDIS_ALREADY_HAVE_TOKEN.New()
		return
	}
	return
}

func checkIsSyncDir() bool {
	if _, e := os.Stat(".redis_sync"); e != nil {
		return false
	}
	return true
}

func addDataToLocal(data []PushData) (err error) {
	for _, pushData := range data {
		if err = setLocalDataValue(pushData); err != nil {
			return
		}
	}
	return
}

func removeLocalData(data []PushData) (err error) {
	for _, d := range data {
		if d.Field == "" {
			var vals map[string]interface{}
			if vals, err = readDataFile("."); err != nil {
				return
			}

			delete(vals, d.Key)

			if err = writeDataFile(".", vals); err != nil {
				return
			}
		} else {
			if _, e := os.Stat(d.Key); e != nil {
				if !os.IsNotExist(e) {
					err = ERR_GET_KEY_DIR_FAILED.New(errors.Params{"err": e})
					return
				}
			} else {
				if e := os.RemoveAll(d.Key); e != nil {
					err = ERR_REMOVE_LOCAL_HKEY_FAILED.New(errors.Params{"err": e})
					return
				}
			}
		}
	}

	return
}

func updateLocalData(data []PushData) (err error) {
	for _, d := range data {
		if err = setLocalDataValue(d); err != nil {
			return
		}
	}
	return
}

func getTypedVal(keyType string, strVal string) (val interface{}, err error) {
	switch keyType {
	case "number":
		{
			if v, e := strconv.ParseFloat(strVal, 64); e != nil {
				err = ERR_COULD_NOT_CONV_VAL_TO_NUMBER.New(errors.Params{"val": strVal, "err": e})
				return
			} else {
				val = v
			}
			return
		}
	case "object":
		{
			if strVal == "" {
				return
			}

			mapVal := map[string]interface{}{}
			if e := json.Unmarshal([]byte(strVal), &mapVal); e != nil {
				err = ERR_COULD_NOT_CONV_VAL_TO_MAP.New(errors.Params{"val": strVal, "err": e})
				return
			} else {
				val = mapVal
			}

			return
		}
	case "array":
		{
			if v, e := unmarshalJsonArray(strVal); e != nil {
				err = ERR_COULD_NOT_CONV_VAL_TO_ARRAY.New(errors.Params{"val": strVal, "err": e})
				return
			} else {
				val = v
			}

			return
		}
	}
	return strVal, nil
}

func getValType(v interface{}) string {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Int,
		reflect.Int8,
		reflect.Int32,
		reflect.Int64,
		reflect.Float32,
		reflect.Float64:
		return "number"
	case reflect.Map:
		return "object"
	case reflect.Slice, reflect.Array:
		return "array"
	default:
		return "string"
	}
}

func setLocalDataValue(data PushData) (err error) {
	if data.Key == "" {
		err = ERR_THE_DATA_KEY_IS_EMPTY.New()
		return
	}

	keyType := ""

	if data.Field == "" {
		keyType, _ = conf.KeyType(data.Key)
	} else {
		keyType, _ = conf.HKeyType(data.Key, data.Field)
	}

	var val interface{}

	if val, err = getTypedVal(keyType, data.Value); err != nil {
		return
	}

	if data.Field == "" {
		if err = initDataFileOnNotExist("."); err != nil {
			return
		}

		vals := map[string]interface{}{}
		if vals, err = readDataFile("."); err != nil {
			return
		}

		originValType := ""
		if originV, exist := vals[data.Key]; exist {
			originValType = getValType(originV)
		} else {
			originValType = keyType
		}

		if originValType == "" {
			vals[data.Key] = data.Value
		} else if originValType != keyType {
			err = ERR_REDIS_KEY_TYPE_NOT_MATCH.New(errors.Params{"originType": originValType, "exceptType": keyType, "key": data.Key})
			return
		} else {
			vals[data.Key] = val
		}

		if err = writeDataFile(".", vals); err != nil {
			return
		}
	} else {
		if err = initDataFileOnNotExist(data.Key); err != nil {
			return
		}

		vals := map[string]interface{}{}
		if vals, err = readDataFile(data.Key); err != nil {
			return
		}

		originValType := ""
		if originV, exist := vals[data.Field]; exist {
			originValType = getValType(originV)
		} else {
			originValType = keyType
		}

		if originValType == "" {
			vals[data.Field] = data.Value
		} else if originValType != keyType {
			err = ERR_REDIS_HKEY_TYPE_NOT_MATCH.New(errors.Params{"originType": originValType, "exceptType": keyType, "key": data.Key, "field": data.Field})
			return
		} else {
			vals[data.Field] = val
		}

		if err = writeDataFile(data.Key, vals); err != nil {
			return
		}
	}

	return
}

func initDataFileOnNotExist(dir string) (err error) {
	if dir != "." {
		os.MkdirAll(dir, 0766)
	}

	datafile := dir + "/data"
	if fi, e := os.Stat(datafile); e != nil {
		if !os.IsNotExist(e) {
			err = ERR_READ_DATAFILE_ERROR.New(errors.Params{"fileName": datafile, "err": e})
			return
		} else {
			if e := ioutil.WriteFile(datafile, []byte("{}"), 0644); e != nil {
				err = ERR_INITAL_DATAFILE_FAILED.New(errors.Params{"fileName": datafile, "err": e})
				return
			}
		}
	} else if fi.IsDir() {
		err = ERR_DATAFILE_COULD_NOT_BE_A_DIR.New(errors.Params{"fileName": datafile})
		return
	}
	return
}

func readDataFile(dir string) (vals map[string]interface{}, err error) {
	datafile := dir + "/data"

	if data, e := ioutil.ReadFile(datafile); e != nil {
		err = ERR_READ_DATAFILE_ERROR.New(errors.Params{"fileName": datafile, "err": e})
		return
	} else if e := json.Unmarshal(data, &vals); e != nil {
		err = ERR_PARSE_DATAFILE_ERROR.New(errors.Params{"fileName": datafile, "err": e})
		return
	}

	return
}

func writeDataFile(dir string, vals map[string]interface{}) (err error) {
	datafile := dir + "/data"

	if data, e := json.MarshalIndent(vals, "", "    "); e != nil {
		err = ERR_SERIALIZE_DATAFILE_FAILED.New(errors.Params{"fileName": datafile, "err": e})
		return
	} else if e := ioutil.WriteFile(datafile, data, 0644); e != nil {
		err = ERR_SAVE_DATAFILE_FAILED.New(errors.Params{"fileName": datafile, "err": e})
		return
	}

	return
}

func getRedisData() (ret map[string][]PushData, err error) {

	client := redis.Client{
		Addr:        conf.Redis.Address,
		Db:          conf.Redis.Db,
		Password:    conf.Redis.Auth,
		MaxPoolSize: 3,
	}

	redisData := make(map[string][]PushData)
	if keys, e := client.Keys("*"); e != nil {
		err = ERR_GET_REDIS_KEYS_FAILED.New(errors.Params{"err": err})
		return
	} else {
		for _, key := range keys {
			if keyType, e := client.Type(key); e != nil {
				return
			} else {
				if keyType == "string" {
					val := ""
					if bVal, e := client.Get(key); e != nil {
						err = ERR_GET_REDIS_VALUE_ERROR.New(errors.Params{"key": key, "err": e})
						return
					} else {
						val = string(bVal)
					}

					redisData[key] = []PushData{PushData{
						Key:   key,
						Value: val,
					}}
				} else if keyType == "hash" {
					fieldValues := map[string]string{}

					if e := client.Hgetall(key, &fieldValues); e != nil {
						err = ERR_GET_REDIS_VALUE_ERROR.New(errors.Params{"key": key, "err": e})
						return
					}

					for field, value := range fieldValues {
						if vals, exist := redisData[key]; exist {
							vals = append(vals, PushData{
								Key:   key,
								Field: field,
								Value: value,
							})
							redisData[key] = vals
						} else {
							vals := append([]PushData{PushData{
								Key:   key,
								Field: field,
								Value: value,
							}})
							redisData[key] = vals
						}
					}

				}
			}
		}
	}

	if _, exist := redisData[_REDIS_SYNC_TOKEN_KEY]; exist {
		delete(redisData, _REDIS_SYNC_TOKEN_KEY)
	}

	ret = redisData
	return
}

func getLocalData() (ret map[string][]PushData, err error) {
	localData := make(map[string][]PushData)

	workDir := ""

	if workDir, err = os.Getwd(); err != nil {
		err = ERR_GET_CWD_FAILED.New(errors.Params{"err": err})
		return
	}

	fnWalk := func(path string, info os.FileInfo, e error) (err error) {

		if !info.IsDir() {
			return
		} else if strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		key, _ := filepath.Rel(workDir, path)

		var data map[string][]PushData
		if data, err = readLocalData(key); err != nil {
			return
		}

		for k, v := range data {
			if originV, exist := localData[k]; exist {
				localData[k] = append(originV, v...)
			} else {
				localData[k] = v
			}
		}

		return
	}

	if err = filepath.Walk(workDir, fnWalk); err != nil {
		return
	}

	ret = localData

	return
}

func readLocalData(key string) (ret map[string][]PushData, err error) {
	localData := make(map[string][]PushData)

	if data, e := ioutil.ReadFile(key + "/data"); e != nil {
		if !os.IsNotExist(e) {
			err = ERR_READ_DATAFILE_ERROR.New(errors.Params{"fileName": "data", "err": e})
			return
		}
	} else {
		keyValues := map[string]interface{}{}
		if e := json.Unmarshal(data, &keyValues); e != nil {
			err = ERR_PARSE_DATAFILE_ERROR.New(errors.Params{"fileName": "data", "err": e})
			return
		}

		if key == "." {
			for k, v := range keyValues {
				if strV, e := serializeObject(v); e != nil {
					err = ERR_COULD_NOT_CONV_VAL_TO_STRING.New(errors.Params{"key": k, "err": e})
					return
				} else {
					localData[k] = []PushData{PushData{
						Key:   k,
						Value: strV,
					}}
				}
			}
		} else {
			pData := []PushData{}
			for k, v := range keyValues {
				if strV, e := serializeObject(v); e != nil {
					err = ERR_COULD_NOT_CONV_VAL_TO_STRING.New(errors.Params{"key": k, "err": e})
					return
				} else {
					pData = append(pData, PushData{
						Key:   key,
						Field: k,
						Value: strV,
					})
					localData[key] = pData
				}
			}
		}

	}

	ret = localData
	return
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
