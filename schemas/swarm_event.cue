package schemas

import "time"

#SwarmEvent: {
        cluster_id: string
        event_type: string
        drone_ids: [...string]
        enemy_id?: string
        ts: time.Time
}

