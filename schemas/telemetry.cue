package schemas

import "time"

#Telemetry: {
        cluster_id: string
        drone_id: string
        mission_id: string
        lat: number
        lon: number
        alt: number
        battery: number
        status: string
        follow: bool
        movement_pattern: string
        speed_mps: number
        heading_deg: number
        previous_position: {
                lat: number
                lon: number
                alt: number
        }
        synced_from?: string
        synced_id?: string
        synced_at?: time.Time
        ts: time.Time
}
