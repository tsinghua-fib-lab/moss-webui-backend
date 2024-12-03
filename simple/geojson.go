package simple

import (
	"context"
	"errors"
	"strings"

	"git.fiblab.net/sim/backend/util"
	"git.fiblab.net/utils/lens"
	"git.fiblab.net/utils/proj"
	"github.com/gin-gonic/gin"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	WGS84CRS = "EPSG:4326"
)

// 枚举: All, Junction, Road
type LaneType int

const (
	AllLane LaneType = iota
	JunctionLane
	RoadLane
)

type mapHeader struct {
	Data struct {
		Projection string `bson:"projection"`
	} `bson:"data"`
}

type mapNode struct {
	X float64 `bson:"x"`
	Y float64 `bson:"y"`
}

type mapLane struct {
	ID   int32     `bson:"id"`
	Line []mapNode `bson:"line"`
	Type int32     `bson:"type"`
}

type mapAoi struct {
	ID        int32     `bson:"id"`
	Positions []mapNode `bson:"positions"`
}

type mapRoad struct {
	ID      int32   `bson:"id"`
	LaneIDs []int32 `bson:"lane_ids"`
}

func newGeoJsonLane(id int32, typ int32, coordinates [][]float64) *geojson.Feature {
	lineString := orb.LineString(lo.Map(coordinates, func(c []float64, _ int) orb.Point {
		return orb.Point{c[0], c[1]}
	}))
	feature := geojson.NewFeature(lineString)
	feature.ID = id
	feature.Properties = map[string]any{
		"id":   id,
		"type": typ,
	}
	return feature
}

func downloadLanes(c *gin.Context, name string, typ LaneType) (geojsons []*geojson.Feature, finished bool) {
	finished = true

	metas, err := QueryMetadata(&name)
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	} else if len(metas) == 0 {
		c.JSON(404, util.NewErrorResponse(errors.New("not found")))
		return
	}
	meta := metas[0]
	parts := strings.Split(meta.Map, ".")
	if len(parts) != 2 {
		c.JSON(500, util.NewErrorResponse(errors.New("bad map path format")))
		return
	}
	col := lens.DefaultMongo().Client().Database(parts[0]).Collection(parts[1])
	// header
	header := col.FindOne(context.Background(), bson.M{"class": "header"})
	if header.Err() != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	var h mapHeader
	if err := header.Decode(&h); err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	xy2lnglat, err := proj.NewProjector(h.Data.Projection, WGS84CRS)
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	defer xy2lnglat.Close()
	lnglat2xy, err := proj.NewProjector(WGS84CRS, h.Data.Projection)
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	defer lnglat2xy.Close()
	minXY := lnglat2xy.Transform(&proj.Coord{X: meta.MinLat, Y: meta.MinLng})
	maxXY := lnglat2xy.Transform(&proj.Coord{X: meta.MaxLat, Y: meta.MaxLng})

	// lanes

	var parentIDFilter bson.D
	switch typ {
	case AllLane:
		parentIDFilter = bson.D{{Key: "$exists", Value: true}}
	case JunctionLane:
		parentIDFilter = bson.D{{Key: "$gte", Value: 300000000}}
	case RoadLane:
		parentIDFilter = bson.D{{Key: "$lt", Value: 300000000}}
	}

	cur, err := col.Aggregate(context.Background(), bson.A{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "class", Value: "lane"},
			{Key: "data.parent_id", Value: parentIDFilter},
			{Key: "data.center_line.nodes", Value: bson.D{
				{Key: "$elemMatch", Value: bson.D{
					{Key: "x", Value: bson.D{
						{Key: "$gte", Value: minXY.X},
						{Key: "$lte", Value: maxXY.X},
					}},
					{Key: "y", Value: bson.D{
						{Key: "$gte", Value: minXY.Y},
						{Key: "$lte", Value: maxXY.Y},
					}},
				}},
			}},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "id", Value: "$data.id"},
			{Key: "line", Value: "$data.center_line.nodes"},
			{Key: "type", Value: "$data.type"},
		}}},
	})

	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	var lanes []*mapLane
	if err := cur.All(context.Background(), &lanes); err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	inMicroscopic := func(n mapNode) bool {
		return minXY.X <= n.X && n.X <= maxXY.X &&
			minXY.Y <= n.Y && n.Y <= maxXY.Y
	}
	convertToLngLat := func(n mapNode, _ int) []float64 {
		c := xy2lnglat.Transform(&proj.Coord{X: n.X, Y: n.Y})
		return []float64{c.Y, c.X}
	}
	geojsons = lo.FilterMap(lanes, func(l *mapLane, _ int) (*geojson.Feature, bool) {
		if !lo.SomeBy(l.Line, inMicroscopic) {
			// 不在微观区域内，跳过
			return nil, false
		}
		coordinates := lo.Map(l.Line, convertToLngLat)
		geoLane := newGeoJsonLane(l.ID, l.Type, coordinates)
		return geoLane, true
	})
	finished = false
	return
}

