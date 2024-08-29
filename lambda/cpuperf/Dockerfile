FROM golang:1.23 as build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY cmd/main.go .

RUN go build -tags lambda.norpc -o main main.go

FROM public.ecr.aws/lambda/provided:al2023

COPY --from=build /app/main ./main
ENTRYPOINT [ "./main" ]
