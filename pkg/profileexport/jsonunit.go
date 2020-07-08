package profileexport

func newJsonUnit(unitListEntry map[string]interface{}) jsonUnit {
	return jsonUnit{
		UnitId:     uint64(unitListEntry["unit_id"].(float64)),
		BuildingId: uint64(unitListEntry["building_id"].(float64)),
		UnitLevel:  uint(unitListEntry["unit_level"].(float64)),
		Class:      uint(unitListEntry["class"].(float64)),
		Attribute:  uint(unitListEntry["attribute"].(float64)),
	}
}

type jsonUnit struct {
	UnitId     uint64 `json:"unit_id"`
	BuildingId uint64 `json:"building_id"`
	UnitLevel  uint   `json:"unit_level"`
	Class      uint   `json:"class"`
	Attribute  uint   `json:"attribute"`
}
