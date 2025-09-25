## Deployment in Kubernetes

### Using the Helm Chart

The `droneops-sim` project includes a Helm chart for deploying the simulator in Kubernetes. Follow these steps to deploy:

1. **Install Helm**:
   Ensure Helm is installed on your system. Refer to the [Helm installation guide](https://helm.sh/docs/intro/install/) if needed.

2. **Navigate to the Helm chart directory**:

   ```bash
   cd helm/droneops-sim
   ```

3. **Customize values**:

   Edit the `values.yaml` file to configure replicas, image, service type, resources, and simulation configuration.

4. **Deploy the chart**:

   Run the following command to deploy:

   ```bash
   helm install droneops-sim .
   ```

   This will deploy the simulator with the default configuration.

5. **Verify deployment**:

   Check the status of the deployment:

   ```bash
   kubectl get all -l app=droneops-sim
   ```

6. **Access the service**:

   The simulator exposes a metrics endpoint. Use the following command to get the service details:

   ```bash
   kubectl get svc droneops-sim
   ```

### Notes

- The Helm chart uses ConfigMaps to manage simulation and schema configurations.
- Ensure the Kubernetes cluster has sufficient resources to handle the configured replicas and resource limits.
- Update the `GREPTIMEDB_ENDPOINT`, `GREPTIMEDB_DATABASE`, `GREPTIMEDB_TABLE`, and `ENEMY_DETECTION_TABLE` environment variables in the deployment if connecting to a real database.

For more details, refer to the `helm/droneops-sim` directory and the `values.yaml` file.

