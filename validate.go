package freightplan

import "errors"

// validate checks basic input invariants required by the business rules.
func validate(order []Unit, offers []Offer) error {
	ids := make(map[string]bool, len(order))
	for _, u := range order {
		if u.ID == "" {
			return errors.New("freightplan: unit with empty ID")
		}
		if ids[u.ID] {
			return errors.New("freightplan: duplicate unit ID " + u.ID)
		}
		ids[u.ID] = true
		if u.WeightG <= 0 {
			return errors.New("freightplan: unit " + u.ID + " has non-positive weight")
		}
	}
	offerIDs := make(map[string]bool, len(offers))
	for _, o := range offers {
		if o.ID == "" {
			return errors.New("freightplan: offer with empty ID")
		}
		if offerIDs[o.ID] {
			return errors.New("freightplan: duplicate offer ID " + o.ID)
		}
		offerIDs[o.ID] = true
		if o.MaxWeightG <= 0 {
			return errors.New("freightplan: offer " + o.ID + " has non-positive weight limit")
		}
		if o.FeeCents <= 0 {
			return errors.New("freightplan: offer " + o.ID + " has non-positive fee")
		}
	}
	return nil
}
