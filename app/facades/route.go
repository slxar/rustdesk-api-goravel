package facades

import (
	"github.com/goravel/framework/contracts/route"
	"github.com/goravel/framework/foundation"
)

func Route() route.Route { return foundation.App.MakeRoute() }
