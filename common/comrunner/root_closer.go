package comrunner

import (
	"fmt"
	"log"
)

var vRootClosers []func()

func RegisterRootCloser(closer func()) {
	vRootClosers = append(vRootClosers, closer)
}

func SafeClose() {
	for _, closer := range vRootClosers {
		defer recoverRootCloser()
		closer()
	}
}

func recoverRootCloser() {
	errObj := recover()
	if errObj == nil {
		return
	}
	err, ok := errObj.(error)
	if !ok {
		err = fmt.Errorf("%v", errObj)
	}
	log.Printf("execute root closer failed | err=%s\n", err.Error())
}
