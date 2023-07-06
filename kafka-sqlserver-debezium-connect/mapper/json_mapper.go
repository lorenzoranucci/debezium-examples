package main

import (
	"encoding/json"
	"fmt"
	"time"
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

func (j *JobMasterRow) UnmarshalJSON(data []byte) error {
	type Alias JobMasterRow
	aux := &struct {
		JobStartDate     int64 `json:"JobStartDate"`
		JobQuoteTimeLast int64 `json:"JobQuoteTimeLast"`
		CreateDate       int64 `json:"CreateDate"`
		*Alias
	}{
		Alias: (*Alias)(j),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	j.JobStartDate = time.Unix(aux.JobStartDate/1000, 0)
	j.JobQuoteTimeLast = time.Unix(aux.JobQuoteTimeLast/1000, 0)
	j.CreateDate = time.Unix(aux.CreateDate/1000, 0)

	return nil
}
