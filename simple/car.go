package simple

import (
	"errors"

	"git.fiblab.net/sim/backend/util"
	"git.fiblab.net/utils/lens"
	"git.fiblab.net/utils/pgxtool"

	"github.com/gin-gonic/gin"
)

type CarV2 struct {
	Step          int     `json:"step" db:"step"`
	Id            int     `json:"id" db:"id"`                        // 车辆ID Vehicle ID
	LaneId        int     `json:"laneId" db:"parent_id"`             // 车辆所在车道ID Vehicle Lane ID
	Direction     float64 `json:"direction" db:"direction"`          // 车辆方向角（rad，正北为0） Vehicle Direction Angle(rad, 0 is north)
	Lng           float64 `json:"lng" db:"lng"`                      // 经度 Longitude
	Lat           float64 `json:"lat" db:"lat"`                      // 纬度 Latitude
	Model         string  `json:"model" db:"model"`                  // 可视化时的车辆模型 Vehicle Model for Visualization
	Z             float64 `json:"z" db:"z"`                          // 高程（单位：米） Elevation (unit: meter)
	Pitch         float64 `json:"pitch" db:"pitch"`                  // 俯仰角（rad，0为水平） Pitch Angle (rad, 0 is horizontal)
	V             float64 `json:"v" db:"v"`                          // 速度（单位：米/秒） Speed (unit: meter/second)
	NumPassengers int32   `json:"numPassengers" db:"num_passengers"` // 乘客数 Number of Passengers
}

func (c *CarV2) GetStep() int {
	return c.Step
}

func (c *CarV2) Copy(newStep int) lens.IHasStep {
	cc := *c
	cc.Step = newStep
	return &cc
}

var (
	carV2Tool = pgxtool.New(&CarV2{})
)

// @Summary Get Vehicles
// @Produce application/json
// @Param tablename path string true "Simulation Name"
// @Param begin query number true "the start step of the data"
// @Param end query number true "Get the end step of the data (not included)"
// @Param lat1 query number true "min latitude for filtering"
// @Param lat2 query number true "max latitude for filtering"
// @Param lng1 query number true "min longitude for filtering"
// @Param lng2 query number true "max longitude for filtering"
// @Param interval query number false "Get the interval of the data (default is 1, return results step=begin,begin+1*interval,begin+2*interval...)"
// @Success 200 object util.Response{data=[]CarV2} ""
// @Router /simple/cars/{tablename} [get]
func GetCarsByName(c *gin.Context) {
	u := lens.ValidateUri(c)
	if u == nil {
		return
	}
	s := lens.ValidateParam[lens.StepCoordinate](c)
	if s == nil {
		return
	}

	// get meta
	metas, err := QueryMetadata(&u.Name)
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
	} else if len(metas) == 0 {
		c.JSON(404, util.NewErrorResponse(errors.New("not found")))
	}
	meta := metas[0]
	// download data
	switch meta.Version {
	case 2:
		all, err := lens.QueryPgTableWithStep[CarV2](
			carV2Tool, u.Name+"_s_cars",
			*s.Begin, *s.End, 1, 0, *s.Interval,
			"LAT>=$1 AND LAT<$2 AND LNG>=$3 AND LNG<$4",
			[]any{*s.Lat1, *s.Lat2, *s.Lng1, *s.Lng2},
		)
		if err != nil {
			c.JSON(500, util.NewErrorResponse(err))
			return
		}
		for _, one := range all {
			one.Direction = util.ToFixed(one.Direction, 2)
			one.Lng = util.ToFixed(one.Lng, 8)
			one.Lat = util.ToFixed(one.Lat, 8)
		}
		c.JSON(200, util.NewResponse(all))
	default:
		c.JSON(500, util.NewErrorResponse(errors.New("unsupported version")))
	}
}
