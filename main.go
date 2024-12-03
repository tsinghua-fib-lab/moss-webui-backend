package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	_ "git.fiblab.net/sim/backend/docs"
	"git.fiblab.net/sim/backend/simple"
	"git.fiblab.net/utils/lens"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	timeout "github.com/vearne/gin-timeout"
)

func main() {
	godotenv.Load()

	lens.InitMongo("mongodb://user:password@url:port/", "db")
	lens.InitPg("postgres://user:password@url:port/db")
	lens.InitEngine("8080")

	r := lens.DefaultEngine()
	r.Use(BlackList(strings.Split(os.Getenv("BLACKLIST"), ",")))
	r.Use(timeout.Timeout(
		timeout.WithTimeout(20*time.Second),
		timeout.WithErrorHttpCode(http.StatusRequestTimeout), // optional
		timeout.WithDefaultMsg("timeout"),                    // optional
	))
	// gin-swagger重定向方式
	// use `swag init` to generate docs
	// don't forget to `import _ "git.fiblab.net/sim/backend/docs"`
	r.GET("/", func(c *gin.Context) {
		c.Redirect(301, "/swagger/index.html")
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// simple API
	simpleGroup := r.Group("/simple")
	{
		simpleGroup.GET("/junclane/:name", simple.GetJunclaneByName)
		simpleGroup.GET("/all-roadlane/:name", simple.GetAllRoadlaneByName)
		simpleGroup.GET("/all-lane/:name", simple.GetAllLaneByName)
		simpleGroup.GET("/roadlane/:name", simple.GetRoadlaneByName)
		simpleGroup.GET("/aoi/:name", simple.GetAoiByName)
		simpleGroup.GET("/sims", simple.GetAllSim)
		simpleGroup.GET("/sims/:name", simple.GetSimByName)
		simpleGroup.GET("/cars/:name", simple.GetCarsByName)
		simpleGroup.GET("/people/:name", simple.GetPeopleByName)
		simpleGroup.GET("/traffic-lights/:name", simple.GetTrafficLightByName)
		simpleGroup.GET("/road-status/:name", simple.GetRoadStatusByName)
		simpleGroup.GET("/road-status-stat/:name", simple.GetRoadStatusStatByName)
	}

	lens.Run()
}
