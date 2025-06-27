# AWX Deployer Development Container

This devcontainer configuration allows you to develop and test the AWX deployer in a consistent environment.

## Prerequisites

1. VS Code with the "Remote - Containers" extension
2. Docker Desktop running on your machine
3. A valid kubeconfig file at `~/.kube/config`

## Usage

1. Open this repository in VS Code
2. When prompted, click "Reopen in Container" (or use `Ctrl+Shift+P` and select "Remote-Containers: Reopen in Container")
3. VS Code will build the container using the same Dockerfile used in the pipeline
4. Your kubeconfig file will be automatically mounted at `/kubeconfig` inside the container
5. The environment variable `KUBECONFIG=/kubeconfig` is set automatically

## Testing the AWX Deployer

Once inside the devcontainer:

```bash
# The Go application is already built as /app/awx-deployer
./awx-deployer

# Or rebuild if you make changes
go build -o awx-deployer ./cmd/awx-deployer
./awx-deployer
```

## Kubeconfig Location

The devcontainer expects your kubeconfig file to be at `~/.kube/config` on your host machine. If your kubeconfig is elsewhere, you can:

1. Copy it to `~/.kube/config`, or
2. Modify the mount path in `.devcontainer/devcontainer.json`

## Benefits

- Same environment as the GitHub Actions pipeline
- No need to install Go, kubectl, or other dependencies locally
- Automatic kubeconfig mounting and environment setup
- Consistent development experience across different machines
