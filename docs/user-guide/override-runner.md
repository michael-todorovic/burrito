# Override the runner pod spec

Both `TerraformRepository` and `TerraformLayer` expose a `spec.overrideRunnerSpec` map field.

If the field is specified for a given `TerraformRepository` it will be applied by default to all `TerraformLayer` linked to it.

If the field is specified for a given `TerraformLayer` it will take precedence over the `TerraformRepository` configuration.

!!! info
    - Maps (dictionaries): A deep merge is performed. Keys in `TerraformLayer` overwrite those in `TerraformRepository`, but unmatched keys are preserved.
    - Arrays (lists): Arrays are not merged; they are fully replaced.

Available overrides are:

|         Fields         |
|:----------------------:|
|       `Affinity`       |
|         `Args`         |
|       `Command`       |
|   `ImagePullSecrets`   |
|        `Image`         |
|    `InitContainers`    |
|     `Tolerations`      |
|     `NodeSelector`     |
|  `ServiceAccountName`  |
|      `Resources`       |
|         `Env`          |
|       `EnvFrom`        |
|       `Volumes`        |
|     `VolumeMounts`     |
| `Metadata.Annotations` |
|   `Metadata.Labels`    |
|    `ExtraInitArgs`     |
|    `ExtraPlanArgs`     |
|    `ExtraApplyArgs`    |

For instance with the following configuration, all the runner pods will have the specifications described inside the `TerraformRepository`:

```yaml
apiVersion: config.terraform.padok.cloud/v1alpha1
kind: TerraformRepository
metadata:
  name: burrito
  namespace: burrito
spec:
  repository:
    url: https://github.com/padok-team/burrito
  terraform:
    enabled: true
  overrideRunnerSpec:
    imagePullSecrets:
    - name: ghcr-creds
    tolerations:
    - effect: NoSchedule
      key: burrito.io/production
      operator: Exists
    nodeSelector:
      production: "true"
    serviceAccountName: "production"
---
apiVersion: config.terraform.padok.cloud/v1alpha1
kind: TerraformLayer
metadata:
  name: random-pets
  namespace: burrito
spec:
  terraform:
    version: "1.3.1"
  path: "internal/e2e/testdata/random-pets"
  branch: "main"
  repository:
    name: burrito
    namespace: burrito
```

In the following case, `nodeSelector` will be merged and `tolerations` will be replaced:

```yaml
apiVersion: config.terraform.padok.cloud/v1alpha1
kind: TerraformRepository
metadata:
  name: burrito
  namespace: burrito
spec:
  repository:
    url: https://github.com/padok-team/burrito
  terraform:
    enabled: true
  overrideRunnerSpec:
    imagePullSecrets:
    - name: ghcr-creds
    tolerations:
    - effect: NoExecute
      key: burrito.io/production
      operator: Exists
    nodeSelector:
      production: "true"
    serviceAccountName: "production"
---
apiVersion: config.terraform.padok.cloud/v1alpha1
kind: TerraformLayer
metadata:
  name: random-pets
  namespace: burrito
spec:
  terraform:
    version: "1.3.1"
  path: "internal/e2e/testdata/random-pets"
  branch: "main"
  repository:
    name: burrito
    namespace: burrito
  overrideRunnerSpec:
    tolerations:
    - effect: NoSchedule
      key: burrito.io/production
      operator: Exists
    nodeSelector: {}
```

Resulting in the following `podSpec`:

```yaml
tolerations:
- effect: NoSchedule
  key: burrito.io/production
  operator: Exists
nodeSelector:
  production: "true"
```

## Prepare runner configuration with `initContainers`

You may need to get refreshed tokens to access your APIs or prepare some dynamic configuration for Terraform. To do so, you can use `initContainers`.

!!! note
    In order to allow access to files generated by your init container from your main container, you'll need to match the uid/gid from the runner image. You can check it [here](https://github.com/padok-team/burrito/blob/main/Dockerfile)

```yaml
apiVersion: config.terraform.padok.cloud/v1alpha1
kind: TerraformLayer
metadata:
  name: my-layer
spec:
  overrideRunnerSpec:
    volumes:
      - name: kubeconfig-volume
        emptyDir: {}
    volumeMounts:
      - name: kubeconfig-volume
        mountPath: /home/burrito/.kube
        readOnly: true
    initContainers:
      - name: init-runner
        image: registry.domain.com/image:tag
        securityContext:
          runAsNonRoot: true
          runAsUser: 65532  # match USER from Dockerfile
          runAsGroup: 65532 # match USER from Dockerfile
        command:
          - 'sh'
          - '-c'
          - 'generate-kube-config --output=/.kube/config'
        envFrom:
          - secretRef:
              name: kube-secrets
        volumeMounts:
          - mountPath: /.kube
            name: kubeconfig-volume
```
