package distsys_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"learning-marketplace/internal/distsys"
)

func TestClockSkewScenario_LastWriteWinsDropsLaterWrite(t *testing.T) {
	scenario := distsys.ClockSkewScenario()

	require.Equal(t, "x=1", scenario.Winner.Value)
	require.Equal(t, "x=2", scenario.LostUpdateFor)
	require.Greater(t, scenario.Earlier.TimestampMS, scenario.Later.TimestampMS)
}

func TestProcessPauseScenario_FencingRejectsZombieWrite(t *testing.T) {
	scenario := distsys.ProcessPauseScenario()

	require.True(t, scenario.AcceptedWrite.Accepted)
	require.False(t, scenario.RejectedWrite.Accepted)
	require.Equal(t, scenario.AcceptedWrite.Value, scenario.FinalValue)
	require.EqualValues(t, 34, scenario.FinalFenceToken)
}

func TestDelayedRequestScenario_FencingRejectsLateArrival(t *testing.T) {
	scenario := distsys.DelayedRequestScenario()

	require.True(t, scenario.AcceptedWrite.Accepted)
	require.False(t, scenario.RejectedWrite.Accepted)
	require.Equal(t, scenario.AcceptedWrite.Value, scenario.FinalValue)
	require.EqualValues(t, 41, scenario.FinalFenceToken)
}
