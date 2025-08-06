package sim

import "droneops-sim/internal/enemy"

// EnemySpawner allows setting a callback used to spawn enemies.
type EnemySpawner interface {
	SetSpawner(func(enemy.Enemy))
}

// EnemyRemover allows setting a callback used to remove enemies.
type EnemyRemover interface {
	SetRemover(func(string))
}

// EnemyStatusUpdater allows setting a callback used to update enemy statuses.
type EnemyStatusUpdater interface {
	SetStatusUpdater(func(string, enemy.EnemyStatus))
}
