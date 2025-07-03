
## fork
```bash

go run main.go fork --dev-token "glpat-Uou_WTfqMyWn9wyZ_HNX" --prod-token "glpat-5QL4aihz5PSymiALe1Uv" --source-group fy-dev --source-project iris --target-group fy-prod -k

```

## list-projects

### fy-dev

```bash
go run main.go list-projects -g fy-dev -t "glpat-Uou_WTfqMyWn9wyZ_HNX" -k
```
### fy-prod
```bash
go run main.go list-projects -g fy-prod -t "glpat-5QL4aihz5PSymiALe1Uv" -k

```

## clone 

```bash

go run main.go clone \
 --from-repo-url "https://aml-gitlab.alaudatech.net/fy-dev/amlmodels/iris" \
 --to-repo-url "https://aml-gitlab.alaudatech.net/fy-prod/amlmodels/iris" \
 --from-ref v0.0.1 \
 --from-token "glpat-Uou_WTfqMyWn9wyZ_HNX" \
 --to-token "glpat-5QL4aihz5PSymiALe1Uv" \
 --on-tag-exists "error"
```