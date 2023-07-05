package main

import (
	"encoding/json"
	"fmt"
)

type JSONMapper struct{}

func (M *JSONMapper) Map(message []byte) (JobMasterRow, error) {
	var jobMasterRow JobMasterRow
	err := json.Unmarshal(message, &jobMasterRow)
	if err != nil {
		return jobMasterRow, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return jobMasterRow, nil
}
