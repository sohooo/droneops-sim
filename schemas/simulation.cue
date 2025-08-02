// CUE schema content for simulation.yaml
package schemas

zones: [...{
    name:        string & !=""
    center_lat:  number
    center_lon:  number
    radius_km:   number & >0
}]

missions: [...{
    name:        string & !=""
    zone:        string
    description: string
}]

fleets: [...{
    name:             string & !=""
    model:            =~"small-fpv|medium-uav|large-uav"
    count:            int & >0
    movement_pattern: =~"patrol|point-to-point|loiter"
    home_region:      string
    behavior?: {
        battery_drain_rate?: number & >=0
        failure_rate?:       number & >=0 & <=1
        speed_min_kmh?:      number & >=0
        speed_max_kmh?:      number & >=0
        sensor_error_rate?:  number & >=0 & <=1
        dropout_rate?:       number & >=0 & <=1
        battery_anomaly_rate?: number & >=0 & <=1
    }
}]

follow_confidence?: number & >=0 & <=100

swarm_responses?: { [=~"patrol|point-to-point|loiter"]: int }

mission_criticality?: =~"low|medium|high"

enemy_count?: int & >=0

detection_radius_m?: number & >0
sensor_noise?: number & >=0
terrain_occlusion?: number & >=0 & <=1
weather_impact?: number & >=0 & <=1
