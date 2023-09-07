package main

import (
	"encoding/json"
	"time"
)

type resources []Resource

type Report struct {
	Time           time.Time `json:"timestamp"`
	Found          resources `json:"found"`
	Deleted        resources `json:"deleted"`
	FailedToDelete resources `json:"failed_to_delete"`
}

func (rep *Report) AddFound(r Resource) {
	rep.Found = append(rep.Found, r)
}

func (rep *Report) AddDeleted(r Resource) {
	rep.Deleted = append(rep.Deleted, r)
}

func (rep *Report) AddFailedToDelete(r Resource) {
	rep.FailedToDelete = append(rep.FailedToDelete, r)
}

func (res resources) MarshalJSON() ([]byte, error) {
	type resourcePrinter struct {
		ClusterID   string    `json:"cluster_id,omitempty"`
		ID          string    `json:"id"`
		LastUpdated time.Time `json:"last_updated"`
		Name        string    `json:"name"`
		Type        string    `json:"type"`
	}

	printers := make([]resourcePrinter, len(res))

	for i := range res {
		var clusterID string
		if c, ok := res[i].(Clusterer); ok {
			clusterID = c.ClusterID()
		}
		printers[i] = resourcePrinter{
			ClusterID:   clusterID,
			ID:          res[i].ID(),
			LastUpdated: res[i].LastUpdated(),
			Name:        res[i].Name(),
			Type:        res[i].Type(),
		}
	}
	return json.Marshal(printers)
}
