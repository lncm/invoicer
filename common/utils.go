package common

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func Max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func FormatRoutes(ginRoutes gin.RoutesInfo) (routes []string) {
	var maxM, maxP int
	for _, r := range ginRoutes {
		maxM = Max(maxM, len(r.Method))
		maxP = Max(maxP, len(r.Path))
	}

	for _, r := range ginRoutes {
		routes = append(routes, fmt.Sprintf(
			fmt.Sprintf("%%-%ds %%-%ds -> %%s", maxM, maxP),
			r.Method,
			r.Path,
			r.Handler,
		))
	}

	return
}
