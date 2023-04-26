package modules

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/storage/sectorblocks"

	"go.uber.org/fx"

	"github.com/filecoin-project/boost/api"
	"github.com/filecoin-project/boost/storagemarket/sealingpipeline"

	lapi "github.com/filecoin-project/lotus/api"
	lclient "github.com/filecoin-project/lotus/api/client"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/filecoin-project/lotus/node/modules/helpers"
)

type MinerSealingService lapi.StorageMiner
type MinerStorageService lapi.StorageMiner

var _ sectorblocks.SectorBuilder = *new(MinerSealingService)

var _ sealingpipeline.API = *new(MinerSealingService)

func connectMinerService(apiInfo string) func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (lapi.StorageMiner, error) {
	return func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (lapi.StorageMiner, error) {
		ctx := helpers.LifecycleCtx(mctx, lc)
		info := cliutil.ParseApiInfo(apiInfo)
		addr, err := info.DialArgs("v0")
		if err != nil {
			return nil, fmt.Errorf("could not get DialArgs: %w", err)
		}

		log.Infof("Checking (svc) api version of %s", addr)

		mapi, closer, err := lclient.NewStorageMinerRPCV0(ctx, addr, info.AuthHeader())
		if err != nil {
			return nil, err
		}
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				v, err := mapi.Version(ctx)
				if err != nil {
					return fmt.Errorf("checking version: %w", err)
				}

				if !v.APIVersion.EqMajorMinor(lapi.Version(api.MinerAPIVersion0)) {
					return fmt.Errorf("remote service API version didn't match (expected %s, remote %s)", api.MinerAPIVersion0, v.APIVersion)
				}

				return nil
			},
			OnStop: func(context.Context) error {
				closer()
				return nil
			}})

		return mapi, nil
	}
}

func ConnectSealingService(apiInfo string) func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (MinerSealingService, error) {
	return func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (MinerSealingService, error) {
		log.Info("Connecting sealing service to miner")
		return connectMinerService(apiInfo)(mctx, lc)
	}
}

func ConnectStorageService(apiInfo string) func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (MinerStorageService, error) {
	return func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (MinerStorageService, error) {
		log.Info("Connecting storage service to miner")
		return connectMinerService(apiInfo)(mctx, lc)
	}
}

// ConnectStorageServices connect several miners
func ConnectSealingServices(apiInfos []string) func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (*sealingpipeline.AllWorkers, error) {
	return func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (*sealingpipeline.AllWorkers, error) {
		log.Infof("Connecting sealing services to miner: api infos=%s", apiInfos)
		allWorkers := &sealingpipeline.AllWorkers{}
		for _, apiInfo := range apiInfos {
			sm, e := connectMinerService(apiInfo)(mctx, lc)
			if e == nil {
				allWorkers.Workers = append(allWorkers.Workers, sm)
			}
		}
		return allWorkers, nil
	}
}

// ConnectStorageServices connect several miners
func ConnectStorageServices(apiInfos []string) func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (sectorblocks.AllSectorBuilders, error) {
	return func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (sectorblocks.AllSectorBuilders, error) {
		log.Infof("Connecting storage services to miner: api infos=%s", apiInfos)
		builders := sectorblocks.AllSectorBuilders{}
		for _, apiInfo := range apiInfos {
			sm, e := connectMinerService(apiInfo)(mctx, lc)
			if e == nil {
				builders.SectorBuilders = append(builders.SectorBuilders, sm)
			}
		}
		return builders, nil
	}
}
