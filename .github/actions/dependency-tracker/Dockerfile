FROM golang:1.15

WORKDIR /dependency-tracker
COPY . .
RUN go install .
ENTRYPOINT ["dependency-tracker"]
