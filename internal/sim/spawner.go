package sim

import "droneops-sim/internal/enemy"

// EnemySpawner allows setting a callback used to spawn enemies.
type EnemySpawner interface {
	SetSpawner(func(enemy.Enemy))
}
