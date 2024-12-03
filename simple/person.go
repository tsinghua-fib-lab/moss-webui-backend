package simple

import (
	"git.fiblab.net/sim/backend/util"
	"git.fiblab.net/utils/lens"
	"git.fiblab.net/utils/pgxtool"
	"github.com/gin-gonic/gin"
)

type Person struct {
	Step      int     `json:"step" db:"step"`
	Id        int     `json:"id" db:"id"`               // 人的ID Person ID
	ParentId  int     `json:"parentId" db:"parent_id"`  // 人所在Aoi或车道ID Aoi or Lane ID where the person is located
	Direction float64 `json:"direction" db:"direction"` // 人的方向角（rad，正北为0） Person's direction angle (rad, 0 is north)
	Lng       float64 `json:"lng" db:"lng"`             // 经度 Longitude
	Lat       float64 `json:"lat" db:"lat"`             // 纬度 Latitude
	Z         float64 `json:"z" db:"z"`                 // 高程（单位：米） Elevation (unit: meter)
	V         float64 `json:"v" db:"v"`                 // 速度（单位：米/秒） Speed (unit: meter/second)
	Model     string  `json:"model" db:"model"`         // 可视化时的人的模型 Person's model for visualization
}

func (p *Person) GetStep() int {
	return p.Step
}

func (p *Person) Copy(newStep int) lens.IHasStep {
	pp := *p
	pp.Step = newStep
	return &pp
}

var (
	personTool = pgxtool.New(&Person{})
)

// @Summary Get Pedestrians
// @Produce application/json
// @Param tablename path string true "Simulation Name"
// @Param begin query number true "the start step of the data"
// @Param end query number true "Get the end step of the data (not included)"
// @Param lat1 query number true "min latitude for filtering"
// @Param lat2 query number true "max latitude for filtering"
// @Param lng1 query number true "min longitude for filtering"
// @Param lng2 query number true "max longitude for filtering"
// @Param interval query number false "Get the interval of the data (default is 1, return results step=begin,begin+1*interval,begin+2*interval...)"
// @Success 200 object util.Response{data=[]Person} "北京返回值"
// @Router /simple/people/{tablename} [get]
func GetPeopleByName(c *gin.Context) {
	u := lens.ValidateUri(c)
	if u == nil {
		return
	}
	s := lens.ValidateParam[lens.StepCoordinate](c)
	if s == nil {
		return
	}

	all, err := lens.QueryPgTableWithStep[Person](
		personTool, u.Name+"_s_people",
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
}
