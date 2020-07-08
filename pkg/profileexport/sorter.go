package profileexport

func abs(n int64) int64 {
	y := n >> 63
	return (n ^ y) - y
}

// Sort units according to the rank etc.
type CraftItemsByTypeAndId []interface{}

func (r CraftItemsByTypeAndId) Len() int { return len(r) }

func (r CraftItemsByTypeAndId) Less(i, j int) bool {
	a := r[i].(map[string]interface{})
	b := r[j].(map[string]interface{})

	typeA := uint(a["craft_type"].(float64))
	typeB := uint(b["craft_type"].(float64))

	itemIdA := uint(a["craft_item_id"].(float64))
	itemIdB := uint(b["craft_item_id"].(float64))

	if abs(int64(typeB-typeA)) != 0 {
		return typeA < typeB
	}

	if abs(int64(itemIdB-itemIdA)) != 0 {
		return itemIdA > itemIdB
	}

	return false
}

func (r CraftItemsByTypeAndId) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

// Sort interface for runes
type RunesBySlot []interface{}

func (r RunesBySlot) Len() int { return len(r) }

func (r RunesBySlot) Less(i, j int) bool {
	runeA := r[i].(map[string]interface{})
	runeB := r[j].(map[string]interface{})

	slotA := uint(runeA["slot_no"].(float64))
	slotB := uint(runeB["slot_no"].(float64))

	return slotA < slotB
}

func (r RunesBySlot) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
