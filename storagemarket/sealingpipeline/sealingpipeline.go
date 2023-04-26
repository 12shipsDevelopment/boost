package sealingpipeline

//go:generate go run github.com/golang/mock/mockgen -destination=mock/sealingpipeline.go -package=mock . API

import (
	"context"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/storage/sealer/storiface"
	"github.com/google/uuid"
)

type API interface {
	ActorAddress(context.Context) (address.Address, error)
	WorkerJobs(context.Context) (map[uuid.UUID][]storiface.WorkerJob, error)
	SectorsStatus(ctx context.Context, sid abi.SectorNumber, showOnChainInfo bool) (api.SectorInfo, error)
	SectorsList(context.Context) ([]abi.SectorNumber, error)
	SectorsSummary(ctx context.Context) (map[api.SectorState]int, error)
	SectorsListInStates(context.Context, []api.SectorState) ([]abi.SectorNumber, error)
}

type AllWorkers struct {
	API
	Workers []API
}

func NewAllWorkers(workers []API, api API) *AllWorkers {
	return &AllWorkers{
		API:     api,
		Workers: workers,
	}
}

func GetStatus(ctx context.Context, wks *AllWorkers) (*Status, error) {
	summaryAll := make(map[api.SectorState]int)
	workersAll := make([]*worker, 0)
	for _, wk := range wks.Workers {
		res, err := wk.WorkerJobs(ctx)
		if err != nil {
			return nil, err
		}

		for workerId, jobs := range res {
			for _, j := range jobs {
				workersAll = append(workersAll, &worker{
					ID:     workerId.String(),
					Start:  j.Start,
					Stage:  j.Task.Short(),
					Sector: int32(j.Sector.Number),
				})
			}
		}

		summary, err := wk.SectorsSummary(ctx)
		if err != nil {
			return nil, err
		}
		for k, v := range summary {
			summaryAll[k] = v
		}
	}

	st := &Status{
		SectorStates: summaryAll,
		Workers:      workersAll,
	}

	return st, nil
}

type worker struct {
	ID     string
	Start  time.Time
	Stage  string
	Sector int32
}

// TODO: maybe add json tags
type Status struct {
	SectorStates map[api.SectorState]int
	Workers      []*worker
}
