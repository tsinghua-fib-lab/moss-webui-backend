package simple

import (
	"context"
	"errors"

	"git.fiblab.net/sim/backend/util"
	"git.fiblab.net/utils/lens"
	"git.fiblab.net/utils/pgxtool"
	"github.com/gin-gonic/gin"
)

type Metadata struct {
	Name        string  `json:"name" db:"name"`                 // 模拟名 Simulation Name
	Start       int     `json:"start" db:"start"`               // 起始模拟步数 Start Step
	Steps       int     `json:"steps" db:"steps"`               // 模拟总步数 Total Steps
	Time        float64 `json:"time" db:"time"`                 // 单步模拟对应的时间长度（秒） Time length of each step (second)
	TotalAgents int     `json:"total_agents" db:"total_agents"` // 模拟的总人口 Total population of the simulation
	Map         string  `json:"-" db:"map"`                     // 地图路径 Map Path in MongoDB (format: "db.collection")

	// 微观区域范围 Microscopic Area Range

	MinLng float64 `json:"min_lng" db:"min_lng"`
	MinLat float64 `json:"min_lat" db:"min_lat"`
	MaxLng float64 `json:"max_lng" db:"max_lng"`
	MaxLat float64 `json:"max_lat" db:"max_lat"`

	// 道路路况 Road Status

	RoadStatusVMin     *float64 `json:"-" db:"road_status_v_min"`
	RoadStatusInterval *int     `json:"-" db:"road_status_interval"`

	// 版本 Version

	Version int `json:"-" db:"version"`
}

var metaTool = pgxtool.New(&Metadata{})

func QueryMetadata(name *string) ([]*Metadata, error) {
	var where string
	var args []any
	if name == nil {
		where = ""
	} else {
		where = "NAME=$1"
		args = []any{*name}
	}
	rows, err := lens.DefaultPg().Query(
		context.Background(),
		metaTool.BuildSelectSQL("meta_simple", where, nil),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	all := make([]*Metadata, 0)
	for rows.Next() {
		one := &Metadata{}
		if err := metaTool.Scan(rows, one); err != nil {
			return nil, err
		}
		all = append(all, one)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if name != nil && len(all) > 1 {
		return nil, errors.New("duplicate records")
	}
	return all, nil
}

// @Summary Get All Simulation Metadata
// @Produce application/json
// @Success 200 object util.Response{data=[]Metadata} "successful operation"
// @Router /simple/sims/ [get]
func GetAllSim(c *gin.Context) {
	if res, err := QueryMetadata(nil); err != nil {
		c.JSON(500, util.NewErrorResponse(err))
	} else {
		c.JSON(200, util.NewResponse(res))
	}
}

// @Summary Get Simulation Metadata
// @Produce application/json
// @Param simname path string true "Simulation Name"
// @Success 200 object util.Response{data=[]Metadata} "successful operation"
// @Router /simple/sims/{simname} [get]
func GetSimByName(c *gin.Context) {
	u := lens.ValidateUri(c)
	if u == nil {
		return
	}
	if res, err := QueryMetadata(&u.Name); err != nil {
		c.JSON(500, util.NewErrorResponse(err))
	} else if len(res) == 0 {
		c.JSON(404, util.NewErrorResponse(errors.New("not found")))
	} else {
		c.JSON(200, util.NewResponse(res))
	}
}