// @Summary Load junction lane geojson
// @Produce application/json
// @Param tablename path string true "Simulation Name"
// @Success 200
// @Router /simple/junclane/{tablename} [get]
func GetJunclaneByName(c *gin.Context) {
	u := lens.ValidateUri(c)
	if u == nil {
		return
	}
	if geojsons, finished := downloadLanes(c, u.Name, JunctionLane); finished {
		return
	} else {
		c.JSON(200, util.NewResponse(geojsons))
	}
}

// @Summary Load road lane geojson in microscopic area
// @Produce application/json
// @Param tablename path string true "Simulation Name"
// @Success 200
// @Router /simple/all-roadlane/{tablename} [get]
func GetAllRoadlaneByName(c *gin.Context) {
	u := lens.ValidateUri(c)
	if u == nil {
		return
	}
	if geojsons, finished := downloadLanes(c, u.Name, RoadLane); finished {
		return
	} else {
		c.JSON(200, util.NewResponse(geojsons))
	}
}

// @Summary Load lane geojson in microscopic area
// @Produce application/json
// @Param tablename path string true "Simulation Name"
// @Success 200
// @Router /simple/all-lane/{tablename} [get]
func GetAllLaneByName(c *gin.Context) {
	u := lens.ValidateUri(c)
	if u == nil {
		return
	}
	if geojsons, finished := downloadLanes(c, u.Name, AllLane); finished {
		return
	} else {
		c.JSON(200, util.NewResponse(geojsons))
	}
}

// @Summary Load road lane geojson in microscopic area
// @Produce application/json
// @Param tablename path string true "Simulation Name"
// @Success 200
// @Router /simple/roadlane/{tablename} [get]
func GetRoadlaneByName(c *gin.Context) {
	u := lens.ValidateUri(c)
	if u == nil {
		return
	}

	metas, err := QueryMetadata(&u.Name)
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	} else if len(metas) == 0 {
		c.JSON(404, util.NewErrorResponse(errors.New("not found")))
		return
	}
	meta := metas[0]
	parts := strings.Split(meta.Map, ".")
	if meta.RoadStatusVMin == nil {
		c.JSON(400, util.NewErrorResponse(errors.New("no road status information")))
		return
	}
	if len(parts) != 2 {
		c.JSON(500, util.NewErrorResponse(errors.New("bad map path format")))
		return
	}
	col := lens.DefaultMongo().Client().Database(parts[0]).Collection(parts[1])
	// header
	header := col.FindOne(context.Background(), bson.M{"class": "header"})
	if header.Err() != nil {
		c.JSON(500, util.NewErrorResponse(header.Err()))
		return
	}
	var h mapHeader
	if err := header.Decode(&h); err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	xy2lnglat, err := proj.NewProjector(h.Data.Projection, WGS84CRS)
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	defer xy2lnglat.Close()

	// candidate lanes
	cur, err := col.Aggregate(context.Background(), bson.A{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "class", Value: "lane"},
			{Key: "data.parent_id", Value: bson.D{{Key: "$lt", Value: 300000000}}},             // road lane
			{Key: "data.type", Value: 1},                                                       // type: driving
			{Key: "data.max_speed", Value: bson.D{{Key: "$gte", Value: *meta.RoadStatusVMin}}}, // max_speed >= v_min
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "id", Value: "$data.id"},
			{Key: "line", Value: "$data.center_line.nodes"},
			{Key: "type", Value: "$data.type"},
		}}},
	})
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	var lanes []*mapLane
	if err := cur.All(context.Background(), &lanes); err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}

	// candidate road
	cur, err = col.Aggregate(context.Background(), bson.A{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "class", Value: "road"},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "id", Value: "$data.id"},
			{Key: "lane_ids", Value: "$data.lane_ids"},
		}}},
	})
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	var roads []*mapRoad
	if err := cur.All(context.Background(), &roads); err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}

	// 根据road找到对应的lanes
	id2Lanes := make(map[int32]*mapLane)
	for _, l := range lanes {
		id2Lanes[l.ID] = l
	}
	roadLanes := make([]*mapLane, 0)
	for _, r := range roads {
		// 找到合适的driving lane（最靠外的）
		for i := len(r.LaneIDs) - 1; i >= 0; i-- {
			if lane := id2Lanes[r.LaneIDs[i]]; lane != nil {
				lane.ID = r.ID // 用road的id替换lane的id
				roadLanes = append(roadLanes, lane)
				break
			}
		}
	}

	// 转换为geojson
	convertToLngLat := func(n mapNode, _ int) []float64 {
		c := xy2lnglat.Transform(&proj.Coord{X: n.X, Y: n.Y})
		return []float64{c.Y, c.X}
	}
	geojsons := lo.FilterMap(roadLanes, func(l *mapLane, _ int) (*geojson.Feature, bool) {
		coordinates := lo.Map(l.Line, convertToLngLat)
		geoLane := newGeoJsonLane(l.ID, l.Type, coordinates)
		return geoLane, true
	})

	c.JSON(200, util.NewResponse(geojsons))
}

