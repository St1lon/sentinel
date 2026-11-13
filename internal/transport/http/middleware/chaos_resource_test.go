package middleware

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Проверки предохранителей ресурсных режимов: они читают состояние пакета
// напрямую, поэтому живут в package middleware (chaos_test.go — во внешнем
// пакете) и без t.Parallel — счётчики общие.

func TestChaos_GoroutineLeakStopsAtCap(t *testing.T) {
	// Счётчик общий на пакет, поэтому подкатываем его к потолку вручную,
	// а не спавним тысячи настоящих горутин — иначе этот тест насытил бы
	// предохранитель для всех остальных. Cleanup возвращает ноль: следующий
	// тест должен уметь спавнить.
	leakedGoroutines.Store(int64(leakedGoroutinesCap) - 5)
	t.Cleanup(func() { leakedGoroutines.Store(0) })

	leakGoroutines(leakedGoroutinesCap + 100)

	require.Equal(t, int64(leakedGoroutinesCap), leakedGoroutines.Load(),
		"предохранитель останавливает утечку ровно на потолке: под не должен задохнуться")
}

func TestChaos_MemHogHoldsAllocatedBytes(t *testing.T) {
	before := hoggedBytes.Load()

	hogMemory(1 << 20)

	require.GreaterOrEqual(t, hoggedBytes.Load(), before+(1<<20),
		"аллокация удерживается, а не отдается сборщику мусора")
}

func TestChaos_MemHogStopsAtCap(t *testing.T) {
	// На порядок больше, чем помещается под предохранителем.
	for range (hoggedBytesCap >> 20) + 8 {
		hogMemory(1 << 20)
	}

	require.LessOrEqual(t, hoggedBytes.Load(), int64(hoggedBytesCap),
		"больше потолка память не удерживается")
	require.Greater(t, hoggedBytes.Load(), int64(hoggedBytesCap-(2<<20)),
		"до потолка память доходит: порог алерта обязан быть достижим")
}
