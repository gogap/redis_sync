package main

import (
	"fmt"
	"os"

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
