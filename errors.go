package main

import (
	"github.com/gogap/errors"
)

const (
	REDIS_SYNC_ERR_NS = "REDIS_SYNC"
)

var (
	ERR_LOAD_CONFIG_FAILED                = errors.TN(REDIS_SYNC_ERR_NS, 1, "load config file of {{.fileName}} failed: err: {{.err}}")
	ERR_PARSE_CONFIG_FAILED               = errors.TN(REDIS_SYNC_ERR_NS, 2, "parse config file of {{.fileName}} failed: err: {{.err}}")
	ERR_CONFIG_VALUE_MUST_INPUT           = errors.TN(REDIS_SYNC_ERR_NS, 3, "config of {{.configName}} must input")
	ERR_REDIS_KEY_IS_EMPTY                = errors.TN(REDIS_SYNC_ERR_NS, 4, "redis key is empty")
	ERR_UNSUPPORT_TYPE_MAPPING            = errors.TN(REDIS_SYNC_ERR_NS, 5, "unsupport type mapping, key: {{.key}}, field: {{.field}}, type: {{.type}}")
	ERR_KEY_TYPES_MAP_ALREADY_EXIST       = errors.TN(REDIS_SYNC_ERR_NS, 6, "key types map already exist, key: {{.key}}, field: {{.field}}, type: {{.type}}")
	ERR_DAIL_REDIS_FAILED                 = errors.TN(REDIS_SYNC_ERR_NS, 7, "dail redis error: {{.err}}")
	ERR_SERIALIZE_CONFIG_FAILED           = errors.TN(REDIS_SYNC_ERR_NS, 8, "serialize config failed: {{.err}}")
	ERR_GET_CWD_FAILED                    = errors.TN(REDIS_SYNC_ERR_NS, 9, "get current dir faild: {{.err}}")
	ERR_WRITE_INIT_CONF_ERROR             = errors.TN(REDIS_SYNC_ERR_NS, 10, "write init config error: {{.err}}")
	ERR_THE_CWD_IS_NOT_SYNC_DIR           = errors.TN(REDIS_SYNC_ERR_NS, 11, "the current dir is not redis sync dir, if you need to sync this dir, please use init command")
	ERR_READ_DATAFILE_ERROR               = errors.TN(REDIS_SYNC_ERR_NS, 12, "read data file of {{.fileName}} err: {{.err}}")
	ERR_PARSE_DATAFILE_ERROR              = errors.TN(REDIS_SYNC_ERR_NS, 13, "parse data file of {{.fileName}} error: {{.err}}")
	ERR_SET_REDIS_DATA_ERROR              = errors.TN(REDIS_SYNC_ERR_NS, 14, "set redis data error, key: {{.key}}, value: {{.value}}, err: {{.err}} ")
	ERR_HSET_REDIS_DATA_ERROR             = errors.TN(REDIS_SYNC_ERR_NS, 15, "hset redis data error, key: {{.key}}, filed: {{.field}}, value: {{.value}}, err: {{.err}} ")
	ERR_GET_REDIS_VALUE_ERROR             = errors.TN(REDIS_SYNC_ERR_NS, 16, "get redis value failed, key: {{.key}}, err: {{.err}}")
	ERR_READE_USER_INPUT_ERROR            = errors.TN(REDIS_SYNC_ERR_NS, 17, "could not get user input info")
	ERR_GET_KEY_STATUS_ERROR              = errors.TN(REDIS_SYNC_ERR_NS, 18, "get key of {{.key}} status error: {{.err}}")
	ERR_INIT_TO_GIT_REPO_FAILED           = errors.TN(REDIS_SYNC_ERR_NS, 19, "could not init the data dir as git repo, error: {{.err}}")
	ERR_ADD_UNTRACKED_FILES_TO_GIT_FAILED = errors.TN(REDIS_SYNC_ERR_NS, 20, "could not add untracked file to git repo, error: {{.err}}")
	ERR_COMMIT_GIT_REPO_FAILED            = errors.TN(REDIS_SYNC_ERR_NS, 21, "commit git repo failed, error: {{.err}}")
	ERR_COMMIT_CURRENT_WORKDIR_NOT_CLEAN  = errors.TN(REDIS_SYNC_ERR_NS, 22, "something changes in data dir, please commit changes before push")
	ERR_COMMIT_MSG_NOT_INPUT              = errors.TN(REDIS_SYNC_ERR_NS, 23, "commit message not input")
	ERR_ADD_MODIFIED_FILES_TO_GIT_FAILED  = errors.TN(REDIS_SYNC_ERR_NS, 24, "could not add modified file to git repo, error: {{.err}}")

	ERR_GET_REPO_STATUS_FAILED       = errors.TN(REDIS_SYNC_ERR_NS, 25, "get repo status faild, err: {{.err}}")
	ERR_GET_REPO_DIFF_FAILED         = errors.TN(REDIS_SYNC_ERR_NS, 26, "get repo diff failed, err: {{.err}}")
	ERR_GET_REDIS_KEY_TYPE_FAILED    = errors.TN(REDIS_SYNC_ERR_NS, 27, "get redis key type failed, key: {{.key}}, err: {{.err}}")
	ERR_DELETE_REDIS_KEY_FAILED      = errors.TN(REDIS_SYNC_ERR_NS, 28, "delete redis key failed, key: {{.key}}, err: {{.err}}")
	ERR_HGET_REDIS_VALUE_ERROR       = errors.TN(REDIS_SYNC_ERR_NS, 29, "hget redis value failed, key: {{.key}}, field: {{.field}}, err: {{.err}}")
	ERR_HGET_KEY_STATUS_ERROR        = errors.TN(REDIS_SYNC_ERR_NS, 30, "hget key of {{.key}}, field: {{.field}}, status error: {{.err}}")
	ERR_COULD_NOT_CONV_VAL_TO_STRING = errors.TN(REDIS_SYNC_ERR_NS, 31, "could not convert value to string, key: {{.key}}, error: {{.err}}")
)
