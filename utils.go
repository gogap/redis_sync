package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/gogap/errors"
)

func exitError(err error) {
	if errCode, ok := err.(errors.ErrCode); ok {
		if viewDetails {
			fmt.Printf("[ERR-%s-%d] %s \n%s\n", errCode.Namespace(), errCode.Code(), errCode.Error(), errCode.StackTrace())
		} else {
			fmt.Printf("[ERR-%s-%d] %s \n", errCode.Namespace(), errCode.Code(), errCode.Error())
		}
	} else {
		fmt.Printf("[ERR-%s] %s \n", REDIS_SYNC_ERR_NS, err.Error())
	}

	os.Exit(1)
}

func serializeObject(obj interface{}) (str string, err error) {
	switch reflect.TypeOf(obj).Kind() {
	case reflect.Map, reflect.Array, reflect.Slice:
		{
			if data, e := json.Marshal(&obj); e != nil {
				err = e
				return
			} else {
				str = string(data)
				return
			}
		}
	default:
		{
			str = fmt.Sprintf("%v", obj)
		}
	}

	return
}

func unmarshalJsonArray(data string) ([]interface{}, error) {
	var ret []interface{}
	err := json.Unmarshal([]byte(data), &ret)
	return ret, err
}
