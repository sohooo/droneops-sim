package schemas

import "time"

#Detection: {
        cluster_id: string
        drone_id:   string
        enemy_id:   string
        enemy_type: string
        lat:        number
        lon:        number
        alt:        number
        drone_lat:  number
        drone_lon:  number
        drone_alt:  number
        distance_m: number
        bearing_deg: number
        enemy_velocity_mps: number
        confidence: number & >=0 & <=100
        ts:         time.Time
}

