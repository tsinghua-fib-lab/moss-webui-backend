package simple

import (
	"git.fiblab.net/sim/backend/util"
	"git.fiblab.net/utils/lens"
	"git.fiblab.net/utils/pgxtool"
	"github.com/gin-gonic/gin"
)

type TrafficLight struct {
	Step  int `json:"step" db:"step"`
	Id    int `json:"id" db:"id"`       // 车道ID
	State int `json:"state" db:"state"` // 信控状态（0无/1红/2绿/3黄）
}

func (t *TrafficLight) GetStep() int {
	return t.Step
}

func (t *TrafficLight) Copy(newStep int) lens.IHasStep {
	tt := *t
	tt.Step = newStep
	return &tt
}

var (
	tlTool = pgxtool.New(&TrafficLight{})
)

// @Summary Get Traffic Lights
// @Produce application/json
// @Param tablename path string true "Simulation Name"
// @Param begin query number true "the start step of the data"
// @Param end query number true "Get the end step of the data (not included)"
// @Param lat1 query number true "min latitude for filtering"
// @Param lat2 query number true "max latitude for filtering"
// @Param lng1 query number true "min longitude for filtering"
// @Param lng2 query number true "max longitude for filtering"
// @Param interval query number false "Get the interval of the data (default is 1, return results step=begin,begin+1*interval,begin+2*interval...)"
// @Success 200 object util.Response{data=[]TrafficLight} "successful operation"
// @Router /simple/traffic-lights/{tablename} [get]
func GetTrafficLightByName(c *gin.Context) {
	u := lens.ValidateUri(c)
	if u == nil {
		return
	}
	s := lens.ValidateParam[lens.StepCoordinate](c)
	if s == nil {
		return
	}

	all, err := lens.QueryPgTableWithStep[TrafficLight](
		tlTool, u.Name+"_s_traffic_light",
		*s.Begin, *s.End, 1, 0, *s.Interval,
		"LAT>=$1 AND LAT<$2 AND LNG>=$3 AND LNG<$4",
		[]any{*s.Lat1, *s.Lat2, *s.Lng1, *s.Lng2},
	)
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}

	c.JSON(200, util.NewResponse(all))
}
