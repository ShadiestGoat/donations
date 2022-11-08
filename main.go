package main

import (
	"net/http"
	"os"
)

func main() {
	InitLogger()
	logger.Logf(LL_DEBUG, "Logger started!")
	InitConfig()
	os.Setenv("TZ", "UTC")
	logger.Logf(LL_DEBUG, "Config loaded!")
	InitAuths()
	InitSnowflake()
	logger.Logf(LL_DEBUG, "SnowNode loaded!")
	InitFrontend()
	logger.Logf(LL_DEBUG, "Frontend loaded!")

	InitDB()
	logger.Logf(LL_DEBUG, "Database connected!")

	r := RouterBase()
	r.Mount(`/api`, RouterAPI())

	logger.Logf(LL_DEBUG, "Server started!")
	PanicIfErr(http.ListenAndServe(":"+PORT, r))
}
