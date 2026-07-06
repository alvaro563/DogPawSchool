package dog

func validateIncompatibilityInput(dogID int, incompatID int) error {
	if dogID <= 0 {
		return &ValidationError{Field: "dog_id"}
	}
	if incompatID <= 0 {
		return &ValidationError{Field: "incompatibility_id"}
	}
	return nil
}
