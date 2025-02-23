package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/golang/mock/gomock"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	authztestutil "github.com/cosmos/cosmos-sdk/x/authz/testutil"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var (
	granteePub  = secp256k1.GenPrivKey().PubKey()
	granterPub  = secp256k1.GenPrivKey().PubKey()
	granteeAddr = sdk.AccAddress(granteePub.Address())
	granterAddr = sdk.AccAddress(granterPub.Address())
)

type GenesisTestSuite struct {
	suite.Suite

	ctx           sdk.Context
	keeper        keeper.Keeper
	baseApp       *baseapp.BaseApp
	accountKeeper *authztestutil.MockAccountKeeper
	encCfg        moduletestutil.TestEncodingConfig
}

func (suite *GenesisTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(keeper.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx.WithBlockHeader(cmtproto.Header{Height: 1})
	suite.encCfg = moduletestutil.MakeTestEncodingConfig(authzmodule.AppModuleBasic{})

	// gomock initializations
	ctrl := gomock.NewController(suite.T())
	suite.accountKeeper = authztestutil.NewMockAccountKeeper(ctrl)

	suite.accountKeeper.EXPECT().StringToBytes(granteeAddr.String()).Return(granteeAddr, nil).AnyTimes()
	suite.accountKeeper.EXPECT().BytesToString(granterAddr).Return(granterAddr.String(), nil).AnyTimes()
	suite.accountKeeper.EXPECT().StringToBytes(granterAddr.String()).Return(granterAddr, nil).AnyTimes()
	suite.accountKeeper.EXPECT().BytesToString(granterAddr).Return(granterAddr.String(), nil).AnyTimes()

	suite.baseApp = baseapp.NewBaseApp(
		"authz",
		log.NewNopLogger(),
		testCtx.DB,
		suite.encCfg.TxConfig.TxDecoder(),
	)
	suite.baseApp.SetCMS(testCtx.CMS)

	bank.RegisterInterfaces(suite.encCfg.InterfaceRegistry)

	msr := suite.baseApp.MsgServiceRouter()
	msr.SetInterfaceRegistry(suite.encCfg.InterfaceRegistry)

	suite.keeper = keeper.NewKeeper(key, suite.encCfg.Codec, msr, suite.accountKeeper)
}

func (suite *GenesisTestSuite) TestImportExportGenesis() {
	coins := sdk.NewCoins(sdk.NewCoin("foo", sdkmath.NewInt(1_000)))

	now := suite.ctx.BlockTime()
	expires := now.Add(time.Hour)
	grant := &bank.SendAuthorization{SpendLimit: coins}
	err := suite.keeper.SaveGrant(suite.ctx, granteeAddr, granterAddr, grant, &expires)
	suite.Require().NoError(err)
	genesis := suite.keeper.ExportGenesis(suite.ctx)

	// TODO, recheck!
	// Clear keeper
	suite.keeper.DeleteGrant(suite.ctx, granteeAddr, granterAddr, grant.MsgTypeURL())

	suite.keeper.InitGenesis(suite.ctx, genesis)
	newGenesis := suite.keeper.ExportGenesis(suite.ctx)
	suite.Require().Equal(genesis, newGenesis)
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}
