package simple

import (
	"errors"
	"sort"
	"time"

	"git.fiblab.net/sim/backend/util"
	"git.fiblab.net/utils/lens"
	"git.fiblab.net/utils/pgxtool"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

type RoadStatus struct {
	Step  int `json:"step" db:"step"`
	Id    int `json:"id" db:"id"`       // 车道ID Road ID
	Level int `json:"level" db:"level"` // 路况状态（0-6） Road status (0-6) https://docs.fiblab.net/cityproto#city.map.v2.RoadLevel
}

func (t *RoadStatus) GetStep() int {
	return t.Step
}

func (t *RoadStatus) Copy(newStep int) lens.IHasStep {
	tt := *t
	tt.Step = newStep
	return &tt
}

var (
	roadStatusTool = pgxtool.New(&RoadStatus{})
	intervalCache  = cache.New(1*time.Minute, 2*time.Minute) // job -> road status interval
)

// @Summary Get Road Status
// @Produce application/json
// @Param tablename path string true "Simulation Name"
// @Param begin query number true "the start step of the data"
// @Param end query number true "Get the end step of the data (not included)"
// @Param interval query number false "Get the interval of the data (default is 1, return results step=begin,begin+1*interval,begin+2*interval...)"
// @Success 200 object util.Response{data=[]RoadStatus} "successful operation"
// @Router /simple/road-status/{tablename} [get]
func GetRoadStatusByName(c *gin.Context) {
	u := lens.ValidateUri(c)
	if u == nil {
		return
	}
	s := lens.ValidateParam[lens.Step](c)
	if s == nil {
		return
	}
	var interval int
	if i, ok := intervalCache.Get(u.Name); ok {
		interval = i.(int)
	} else {
		metas, err := QueryMetadata(&u.Name)
		if err != nil {
			c.JSON(500, util.NewErrorResponse(err))
			return
		} else if len(metas) == 0 {
			c.JSON(404, util.NewErrorResponse(errors.New("no found")))
			return
		}
		meta := metas[0]
		if meta.RoadStatusInterval == nil {
			c.JSON(400, util.NewErrorResponse(errors.New("no road status information")))
			return
		}
		interval = *meta.RoadStatusInterval
		intervalCache.Set(u.Name, interval, cache.DefaultExpiration)
	}
	all, err := lens.QueryPgTableWithStep[RoadStatus](
		roadStatusTool, u.Name+"_s_road",
		*s.Begin, *s.End, interval, 0, *s.Interval,
		"", nil,
	)
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}

	c.JSON(200, util.NewResponse(all))
}

type RoadStatusStat struct {
	Step                int     `json:"step"`
	MeanCongestionLevel float64 `json:"meanCongestionLevel"`   // 平均拥堵指数 Mean congestion level
	LevelCounts         []int   `json:"congestionLevelCounts"` // 拥堵比例（未归一化，按顺序从等级2->5 轻度拥堵/中度拥堵/重度拥堵/极端拥堵） Congestion level counts (not normalized, in order from level 2->5: mild/moderate/severe/extreme)
}

// @Summary Get Road Status Statistics
// @Produce application/json
// @Param tablename path string true "Simulation Name"
// @Param begin query number true "the start step of the data"
// @Param end query number true "Get the end step of the data (not included)"
// @Param interval query number false "Get the interval of the data (default is 1, return results step=begin,begin+1*interval,begin+2*interval...)"
// @Success 200 object util.Response{data=[]RoadStatusStat} "successful operation"
// @Router /simple/road-status-stat/{tablename} [get]
func GetRoadStatusStatByName(c *gin.Context) {
	// 查询与GetRoadStatusByName一致，在后端计算统计指标
	// MeanCongestionLevel = sum(level)/len(level)
	// LevelCounts = [count(level=2), count(level=3), count(level=4), count(level=5)]
	u := lens.ValidateUri(c)
	if u == nil {
		return
	}

	s := lens.ValidateParam[lens.Step](c)
	if s == nil {
		return
	}

	var interval int
	if i, ok := intervalCache.Get(u.Name); ok {
		interval = i.(int)
	} else {
		metas, err := QueryMetadata(&u.Name)
		if err != nil {
			c.JSON(500, util.NewErrorResponse(err))
			return
		} else if len(metas) == 0 {
			c.JSON(404, util.NewErrorResponse(errors.New("no found")))
			return
		}
		meta := metas[0]
		if meta.RoadStatusInterval == nil {
			c.JSON(400, util.NewErrorResponse(errors.New("no road status information")))
			return
		}
		interval = *meta.RoadStatusInterval
		intervalCache.Set(u.Name, interval, cache.DefaultExpiration)
	}

	all, err := lens.QueryPgTableWithStep[RoadStatus](
		roadStatusTool, u.Name+"_s_road",
		*s.Begin, *s.End, interval, 0, *s.Interval,
		"", nil,
	)
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}

	// 计算统计指标
	// 按照step分组
	step2status := make(map[int][]*RoadStatus)
	for _, v := range all {
		step2status[v.Step] = append(step2status[v.Step], v)
	}
	stat := make([]RoadStatusStat, 0, len(step2status))
	for step, statuses := range step2status {
		meanCongestionLevel := 0.0
		levelCounts := make([]int, 4)
		for _, status := range statuses {
			meanCongestionLevel += float64(status.Level)
			if status.Level >= 2 && status.Level <= 5 {
				levelCounts[status.Level-2]++
			}
		}
		if len(statuses) > 0 {
			meanCongestionLevel /= float64(len(statuses))
		}
		stat = append(stat, RoadStatusStat{
			Step:                step,
			MeanCongestionLevel: meanCongestionLevel,
			LevelCounts:         levelCounts,
		})
	}
	sort.Slice(stat, func(i, j int) bool {
		return stat[i].Step < stat[j].Step
	})
	c.JSON(200, util.NewResponse(stat))
}
