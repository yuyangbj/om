package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type MaxInFlightProperties struct {
	Properties map[string]interface{} `json:"max_in_flight,omitempty"`
}

func (a Api) UpdateStagedProductMaxInFlight(productID string, maxInFlight MaxInFlightProperties) error {
	path := fmt.Sprintf("/api/v0/staged/products/%s/max_in_flight", productID)

	bodyBytes := bytes.NewBuffer([]byte{})
	err := json.NewEncoder(bodyBytes).Encode(maxInFlight)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", path, bodyBytes)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("could not make api request to max_in_flight endpoint: %s", err)
	}

	defer resp.Body.Close()

	if err = validateStatusOK(resp); err != nil {
		return err
	}

	return nil
}
