package api

import (
	"fmt"
	"net/http"

	"shopble/api"
	"shopble/common/comrunner"
)

func StartHttpSvc(host string, port int) {
	router := api.New()
	InitRouter(router)
	session := comrunner.NewSession(comrunner.NewHttpService(&http.Server{
		Handler: router,
		Addr:    fmt.Sprintf("%s:%d", host, port),
	}))
	comrunner.RunSession(session)
}
