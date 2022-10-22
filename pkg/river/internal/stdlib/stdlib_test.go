package stdlib_test

import (
	"testing"

	"github.com/grafana/agent/pkg/river/internal/stdlib"
	"github.com/grafana/agent/pkg/river/internal/value"
	"github.com/stretchr/testify/require"
)

func Test_map(t *testing.T) {
	mapFn := value.Func(stdlib.Functions["map"])

	t.Run("double", func(t *testing.T) {
		fn := value.Func(func(i int) int { return i * 2 })
		arr := value.Array(
			value.Int(1),
			value.Int(2),
			value.Int(3),
			value.Int(4),
			value.Int(5),
		)

		res, err := mapFn.Call(fn, arr)
		require.NoError(t, err)

		var nums []int
		require.NoError(t, value.Decode(res, &nums))
		require.Equal(t, []int{2, 4, 6, 8, 10}, nums)
	})
}
