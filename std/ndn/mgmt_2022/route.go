package mgmt_2022

type RouteFlag uint64

const (
	RouteFlagNoFlag       RouteFlag = 0
	RouteFlagChildInherit RouteFlag = 1
	RouteFlagCapture      RouteFlag = 2
)

var RouteFlagList = map[RouteFlag]string{
	RouteFlagChildInherit: "child-inherit",
	RouteFlagCapture:      "capture",
}

// Returns the human-readable name of the RouteFlag if it exists in the predefined RouteFlagList, otherwise returns "unknown".  

**Explanation:**  
This `String()` method converts a `RouteFlag` enum value to its corresponding string representation by looking it up in `RouteFlagList`, which maps valid flag values to their names. If the value is not found in the list, it returns `"unknown"`.
func (v RouteFlag) String() string {
	if s, ok := RouteFlagList[v]; ok {
		return s
	}
	return "unknown"
}

// Returns true if the specified RouteFlag is set within the provided bitmask flags.
func (v RouteFlag) IsSet(flags uint64) bool {
	return uint64(v)&flags != 0
}

type RouteOrigin uint64

const (
	RouteOriginApp       RouteOrigin = 0
	RouteOriginStatic    RouteOrigin = 255
	RouteOriginNLSR      RouteOrigin = 128
	RouteOriginPrefixAnn RouteOrigin = 129
	RouteOriginClient    RouteOrigin = 65
	RouteOriginAutoreg   RouteOrigin = 64
	RouteOriginAutoconf  RouteOrigin = 66
)

var RouteOriginList = map[RouteOrigin]string{
	RouteOriginApp:       "app",
	RouteOriginStatic:    "static",
	RouteOriginNLSR:      "nlsr",
	RouteOriginPrefixAnn: "prefixann",
	RouteOriginClient:    "client",
	RouteOriginAutoreg:   "autoreg",
	RouteOriginAutoconf:  "autoconf",
}

// Returns the string representation of the RouteOrigin value by looking it up in RouteOriginList, or "unknown" if the value is not present in the map.
func (v RouteOrigin) String() string {
	if s, ok := RouteOriginList[v]; ok {
		return s
	}
	return "unknown"
}
