# Swarm Response Behaviour

This feature controls how many drones break formation to follow a detected enemy depending on their movement pattern.

| Movement Pattern | Behaviour | Purpose |
|------------------|-----------|---------|
| **patrol** | Only one additional drone is dispatched to follow the enemy while the rest continue patrolling. | Keeps patrol coverage while still tracking potential threats. |
| **point-to-point** | The detecting drone itself deviates from its route to pursue the target. | Maintains delivery or transport integrity without pulling extra units. |
| **loiter** | Up to two drones converge on the enemy position. | Provides focused attention in loiter scenarios where rapid response is desired. |

The mapping of patterns to follower counts can be configured under `swarm_responses` in `config/simulation.yaml`.

