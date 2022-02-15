package repo

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStoragePath(t *testing.T) {
	p := GetStoragePath("/data", "order")
	assert.Equal(t, p, "/data/storage/order")
	p = GetStoragePath("/data")
	assert.Equal(t, p, "/data/storage")

	_, err := Load("testdata", "", "", "")
	require.Nil(t, err)

	_, err = GetAPI("testdata")
	require.Nil(t, err)

	path := GetKeyPath("testdata")
	require.Contains(t, path, KeyName)
}

func TestCheckStrategyInfo(t *testing.T) {
	// illegal type
	err := CheckStrategyInfo("", AppchainMgr, "", 0)
	require.NotNil(t, err)

	// illegal exp
	err = CheckStrategyInfo(SimpleMajority, AppchainMgr, "+", 0)
	require.NotNil(t, err)
	err = CheckStrategyInfo(SimpleMajority, AppchainMgr, "a==1", 0)
	require.NotNil(t, err)

	// illegal module
	err = CheckStrategyInfo(SimpleMajority, "", "a==1", 1)
	require.NotNil(t, err)

	err = CheckStrategyInfo(SimpleMajority, AppchainMgr, "a==1", 1)
	require.Nil(t, err)
}

func TestMakeStrategyDecision(t *testing.T) {
	isOver, isPass, err := MakeStrategyDecision("a==4", 0, 0, 4, 4)
	require.False(t, isOver)
	require.False(t, isPass)
	require.Nil(t, err)

	isOver, isPass, err = MakeStrategyDecision("a==4", 4, 0, 4, 4)
	require.True(t, isOver)
	require.True(t, isPass)
	require.Nil(t, err)

	isOver, isPass, err = MakeStrategyDecision("a==4", 0, 1, 4, 4)
	require.True(t, isOver)
	require.False(t, isPass)
	require.Nil(t, err)

	isOver, isPass, err = MakeStrategyDecision("+", 0, 1, 4, 4)
	require.False(t, isOver)
	require.False(t, isPass)
	require.NotNil(t, err)

	isOver, isPass, err = MakeStrategyDecision("1+1", 0, 1, 4, 4)
	require.False(t, isOver)
	require.False(t, isPass)
	require.NotNil(t, err)
}
