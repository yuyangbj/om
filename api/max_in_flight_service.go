package api

type MaxInFlightProperties struct {
	Properties map[string]interface{} `json:"max_in_flight,omitempty"`
}
