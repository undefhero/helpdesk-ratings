# Backend Software Engineer Test Task

Build a small [gRPC](https://grpc.io) service using the language of your choice. The preferred language for services in ZQA is [Go](https://golang.org).

The service should use the provided data from SQLite database (`database.db`). The data consists of **ratings to helpdesk tickets** - information about how well a given agent helped a customer.

Please share your solution with us in whatever format suitable (link to GitHub repository, ZIP of the project, etc).

### Tasks

1. Come up with a ticket scoring algorithm that accounts for rating category weights (available in the `rating_categories` table). 
Ratings are given on a scale of 0 to 5. The score should be representable in percentages from 0 to 100.
You need to use the algorithm in both of the endpoints in the second task.

2. Build a service that can be queried using [gRPC](https://grpc.io/docs/tutorials/basic/go/) calls and can answer following questions:

    * **Aggregated category scores over a period of time**
    
        I.e. what have been the daily ticket scores for a past week or what were the scores between 1st and 31st of January.

        For periods longer than one month weekly aggregates should be returned instead of daily values.

        Based on the response, the following UI representation should be possible:

        | Category | Ratings | Date 1 | Date 2 | ... | Score |
        |----|----|----|----|----|----|
        | Tone | 1 | 30% | N/A | N/A | X% |
        | Grammar | 2 | N/A | 90% | 100% | X% |
        | Random | 6 | 12% | 10% | 10% | X% |

    * **Overall quality score**

        I.e. what is the overall aggregate score for a period.

        E.g. the overall score over past week has been 96%.


### Bonus

* How would you build and deploy the solution?

    At ZQA we make heavy use of containers and [Kubernetes](https://kubernetes.io).


## Solution

### Deployment and launcing
The [./kubernetes/](./kubernetes/deployment.yaml) folder containis an example of how the service could be deployed to Kubernetes. 

**Important**: The database file is not included (by default) into Docker image. The `database.db` file has to be mounted to the container at runtime.
The list of available images at DockerHub: [aiprospace/helpdesk-ratings](https://hub.docker.com/r/aiprospace/helpdesk-ratings/tags)

Versions v0.3.1 and v0.3.1-db (with the database included) are final:
* [v0.3.1](https://hub.docker.com/layers/aiprospace/helpdesk-ratings/v0.3.1/images/sha256-acf3d51154c2f48b60d15858b6ce8ed7a0deb2d3915b480136ce6809fcf9cc29)
* [v0.3.1-db](https://hub.docker.com/layers/aiprospace/helpdesk-ratings/v0.3.1-db/images/sha256-260053a90a6e7ac9db3929abc1d72f2a5040f327a5857999374eca26617d10e8)

The service accepts environment variables:
|Variable    |Default Value.  |Description              |
|------------|----------------|-------------------------|
|SERVER_HOST |0.0.0.0         |Server address           |
|SERVER_PORT |"50051"         |gRPC server port         |
|DB_FILE_PATH|/app/database.db|SQLite database file path|

The repository also includes `docker-compose.yml` for local and remote service running.

It can be locally built and launched:
```bash
docker compose --profile local up
```
Or with remote image:
```bash
docker compose --profile remote up
```

### Implementation
The main logic and tests are on the [internal/service/](internal/service/) folder.

I also included test scenarios that I used during development.

## Test Scenarious

#### Base positive:
* One month January
```bash
grpcurl -plaintext -d '{
  "start_date": "2025-01-01T00:00:00Z",
  "end_date": "2025-01-31T23:59:59Z"
}' localhost:50051 ratings.Service/GetAggregatedScores
```

* One week
```bash
grpcurl -plaintext -d '{
  "start_date": "2025-01-01T00:00:00Z",
  "end_date": "2025-01-07T23:59:59Z"
}' localhost:50051 ratings.Service/GetAggregatedScores
```

* One day
```bash
grpcurl -plaintext -d '{
  "start_date": "2025-01-01T00:00:00Z",
  "end_date": "2025-01-01T23:59:59Z"
}' localhost:50051 ratings.Service/GetAggregatedScores
```

* One month and one day
```bash
grpcurl -plaintext -d '{
  "start_date": "2025-01-01T00:00:00Z",
  "end_date": "2025-02-01T23:59:59Z"
}' localhost:50051 ratings.Service/GetAggregatedScores
```

* One year
```bash
grpcurl -plaintext -d '{
  "start_date": "2025-01-01T00:00:00Z",
  "end_date": "2025-12-31T23:59:59Z"
}' localhost:50051 ratings.Service/GetAggregatedScores
```

* GetOverallScore a year
```bash
grpcurl -plaintext -d '{
  "start_date": "2025-01-01T00:00:00Z",
  "end_date": "2025-01-31T23:59:59Z"
}' localhost:50051 ratings.Service/GetOverallScore
```

#### Edge cases
* 28 days different, months
```bash
grpcurl -plaintext -d '{
  "start_date": "2025-01-06T00:00:00Z",
  "end_date": "2025-02-02T23:59:59Z"
}' localhost:50051 ratings.Service/GetAggregatedScores
```

* 29 days different, months
```bash
grpcurl -plaintext -d '{
  "start_date": "2025-01-06T00:00:00Z",
  "end_date": "2025-02-03T23:59:59Z"
}' localhost:50051 ratings.Service/GetAggregatedScores
```

Negative
* Start date after End date
```bash
grpcurl -plaintext -d '{
  "start_date": "2025-02-01T00:00:00Z",
  "end_date": "2025-01-01T23:59:59Z"
}' localhost:50051 ratings.Service/GetAggregatedScores
```

* No End date
```bash
grpcurl -plaintext -d '{
  "start_date": "2025-02-01T00:00:00Z"
}' localhost:50051 ratings.Service/GetAggregatedScores
```

* Empty
```bash
grpcurl -plaintext -d '{}' localhost:50051 ratings.Service/GetAggregatedScores
```

```bash
grpcurl -plaintext -d '{}' localhost:50051 ratings.Service/GetOverallScore
```
