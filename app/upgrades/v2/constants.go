package v2

import (
	storetypes "cosmossdk.io/store/types"

	"github.com/dgtlkitchen/vrf/app/upgrades"
)

const UpgradeName = "v2"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV2UpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{},
}
