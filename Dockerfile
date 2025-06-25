FROM python:3.11-slim

# Install kubectl
RUN apt-get update && \
    apt-get install -y curl && \
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/ && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy deployment script
COPY deploy_awx.py /app/deploy_awx.py

# Make script executable
RUN chmod +x /app/deploy_awx.py

# Set entrypoint
ENTRYPOINT ["python3", "/app/deploy_awx.py"]
