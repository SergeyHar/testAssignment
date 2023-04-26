# Project instructions:

## Run application
```go run .``` 

## Upload csv files

```curl --location --request POST  'http://localhost:8080/promotions/upload' --form 'file=@"file_path"'```

## Get data by ID
```curl --location --request GET 'http://localhost:8080/promotions/:id'```

