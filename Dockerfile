# Dedicated GoLang image
FROM golang:1.23-alpine

# Establishing internal workdir
WORKDIR /otteMultiplayerService

# Copy go.mod and go.sum files to internal workdir
COPY go.mod go.sum ./

# Updates go.mod and go.sum files based on project usage
RUN go mod tidy
# Verifies dependencies
RUN go mod verify
# Downloads dependencies
RUN go mod download

# Copy the entire project to internal workdir
COPY . .
# What the executable will be known as when containerized
ARG _containerAlias="otteMultiplayerService"

# Compile - builds executable
RUN ["go", "build", "-o", "$_containerAlias", "./src"]

# Run the executable
CMD [ "./$_containerAlias", "--prod" ]