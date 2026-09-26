project := "random-number-api-509815"

default:
    @just --list

# Build from the root Dockerfile and deploy the public API to Cloud Run.
deploy service="random-number-api" region="us-central1":
    gcloud run deploy {{service}} --project={{project}} --region={{region}} --source=. --port=4000 --allow-unauthenticated
