package util

import (
	"errors"
	"math"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgconn"
)

func ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return math.Round(num*output) / output
}

func CheckIsTableNotFound(err error) bool {
	pgErr := &pgconn.PgError{}
	if errors.As(err, &pgErr) {
		if pgErr.Code == "42P01" { // table not found
			return true
		}
	}
	return false
}

func ResponseEmptyIfTableNotFound[T any](c *gin.Context, targetHint []T, err error) (shouldReturn bool) {
	if CheckIsTableNotFound(err) {
		c.JSON(200, NewResponse(make([]T, 0)))
		return true
	}
	return false
}
