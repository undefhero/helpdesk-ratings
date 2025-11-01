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