// @Summary Load Aoi GeoJSON in microscopic area
// @Produce application/json
// @Param tablename path string true "Simulation Name"
// @Success 200
// @Router /simple/aoi/{tablename} [get]
func GetAoiByName(c *gin.Context) {
	u := lens.ValidateUri(c)
	if u == nil {
		return
	}

	metas, err := QueryMetadata(&u.Name)
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	} else if len(metas) == 0 {
		c.JSON(404, util.NewErrorResponse(errors.New("not found")))
		return
	}
	meta := metas[0]
	parts := strings.Split(meta.Map, ".")
	if len(parts) != 2 {
		c.JSON(500, util.NewErrorResponse(errors.New("bad map path format")))
		return
	}
	col := lens.DefaultMongo().Client().Database(parts[0]).Collection(parts[1])
	// header
	header := col.FindOne(context.Background(), bson.M{"class": "header"})
	if header.Err() != nil {
		c.JSON(500, util.NewErrorResponse(header.Err()))
		return
	}
	var h mapHeader
	if err := header.Decode(&h); err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	xy2lnglat, err := proj.NewProjector(h.Data.Projection, WGS84CRS)
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	defer xy2lnglat.Close()
	lnglat2xy, err := proj.NewProjector(WGS84CRS, h.Data.Projection)
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	defer lnglat2xy.Close()
	minXY := lnglat2xy.Transform(&proj.Coord{X: meta.MinLat, Y: meta.MinLng})
	maxXY := lnglat2xy.Transform(&proj.Coord{X: meta.MaxLat, Y: meta.MaxLng})

	// aoi
	cur, err := col.Aggregate(context.Background(), bson.A{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "class", Value: "aoi"},
			{Key: "data.area", Value: bson.D{{Key: "$exists", Value: true}}},
			{Key: "data.positions", Value: bson.D{
				{Key: "$elemMatch", Value: bson.D{
					{Key: "x", Value: bson.D{
						{Key: "$gte", Value: minXY.X},
						{Key: "$lte", Value: maxXY.X},
					}},
					{Key: "y", Value: bson.D{
						{Key: "$gte", Value: minXY.Y},
						{Key: "$lte", Value: maxXY.Y},
					}},
				}},
			}},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "id", Value: "$data.id"},
			{Key: "positions", Value: "$data.positions"},
		}}},
	})
	if err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	var aois []*mapAoi
	if err := cur.All(context.Background(), &aois); err != nil {
		c.JSON(500, util.NewErrorResponse(err))
		return
	}
	convertToLngLat := func(n mapNode, _ int) orb.Point {
		c := xy2lnglat.Transform(&proj.Coord{X: n.X, Y: n.Y})
		return orb.Point{c.Y, c.X}
	}
	geojsons := lo.Map(aois, func(a *mapAoi, _ int) *geojson.Feature {
		coordinates := lo.Map(a.Positions, convertToLngLat)
		polygon := orb.Polygon([]orb.Ring{coordinates})
		feature := geojson.NewFeature(polygon)
		feature.ID = a.ID
		feature.Properties = map[string]any{
			"id": a.ID,
		}
		return feature
	})

	c.JSON(200, util.NewResponse(geojsons))
}
