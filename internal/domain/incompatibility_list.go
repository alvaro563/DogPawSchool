package domain

func containsIncompatibility(list []Incompatibility, id int) bool {
	for _, v := range list {
		if v.ID() == id {
			return true
		}
	}
	return false
}

func removeIncompatibility(list []Incompatibility, id int) []Incompatibility {
	out := make([]Incompatibility, 0, len(list))
	for _, v := range list {
		if v.ID() != id {
			out = append(out, v)
		}
	}
	return out
}
