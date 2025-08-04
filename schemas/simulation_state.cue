package schemas

import "time"

#SimulationState: {
        cluster_id: string
        communication_loss: number
        messages_sent: int
        sensor_noise: number
        weather_impact: number
        chaos_mode: bool
        ts: time.Time
}

